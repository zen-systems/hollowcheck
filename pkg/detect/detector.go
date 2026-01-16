// Package detect provides detection of quality issues in code.
package detect

import (
	"github.com/zen-systems/hollowcheck/pkg/contract"
)

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

// Violation represents a single detected issue.
type Violation struct {
	Rule     string // e.g., "forbidden_pattern", "missing_symbol", "mock_data"
	Message  string
	File     string
	Line     int
	Severity string // "error", "warning"
}

// DetectionResult contains the results of running detection.
type DetectionResult struct {
	Violations []Violation
	Scanned    int // files scanned
}

// Merge combines another result into this one.
func (r *DetectionResult) Merge(other *DetectionResult) {
	if other == nil {
		return
	}
	r.Violations = append(r.Violations, other.Violations...)
	r.Scanned += other.Scanned
}

// AddViolation adds a violation to the result.
func (r *DetectionResult) AddViolation(v Violation) {
	r.Violations = append(r.Violations, v)
}

// HasErrors returns true if there are any error-severity violations.
func (r *DetectionResult) HasErrors() bool {
	for _, v := range r.Violations {
		if v.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Detector is the interface for detection implementations.
type Detector interface {
	Detect(files []string, c *contract.Contract) (*DetectionResult, error)
}

// Runner executes all detection checks against a set of files.
type Runner struct {
	BaseDir string
}

// NewRunner creates a new detection runner.
func NewRunner(baseDir string) *Runner {
	return &Runner{BaseDir: baseDir}
}

// Run executes all detection checks defined in the contract.
func (r *Runner) Run(files []string, c *contract.Contract) (*DetectionResult, error) {
	result := &DetectionResult{}

	// Check required files
	fileResult, err := DetectMissingFiles(r.BaseDir, c.RequiredFiles)
	if err != nil {
		return nil, err
	}
	result.Merge(fileResult)

	// Scan for forbidden patterns
	patternResult, err := DetectForbiddenPatterns(files, c.ForbiddenPatterns)
	if err != nil {
		return nil, err
	}
	result.Merge(patternResult)

	// Scan for mock data signatures
	mockResult, err := DetectMockData(files, c.MockSignatures)
	if err != nil {
		return nil, err
	}
	result.Merge(mockResult)

	// Check required symbols (Go files only)
	symbolResult, err := DetectMissingSymbols(r.BaseDir, files, c.RequiredSymbols)
	if err != nil {
		return nil, err
	}
	result.Merge(symbolResult)

	// Check complexity requirements (Go files only)
	complexityResult, err := DetectLowComplexity(r.BaseDir, files, c.Complexity)
	if err != nil {
		return nil, err
	}
	result.Merge(complexityResult)

	// Check required tests
	testResult, err := DetectMissingTests(r.BaseDir, files, c.RequiredTests)
	if err != nil {
		return nil, err
	}
	result.Merge(testResult)

	return result, nil
}
