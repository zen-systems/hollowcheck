package cli

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

//go:embed templates/*.yaml
var templatesFS embed.FS

// Template represents a contract template.
type Template struct {
	Name        string
	Description string
	Filename    string
}

// Available templates
var templates = []Template{
	{
		Name:        "minimal",
		Description: "Bare minimum quality gate - no stubs, no obvious mocks",
		Filename:    "minimal.yaml",
	},
	{
		Name:        "crud-endpoint",
		Description: "REST API endpoint with CRUD database operations",
		Filename:    "crud-endpoint.yaml",
	},
	{
		Name:        "cli-tool",
		Description: "Command-line tool with argument parsing and subcommands",
		Filename:    "cli-tool.yaml",
	},
	{
		Name:        "client-sdk",
		Description: "API client SDK with authentication, retry, and error handling",
		Filename:    "client-sdk.yaml",
	},
	{
		Name:        "worker",
		Description: "Background job processor with graceful shutdown",
		Filename:    "worker.yaml",
	},
}

// InitOptions holds the options for the init command.
type InitOptions struct {
	Template string
	Output   string
	List     bool
}

// NewInitCmd creates the init command.
func NewInitCmd() *cobra.Command {
	opts := &InitOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new hollowcheck contract from a template",
		Long: `Initialize a new hollowcheck contract file from a template.

With no arguments, creates a minimal contract (hollowcheck.yaml) in the
current directory. Use --template to select a different starting point.

Available templates:
  minimal       Bare minimum quality gate (default)
  crud-endpoint REST API with database operations
  cli-tool      Command-line tool with args
  client-sdk    API client with auth/retry
  worker        Background job processor

Examples:
  hollowcheck init
  hollowcheck init --template client-sdk
  hollowcheck init --template worker --output contracts/worker.yaml
  hollowcheck init --list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Template, "template", "t", "minimal", "Template to use")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "hollowcheck.yaml", "Output file path")
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List available templates")

	return cmd
}

func runInit(opts *InitOptions) error {
	// List mode
	if opts.List {
		return listTemplates()
	}

	// Find template
	var tmpl *Template
	for i := range templates {
		if templates[i].Name == opts.Template {
			tmpl = &templates[i]
			break
		}
	}
	if tmpl == nil {
		fmt.Fprintf(os.Stderr, "Error: unknown template %q\n", opts.Template)
		fmt.Fprintln(os.Stderr, "Run 'hollowcheck init --list' to see available templates")
		os.Exit(ExitError)
	}

	// Check if output already exists
	if _, err := os.Stat(opts.Output); err == nil {
		fmt.Fprintf(os.Stderr, "Error: file already exists: %s\n", opts.Output)
		fmt.Fprintln(os.Stderr, "Remove it or use --output to specify a different path")
		os.Exit(ExitError)
	}

	// Read template content
	content, err := templatesFS.ReadFile("templates/" + tmpl.Filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read template: %v\n", err)
		os.Exit(ExitError)
	}

	// Create output directory if needed
	dir := filepath.Dir(opts.Output)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create directory: %v\n", err)
			os.Exit(ExitError)
		}
	}

	// Write contract file
	if err := os.WriteFile(opts.Output, content, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write contract: %v\n", err)
		os.Exit(ExitError)
	}

	// Success message
	fmt.Printf("Created %s from template '%s'\n", opts.Output, tmpl.Name)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Edit %s to customize for your project\n", opts.Output)
	fmt.Printf("  2. Run: hollowcheck lint . --contract %s\n", opts.Output)

	return nil
}

func listTemplates() error {
	fmt.Println("Available templates:")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, t := range templates {
		name := t.Name
		if t.Name == "minimal" {
			name += " (default)"
		}
		fmt.Fprintf(w, "  %s\t%s\n", name, t.Description)
	}
	w.Flush()

	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  hollowcheck init --template <name>")

	return nil
}

// GetTemplateNames returns a list of available template names.
func GetTemplateNames() []string {
	names := make([]string, len(templates))
	for i, t := range templates {
		names[i] = t.Name
	}
	return names
}

// GetTemplateContent returns the content of a template by name.
func GetTemplateContent(name string) ([]byte, error) {
	for _, t := range templates {
		if t.Name == name {
			return templatesFS.ReadFile("templates/" + t.Filename)
		}
	}
	return nil, fmt.Errorf("template not found: %s", name)
}

// ValidateTemplateName checks if a template name is valid.
func ValidateTemplateName(name string) bool {
	name = strings.ToLower(name)
	for _, t := range templates {
		if t.Name == name {
			return true
		}
	}
	return false
}
