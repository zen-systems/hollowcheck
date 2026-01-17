// Package cli implements the hollowcheck command-line interface.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zen-systems/hollowcheck/pkg/contract"
	"github.com/zen-systems/hollowcheck/pkg/detect"
	"github.com/zen-systems/hollowcheck/pkg/detect/prose"
	"github.com/zen-systems/hollowcheck/pkg/git"
	"github.com/zen-systems/hollowcheck/pkg/parser"
	"github.com/zen-systems/hollowcheck/pkg/report"
	"github.com/zen-systems/hollowcheck/pkg/score"
)

// Exit codes
const (
	ExitSuccess = 0
	ExitFailed  = 1
	ExitError   = 2
)

// Default contract file names to search for
var defaultContractNames = []string{
	"hollowcheck.yaml",
	"hollow.yaml",
	".hollowcheck.yaml",
}

// ErrThresholdExceeded is returned when the hollowness score exceeds the threshold.
// This is a "soft" failure - the tool ran successfully but the code didn't pass.
var ErrThresholdExceeded = errors.New("hollowness threshold exceeded")

// Version is set at build time
var Version = "0.1.0"

// LintOptions holds the options for the lint command.
type LintOptions struct {
	ContractPath   string
	Format         string
	Threshold      int
	Mode           string // "code" (default) or "prose"
	ShowSuppressed bool   // Show suppressed violations in output
	DiffRef        string // Git ref for diff mode (only check changed files)
	BaselineRef    string // Git ref for baseline mode (only fail on new violations)
}

