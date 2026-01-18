// Package analysis provides pluggable code analysis engines.
//
// This package implements the Strategy Pattern with build tags to support
// two analysis engines:
//   - Smart Engine (default): Uses Tree-sitter for AST-based analysis (requires CGO)
//   - Simple Engine (fallback): Uses regex-based analysis (pure Go, CGO_ENABLED=0)
//
// Build with -tags simple to use the regex-based engine:
//
//	go build -tags simple ./...
package analysis

import (
	"github.com/zen-systems/hollowcheck/pkg/contract"
)

// Result contains the analysis results.
type Result struct {
	Violations []Violation
	Scanned    int // Number of files scanned
}

// Violation represents a single detected issue.
type Violation struct {
	Rule     string // e.g., "forbidden_pattern", "missing_symbol", "low_complexity"
	Message  string
	File     string
	Line     int
	Column   int    // Column position of the match (1-indexed, 0 if unknown)
	Match    string // The actual matched content
	Context  string // The full line containing the match (for display)
	Severity string // "error", "warning", "info"
}

// Severity levels for violations.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// Rule names for different violation types.
const (
	RuleForbiddenPattern = "forbidden_pattern"
	RuleMockData         = "mock_data"
	RuleMissingFile      = "missing_file"
	RuleMissingSymbol    = "missing_symbol"
	RuleLowComplexity    = "low_complexity"
	RuleMissingTest      = "missing_test"
)

// Analyzer defines the interface for code analysis engines.
// Implementations must be able to analyze source files against a contract
// and return any violations found.
type Analyzer interface {
	// Analyze scans the provided files against the contract rules.
	// The files map contains file paths as keys and file contents as values.
	// Returns a Result containing all violations found, or an error if analysis fails.
	Analyze(files map[string]string, contract *contract.Contract) (*Result, error)

	// Name returns the name of the analyzer engine (e.g., "smart", "simple").
	Name() string
}

// AddViolation is a helper method to add a violation to the result.
func (r *Result) AddViolation(v Violation) {
	r.Violations = append(r.Violations, v)
}

// Merge combines another result into this one.
func (r *Result) Merge(other *Result) {
	if other == nil {
		return
	}
	r.Violations = append(r.Violations, other.Violations...)
	r.Scanned += other.Scanned
}

// HasErrors returns true if there are any error-severity violations.
func (r *Result) HasErrors() bool {
	for _, v := range r.Violations {
		if v.Severity == SeverityError {
			return true
		}
	}
	return false
}
