// Command hollowcheck is an AI output quality gate system.
package main

import (
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
		Version: cli.Version,
	}

	rootCmd.AddCommand(cli.NewLintCmd())
	rootCmd.AddCommand(cli.NewInitCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitError)
	}
}
