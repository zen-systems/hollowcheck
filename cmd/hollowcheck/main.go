// Command hollowcheck is an AI output quality gate system.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/zen-systems/hollowcheck/pkg/cli"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "hollowcheck",
		Short: "AI output quality gate system",
		Long: `Hollowcheck validates AI-generated code against quality contracts.

It detects "hollow" code - implementations that look complete but lack
real functionality: stub implementations, placeholder data, TODO markers,
and functions with suspiciously low complexity.`,
		Version:       cli.Version,
		SilenceErrors: true, // We handle error output ourselves
		SilenceUsage:  true, // Don't print usage on errors
	}

	rootCmd.AddCommand(cli.NewLintCmd())
	rootCmd.AddCommand(cli.NewInitCmd())

	if err := rootCmd.Execute(); err != nil {
		// Threshold exceeded is a "soft" failure - exit 1, no extra message
		// (the report was already printed)
		if errors.Is(err, cli.ErrThresholdExceeded) {
			os.Exit(cli.ExitFailed)
		}

		// All other errors - print and exit 2
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.ExitError)
	}
}