// NewLintCmd creates the lint command.
func NewLintCmd() *cobra.Command {
	opts := &LintOptions{}

	cmd := &cobra.Command{
		Use:   "lint <path>",
		Short: "Check code or prose quality against a contract",
		Long: `Lint scans the specified path for quality issues defined in the contract.

Code mode (default) checks for:
  - Missing required files
  - Missing required symbols (functions, types, etc.)
  - Forbidden patterns (TODO, FIXME, etc.)
  - Mock data signatures (example.com, fake IDs, etc.)
  - Low cyclomatic complexity (stub implementations)
  - Missing required tests

Prose mode (--mode prose) checks for:
  - Filler phrases that add no meaning
  - Weasel words and vague language
  - Low information density
  - Repetitive sentence structures
  - Middle sag (weak middle sections)

Incremental modes:
  --diff <ref>      Only check files changed since git ref
  --baseline <ref>  Fail only if hollowness increased vs baseline

Exit codes:
  0 - Passed (score <= threshold)
  1 - Failed (score > threshold, or new violations in baseline mode)
  2 - Error (bad contract, path not found, etc.)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLint(args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ContractPath, "contract", "c", "", "Path to contract YAML file (default: auto-discover)")
	cmd.Flags().StringVarP(&opts.Format, "format", "f", "pretty", "Output format: pretty, json, or sarif")
	cmd.Flags().IntVarP(&opts.Threshold, "threshold", "t", -1, "Override threshold (default: from contract or 25)")
	cmd.Flags().StringVarP(&opts.Mode, "mode", "m", "", "Analysis mode: code (default) or prose")
	cmd.Flags().BoolVar(&opts.ShowSuppressed, "show-suppressed", false, "Show details of suppressed violations")
	cmd.Flags().StringVar(&opts.DiffRef, "diff", "", "Only check files changed since git ref (e.g., main, HEAD~1)")
	cmd.Flags().StringVar(&opts.BaselineRef, "baseline", "", "Fail only if new violations vs git ref (e.g., origin/main)")

	return cmd
}

// discoverContract looks for a contract file in the current directory.
// Returns the path to the first matching file, or an error if none found.
func discoverContract() (string, error) {
	for _, name := range defaultContractNames {
		if _, err := os.Stat(name); err == nil {
			return name, nil
		}
	}
	return "", fmt.Errorf("no contract file found (looked for %s)", strings.Join(defaultContractNames, ", "))
}

func runLint(path string, opts *LintOptions) error {
	// Validate format
	if opts.Format != "pretty" && opts.Format != "json" && opts.Format != "sarif" {
		return fmt.Errorf("invalid format %q, must be 'pretty', 'json', or 'sarif'", opts.Format)
	}

	// Validate mode
	mode := opts.Mode
	if mode == "" {
		mode = "code" // default
	}
	if mode != "code" && mode != "prose" {
		return fmt.Errorf("invalid mode %q, must be 'code' or 'prose'", mode)
	}

	// Validate --diff and --baseline are mutually exclusive
	if opts.DiffRef != "" && opts.BaselineRef != "" {
		return fmt.Errorf("--diff and --baseline are mutually exclusive")
	}

	// Discover contract if not specified
	contractPath := opts.ContractPath
	if contractPath == "" {
		discovered, err := discoverContract()
		if err != nil {
			return err
		}
		contractPath = discovered
	}

	// Parse contract
	c, err := contract.ParseFile(contractPath)
	if err != nil {
		return fmt.Errorf("failed to parse contract: %w", err)
	}

	// Override mode from contract if set and not overridden by CLI
	if opts.Mode == "" && c.Mode != "" {
		mode = c.Mode
	}

	// Validate contract
	if err := contract.Validate(c); err != nil {
		return fmt.Errorf("invalid contract: %w", err)
	}

	// Resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("cannot access path: %w", err)
	}

	// Validate git ref if --diff or --baseline is specified
	var repoRoot string
	if opts.DiffRef != "" || opts.BaselineRef != "" {
		repoRoot, err = git.GetRepoRoot(absPath)
		if err != nil {
			return fmt.Errorf("--diff and --baseline require a git repository: %w", err)
		}

		ref := opts.DiffRef
		if ref == "" {
			ref = opts.BaselineRef
		}
		if !git.RefExists(ref, repoRoot) {
			return fmt.Errorf("git ref %q does not exist", ref)
		}
	}

	// Collect files to scan (based on mode)
	var files []string
	if info.IsDir() {
		if mode == "prose" {
			files, err = collectProseFiles(absPath, c.Prose)
		} else {
			files, err = collectFiles(absPath, c.ShouldIncludeTestFiles())
		}
		if err != nil {
			return fmt.Errorf("failed to collect files: %w", err)
		}
	} else {
		files = []string{absPath}
	}

	// In diff mode, filter to only changed files
	if opts.DiffRef != "" {
		files, err = filterChangedFiles(files, opts.DiffRef, absPath)
		if err != nil {
			return fmt.Errorf("failed to get changed files: %w", err)
		}
		if len(files) == 0 {
			// No changed files - report success
			fmt.Fprintf(os.Stderr, "No files changed since %s\n", opts.DiffRef)
			return nil
		}
	}

	// Run detection based on mode
	var result *detect.DetectionResult
	if mode == "prose" {
		detector := prose.NewDetector(c.Prose)
		result, err = detector.Detect(files, c)
		if err != nil {
			return fmt.Errorf("prose detection failed: %w", err)
		}
	} else {
		runner := detect.NewRunner(absPath)
		result, err = runner.Run(files, c)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}
	}

	// In baseline mode, compare against baseline
	if opts.BaselineRef != "" {
		result, err = compareToBaseline(result, opts.BaselineRef, absPath, repoRoot, c, mode)
		if err != nil {
			return fmt.Errorf("baseline comparison failed: %w", err)
		}
	}

	// Calculate score
	var hollowness score.HollownessScore
	if result.IsBaselineMode() {
		// In baseline mode, score based on new violations only
		hollowness = score.CalculateForNewViolations(result, opts.Threshold)
	} else if opts.Threshold >= 0 {
		hollowness = score.CalculateWithThreshold(result, opts.Threshold)
	} else {
		hollowness = score.Calculate(result, c)
	}

	// Output results
	switch opts.Format {
	case "json":
		if err := report.WriteJSON(os.Stdout, path, contractPath, Version, result, hollowness); err != nil {
			return fmt.Errorf("failed to write JSON: %w", err)
		}
	case "sarif":
		if err := report.WriteSARIF(os.Stdout, absPath, Version, result); err != nil {
			return fmt.Errorf("failed to write SARIF: %w", err)
		}
	default:
		prettyOpts := report.PrettyOptions{
			ShowSuppressed: opts.ShowSuppressed,
		}
		report.WritePrettyWithOptions(os.Stdout, path, contractPath, Version, result, hollowness, prettyOpts)
	}

	// Return error if threshold exceeded (handled specially by main)
	if !hollowness.Passed {
		return ErrThresholdExceeded
	}

	return nil
}

// filterChangedFiles filters the file list to only include files changed since the git ref.
func filterChangedFiles(files []string, ref string, basePath string) ([]string, error) {
	// Build supported extensions map
	supported := make(map[string]bool)
	supported[".go"] = true
	for _, ext := range parser.SupportedExtensions() {
		supported[ext] = true
	}

	changedFiles, err := git.GetChangedFiles(ref, basePath, supported)
	if err != nil {
		return nil, err
	}

	// Create a set of changed files for fast lookup
	changedSet := make(map[string]bool)
	for _, f := range changedFiles {
		changedSet[f] = true
	}

	// Filter the original file list
	var result []string
	for _, f := range files {
		if changedSet[f] {
			result = append(result, f)
		}
	}

	return result, nil
}

// compareToBaseline compares current violations against a baseline ref.
// Returns a new DetectionResult with NewViolations populated.
func compareToBaseline(current *detect.DetectionResult, baselineRef string, absPath string, repoRoot string, c *contract.Contract, mode string) (*detect.DetectionResult, error) {
	// Build a set of current violations for comparison
	currentViolations := make(map[string]detect.Violation)
	for _, v := range current.Violations {
		key := detect.ViolationKey(v)
		currentViolations[key] = v
	}

	// Get baseline violations by running detection on files at the baseline ref
	// For each file with violations, check if the violation existed at baseline
	baselineViolations := make(map[string]bool)

	// Get unique files with violations
	violationFiles := make(map[string]bool)
	for _, v := range current.Violations {
		violationFiles[v.File] = true
	}

	// For each file with violations, get its content at baseline and run detection
	for filePath := range violationFiles {
		baselineContent, err := git.GetFileAtRef(baselineRef, filePath, repoRoot)
		if err != nil {
			// Error getting file - skip
			continue
		}
		if baselineContent == nil {
			// File didn't exist at baseline - all violations in this file are new
			continue
		}

		// Run detection on baseline content
		baselineResult, err := runDetectionOnContent(baselineContent, filePath, c, mode)
		if err != nil {
			continue
		}

		// Add baseline violations to the set
		for _, v := range baselineResult.Violations {
			key := detect.ViolationKey(v)
			baselineViolations[key] = true
		}
	}

	// Find new violations (in current but not in baseline)
	var newViolations []detect.Violation
	for key, v := range currentViolations {
		if !baselineViolations[key] {
			newViolations = append(newViolations, v)
		}
	}

	// Return updated result
	return &detect.DetectionResult{
		Violations:    current.Violations,
		Suppressed:    current.Suppressed,
		NewViolations: newViolations,
		Scanned:       current.Scanned,
		BaselineRef:   baselineRef,
	}, nil
}

// runDetectionOnContent runs detection on file content (for baseline comparison).
func runDetectionOnContent(content []byte, filePath string, c *contract.Contract, mode string) (*detect.DetectionResult, error) {
	result := &detect.DetectionResult{}

	if mode == "prose" {
		// For prose mode, we'd need to run prose detection
		// This is a simplified implementation
		return result, nil
	}

	// Detect forbidden patterns
	patternResult := detectPatternsInContent(content, filePath, c.ForbiddenPatterns)
	result.Merge(patternResult)

	// Detect mock data
	mockResult := detectMockDataInContent(content, filePath, c.MockSignatures)
	result.Merge(mockResult)

	return result, nil
}

// detectPatternsInContent detects forbidden patterns in content.
func detectPatternsInContent(content []byte, filePath string, patterns []contract.ForbiddenPattern) *detect.DetectionResult {
	result := &detect.DetectionResult{}

	if len(patterns) == 0 {
		return result
	}

	lines := strings.Split(string(content), "\n")
	for _, p := range patterns {
		for lineNum, line := range lines {
			if strings.Contains(line, p.Pattern) {
				msg := fmt.Sprintf("forbidden pattern %q found", p.Pattern)
				if p.Description != "" {
					msg = fmt.Sprintf("%s: %s", msg, p.Description)
				}
				result.AddViolation(detect.Violation{
					Rule:     detect.RuleForbiddenPattern,
					Message:  msg,
					File:     filePath,
					Line:     lineNum + 1,
					Severity: detect.SeverityError,
				})
			}
		}
	}

	return result
}

// detectMockDataInContent detects mock data patterns in content.
func detectMockDataInContent(content []byte, filePath string, config *contract.MockSignaturesConfig) *detect.DetectionResult {
	result := &detect.DetectionResult{}

	if config == nil || len(config.Patterns) == 0 {
		return result
	}

	lines := strings.Split(string(content), "\n")
	for _, p := range config.Patterns {
		for lineNum, line := range lines {
			if strings.Contains(line, p.Pattern) {
				msg := fmt.Sprintf("mock data pattern %q found", p.Pattern)
				if p.Description != "" {
					msg = fmt.Sprintf("%s: %s", msg, p.Description)
				}
				result.AddViolation(detect.Violation{
					Rule:     detect.RuleMockData,
					Message:  msg,
					File:     filePath,
					Line:     lineNum + 1,
					Severity: detect.SeverityWarning,
				})
			}
		}
	}

	return result
}

// collectFiles recursively collects all files with supported extensions.
// If includeTestFiles is false, files ending in _test.go are excluded.
func collectFiles(root string, includeTestFiles bool) ([]string, error) {
	// Build set of supported extensions
	supported := make(map[string]bool)
	supported[".go"] = true // Always support Go
	for _, ext := range parser.SupportedExtensions() {
		supported[ext] = true
	}

	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip vendor and node_modules directories
		if info.IsDir() && (info.Name() == "vendor" || info.Name() == "node_modules") {
			return filepath.SkipDir
		}

		// Collect files with supported extensions
		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			if supported[ext] {
				// Skip test files unless explicitly included
				if !includeTestFiles && strings.HasSuffix(info.Name(), "_test.go") {
					return nil
				}
				files = append(files, path)
			}
		}

		return nil
	})

	return files, err
}

// collectProseFiles recursively collects all prose files.
func collectProseFiles(root string, cfg *contract.ProseConfig) ([]string, error) {
	// Default prose extensions
	proseExts := map[string]bool{
		".md":   true,
		".txt":  true,
		".rst":  true,
		".adoc": true,
		".tex":  true,
		".html": true,
	}

	// Override with config if provided
	if cfg != nil && len(cfg.Extensions) > 0 {
		proseExts = make(map[string]bool)
		for _, ext := range cfg.Extensions {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			proseExts[ext] = true
		}
	}

	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip vendor and node_modules directories
		if info.IsDir() && (info.Name() == "vendor" || info.Name() == "node_modules") {
			return filepath.SkipDir
		}

		// Collect files with prose extensions
		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			if proseExts[ext] {
				files = append(files, path)
			}
		}

		return nil
	})

	return files, err
}
