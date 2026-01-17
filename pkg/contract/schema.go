// Package contract defines the schema for hollowcheck contract files.
package contract

// Contract represents the top-level contract definition.
type Contract struct {
	Version           string                  `yaml:"version"`
	Name              string                  `yaml:"name"`
	Description       string                  `yaml:"description,omitempty"`
	Mode              string                  `yaml:"mode,omitempty"` // "code" (default) or "prose"
	IncludeTestFiles  *bool                   `yaml:"include_test_files,omitempty"` // Default: false
	RequiredFiles     []RequiredFile          `yaml:"required_files,omitempty"`
	RequiredSymbols   []RequiredSymbol        `yaml:"required_symbols,omitempty"`
	ForbiddenPatterns []ForbiddenPattern      `yaml:"forbidden_patterns,omitempty"`
	MockSignatures    *MockSignaturesConfig   `yaml:"mock_signatures,omitempty"`
	Complexity        []ComplexityRequirement `yaml:"complexity,omitempty"`
	RequiredTests     []RequiredTest          `yaml:"required_tests,omitempty"`
	CoverageThreshold *float64                `yaml:"coverage_threshold,omitempty"`
	Prose             *ProseConfig            `yaml:"prose,omitempty"` // Prose analysis configuration
}

// ShouldIncludeTestFiles returns whether to include test files (defaults to false).
func (c *Contract) ShouldIncludeTestFiles() bool {
	if c.IncludeTestFiles == nil {
		return false // default: exclude test files
	}
	return *c.IncludeTestFiles
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

// ProseConfig holds configuration for prose analysis.
type ProseConfig struct {
	// Extensions specifies which file extensions to analyze (default: .md, .txt, .rst, .adoc)
	Extensions []string `yaml:"extensions,omitempty"`

	// FillerThreshold is the max number of fillers before reporting violations
	FillerThreshold int `yaml:"filler_threshold,omitempty"`

	// WeaselThreshold is the max number of weasel words before reporting violations
	WeaselThreshold int `yaml:"weasel_threshold,omitempty"`

	// Weights for different prose issues in scoring
	Weights *ProseWeightsConfig `yaml:"weights,omitempty"`

	// Density configuration for information density analysis
	Density *ProseDensityConfig `yaml:"density,omitempty"`

	// Custom filler patterns to detect (in addition to built-in)
	CustomFillers []ProsePattern `yaml:"custom_fillers,omitempty"`

	// Custom weasel patterns to detect (in addition to built-in)
	CustomWeasels []ProsePattern `yaml:"custom_weasels,omitempty"`

	// Patterns to ignore (regex patterns that should not trigger violations)
	IgnorePatterns []string `yaml:"ignore_patterns,omitempty"`
}

// ProseWeightsConfig specifies how different prose issues are weighted.
type ProseWeightsConfig struct {
	Filler    float64 `yaml:"filler,omitempty"`    // Weight for filler phrases (default: 0.25)
	Weasel    float64 `yaml:"weasel,omitempty"`    // Weight for weasel words (default: 0.25)
	Density   float64 `yaml:"density,omitempty"`   // Weight for density issues (default: 0.25)
	Structure float64 `yaml:"structure,omitempty"` // Weight for structure issues (default: 0.25)
}

// ProseDensityConfig specifies configuration for density analysis.
type ProseDensityConfig struct {
	MinSectionWords int     `yaml:"min_section_words,omitempty"` // Min words for section analysis (default: 20)
	LowThreshold    float64 `yaml:"low_threshold,omitempty"`     // Below this is "low density" (default: 0.3)
	HighThreshold   float64 `yaml:"high_threshold,omitempty"`    // Above this is "high density" (default: 0.8)
}

// ProsePattern defines a custom pattern for prose analysis.
type ProsePattern struct {
	Pattern     string  `yaml:"pattern"`               // Regex pattern
	Description string  `yaml:"description,omitempty"` // Human-readable description
	Weight      float64 `yaml:"weight,omitempty"`      // Scoring weight (default: 1.0)
}
