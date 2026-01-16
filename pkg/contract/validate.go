package contract

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationError represents a single validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no errors"
	}
	var b strings.Builder
	b.WriteString("contract validation failed:\n")
	for _, err := range e {
		b.WriteString("  - ")
		b.WriteString(err.Error())
		b.WriteString("\n")
	}
	return b.String()
}

// Validate checks that the contract is well-formed.
// Returns nil if valid, or ValidationErrors if there are problems.
func Validate(c *Contract) error {
	var errs ValidationErrors

	if c.Version == "" {
		errs = append(errs, ValidationError{
			Field:   "version",
			Message: "required field is empty",
		})
	}

	if c.Name == "" {
		errs = append(errs, ValidationError{
			Field:   "name",
			Message: "required field is empty",
		})
	}

	errs = append(errs, validateRequiredFiles(c.RequiredFiles)...)
	errs = append(errs, validateRequiredSymbols(c.RequiredSymbols)...)
	errs = append(errs, validateForbiddenPatterns(c.ForbiddenPatterns)...)
	errs = append(errs, validateMockSignatures(c.MockSignatures)...)
	errs = append(errs, validateComplexity(c.Complexity)...)
	errs = append(errs, validateRequiredTests(c.RequiredTests)...)
	errs = append(errs, validateCoverageThreshold(c.CoverageThreshold)...)

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateRequiredFiles(files []RequiredFile) ValidationErrors {
	var errs ValidationErrors
	for i, f := range files {
		if f.Path == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("required_files[%d].path", i),
				Message: "path cannot be empty",
			})
		}
	}
	return errs
}

func validateRequiredSymbols(symbols []RequiredSymbol) ValidationErrors {
	var errs ValidationErrors
	validKinds := map[SymbolKind]bool{
		SymbolFunction: true,
		SymbolMethod:   true,
		SymbolType:     true,
		SymbolConst:    true,
	}

	for i, s := range symbols {
		if s.Name == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("required_symbols[%d].name", i),
				Message: "name cannot be empty",
			})
		}
		if s.Kind == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("required_symbols[%d].kind", i),
				Message: "kind cannot be empty",
			})
		} else if !validKinds[s.Kind] {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("required_symbols[%d].kind", i),
				Message: fmt.Sprintf("invalid kind %q, must be one of: function, method, type, const", s.Kind),
			})
		}
		if s.File == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("required_symbols[%d].file", i),
				Message: "file cannot be empty",
			})
		}
	}
	return errs
}

func validateForbiddenPatterns(patterns []ForbiddenPattern) ValidationErrors {
	var errs ValidationErrors
	for i, p := range patterns {
		if p.Pattern == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("forbidden_patterns[%d].pattern", i),
				Message: "pattern cannot be empty",
			})
			continue
		}
		if _, err := regexp.Compile(p.Pattern); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("forbidden_patterns[%d].pattern", i),
				Message: fmt.Sprintf("invalid regex: %v", err),
			})
		}
	}
	return errs
}

func validateMockSignatures(cfg *MockSignaturesConfig) ValidationErrors {
	var errs ValidationErrors
	if cfg == nil {
		return errs
	}
	for i, s := range cfg.Patterns {
		if s.Pattern == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("mock_signatures.patterns[%d].pattern", i),
				Message: "pattern cannot be empty",
			})
			continue
		}
		if _, err := regexp.Compile(s.Pattern); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("mock_signatures.patterns[%d].pattern", i),
				Message: fmt.Sprintf("invalid regex: %v", err),
			})
		}
	}
	if cfg.TestFileSeverity != "" && cfg.TestFileSeverity != "info" && cfg.TestFileSeverity != "warning" {
		errs = append(errs, ValidationError{
			Field:   "mock_signatures.test_file_severity",
			Message: fmt.Sprintf("invalid severity %q, must be one of: info, warning", cfg.TestFileSeverity),
		})
	}
	return errs
}

func validateComplexity(reqs []ComplexityRequirement) ValidationErrors {
	var errs ValidationErrors
	for i, r := range reqs {
		if r.Symbol == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("complexity[%d].symbol", i),
				Message: "symbol cannot be empty",
			})
		}
		if r.MinComplexity < 1 {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("complexity[%d].min_complexity", i),
				Message: "min_complexity must be at least 1",
			})
		}
	}
	return errs
}

func validateRequiredTests(tests []RequiredTest) ValidationErrors {
	var errs ValidationErrors
	for i, t := range tests {
		if t.Name == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("required_tests[%d].name", i),
				Message: "name cannot be empty",
			})
		}
	}
	return errs
}

func validateCoverageThreshold(threshold *float64) ValidationErrors {
	var errs ValidationErrors
	if threshold != nil {
		if *threshold < 0 || *threshold > 100 {
			errs = append(errs, ValidationError{
				Field:   "coverage_threshold",
				Message: "must be between 0 and 100",
			})
		}
	}
	return errs
}
