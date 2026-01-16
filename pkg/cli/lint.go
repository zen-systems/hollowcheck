// Package cli implements the hollowcheck command-line interface.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zen-systems/hollowcheck/pkg/contract"
	"github.com/zen-systems/hollowcheck/pkg/detect"
	"github.com/zen-systems/hollowcheck/pkg/report"
	"github.com/zen-systems/hollowcheck/pkg/score"
)

// Exit codes
const (
	ExitSuccess = 0
	ExitFailed  = 1
	ExitError   = 2
)

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

	cmd.Flags().StringVarP(&opts.ContractPath, "contract", "c", "", "Path to contract YAML file (required)")
	cmd.Flags().StringVarP(&opts.Format, "format", "f", "pretty", "Output format: pretty or json")
	cmd.Flags().IntVarP(&opts.Threshold, "threshold", "t", -1, "Override threshold (default: from contract or 25)")

	cmd.MarkFlagRequired("contract")

	return cmd
}

func runLint(path string, opts *LintOptions) error {
	// Validate format
	if opts.Format != "pretty" && opts.Format != "json" {
		fmt.Fprintf(os.Stderr, "Error: invalid format %q, must be 'pretty' or 'json'\n", opts.Format)
		os.Exit(ExitError)
	}

	// Parse contract
	c, err := contract.ParseFile(opts.ContractPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to parse contract: %v\n", err)
		os.Exit(ExitError)
	}

	// Validate contract
	if err := contract.Validate(c); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid contract: %v\n", err)
		os.Exit(ExitError)
	}

	// Resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
		os.Exit(ExitError)
	}

	// Check path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: path does not exist: %s\n", path)
		} else {
			fmt.Fprintf(os.Stderr, "Error: cannot access path: %v\n", err)
		}
		os.Exit(ExitError)
	}

	// Collect files to scan
	var files []string
	if info.IsDir() {
		files, err = collectGoFiles(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to collect files: %v\n", err)
			os.Exit(ExitError)
		}
	} else {
		files = []string{absPath}
	}

	// Run detection
	runner := detect.NewRunner(absPath)
	result, err := runner.Run(files, c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: detection failed: %v\n", err)
		os.Exit(ExitError)
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
		if err := report.WriteJSON(os.Stdout, path, opts.ContractPath, Version, result, hollowness); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to write JSON: %v\n", err)
			os.Exit(ExitError)
		}
	default:
		report.WritePretty(os.Stdout, path, opts.ContractPath, Version, result, hollowness)
	}

	// Exit with appropriate code
	if hollowness.Passed {
		os.Exit(ExitSuccess)
	}
	os.Exit(ExitFailed)

	return nil
}

// collectGoFiles recursively collects all .go files in a directory.
func collectGoFiles(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip vendor directory
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}

		// Collect .go files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
