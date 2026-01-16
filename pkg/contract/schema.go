// Package contract defines the schema for hollowcheck contract files.
package contract

// Contract represents the top-level contract definition.
type Contract struct {
	Version           string                  `yaml:"version"`
	Name              string                  `yaml:"name"`
	Description       string                  `yaml:"description,omitempty"`
	RequiredFiles     []RequiredFile          `yaml:"required_files,omitempty"`
	RequiredSymbols   []RequiredSymbol        `yaml:"required_symbols,omitempty"`
	ForbiddenPatterns []ForbiddenPattern      `yaml:"forbidden_patterns,omitempty"`
	MockSignatures    *MockSignaturesConfig   `yaml:"mock_signatures,omitempty"`
	Complexity        []ComplexityRequirement `yaml:"complexity,omitempty"`
	RequiredTests     []RequiredTest          `yaml:"required_tests,omitempty"`
	CoverageThreshold *float64                `yaml:"coverage_threshold,omitempty"`
}

// RequiredFile specifies a file that must exist.
type RequiredFile struct {
	Path     string `yaml:"path"`
	Required bool   `yaml:"required"`
}

// SymbolKind represents the type of symbol.
type SymbolKind string

const (
	SymbolFunction SymbolKind = "function"
	SymbolMethod   SymbolKind = "method"
	SymbolType     SymbolKind = "type"
	SymbolConst    SymbolKind = "const"
)

// RequiredSymbol specifies a symbol that must be defined.
type RequiredSymbol struct {
	Name string     `yaml:"name"`
	Kind SymbolKind `yaml:"kind"`
	File string     `yaml:"file"`
}

// ForbiddenPattern specifies a regex pattern that must not appear in the code.
type ForbiddenPattern struct {
	Pattern     string `yaml:"pattern"`
	Description string `yaml:"description,omitempty"`
}

// MockSignature specifies a regex pattern identifying mock/placeholder data.
type MockSignature struct {
	Pattern     string `yaml:"pattern"`
	Description string `yaml:"description,omitempty"`
}

// MockSignaturesConfig holds mock signature patterns and their configuration.
type MockSignaturesConfig struct {
	Patterns         []MockSignature `yaml:"patterns,omitempty"`
	SkipTestFiles    *bool           `yaml:"skip_test_files,omitempty"`    // Default: true
	TestFileSeverity string          `yaml:"test_file_severity,omitempty"` // "info", "warning", or "" (skip)
}

// ShouldSkipTestFiles returns whether to skip test files (defaults to true).
func (m *MockSignaturesConfig) ShouldSkipTestFiles() bool {
	if m.SkipTestFiles == nil {
		return true // default
	}
	return *m.SkipTestFiles
}

// GetTestFileSeverity returns the severity to use for test files.
// Returns empty string if test files should be skipped entirely.
func (m *MockSignaturesConfig) GetTestFileSeverity() string {
	if m.ShouldSkipTestFiles() {
		return ""
	}
	if m.TestFileSeverity != "" {
		return m.TestFileSeverity
	}
	return "warning" // default severity if not skipping
}

// ComplexityRequirement specifies minimum cyclomatic complexity for a symbol.
type ComplexityRequirement struct {
	Symbol        string `yaml:"symbol"`
	File          string `yaml:"file,omitempty"`
	MinComplexity int    `yaml:"min_complexity"`
}

// RequiredTest specifies a test function that must exist.
type RequiredTest struct {
	Name string `yaml:"name"`
	File string `yaml:"file,omitempty"`
}
