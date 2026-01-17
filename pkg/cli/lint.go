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
	ContractPath string
	Format       string
	Threshold    int
}

// NewLintCmd creates the lint command.
func NewLintCmd() *cobra.Command {
	opts := &LintOptions{}

	cmd := &cobra.Command{
		Use:   "lint <path>",
		Short: "Check code quality against a contract",
		Long: `Lint scans the specified path for code quality issues defined in the contract.

It checks for:
  - Missing required files
  - Missing required symbols (functions, types, etc.)
  - Forbidden patterns (TODO, FIXME, etc.)
  - Mock data signatures (example.com, fake IDs, etc.)
  - Low cyclomatic complexity (stub implementations)
  - Missing required tests

Exit codes:
  0 - Passed (score <= threshold)
  1 - Failed (score > threshold)
  2 - Error (bad contract, path not found, etc.)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLint(args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ContractPath, "contract", "c", "", "Path to contract YAML file (default: auto-discover)")
	cmd.Flags().StringVarP(&opts.Format, "format", "f", "pretty", "Output format: pretty or json")
	cmd.Flags().IntVarP(&opts.Threshold, "threshold", "t", -1, "Override threshold (default: from contract or 25)")

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
	if opts.Format != "pretty" && opts.Format != "json" {
		return fmt.Errorf("invalid format %q, must be 'pretty' or 'json'", opts.Format)
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

	// Collect files to scan
	var files []string
	if info.IsDir() {
		files, err = collectFiles(absPath)
		if err != nil {
			return fmt.Errorf("failed to collect files: %w", err)
		}
	} else {
		files = []string{absPath}
	}

	// Run detection
	runner := detect.NewRunner(absPath)
	result, err := runner.Run(files, c)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	// Calculate score
	var hollowness score.HollownessScore
	if opts.Threshold >= 0 {
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
	default:
		report.WritePretty(os.Stdout, path, contractPath, Version, result, hollowness)
	}

	// Return error if threshold exceeded (handled specially by main)
	if !hollowness.Passed {
		return ErrThresholdExceeded
	}

	return nil
}

// collectFiles recursively collects all files with supported extensions.
func collectFiles(root string) ([]string, error) {
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
				files = append(files, path)
			}
		}

		return nil
	})

	return files, err
}
