package detect

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/zen-systems/hollowcheck/pkg/contract"
)

// compiledMockSignature holds a pre-compiled regex with its metadata.
type compiledMockSignature struct {
	regex       *regexp.Regexp
	description string
}

// isTestFile returns true if the file path ends with _test.go.
func isTestFile(filePath string) bool {
	return strings.HasSuffix(filePath, "_test.go")
}

// DetectMockData scans files for mock data signatures defined in the contract.
func DetectMockData(files []string, cfg *contract.MockSignaturesConfig) (*DetectionResult, error) {
	result := &DetectionResult{}

	if cfg == nil || len(cfg.Patterns) == 0 {
		return result, nil
	}

	// Pre-compile all patterns
	compiled := make([]compiledMockSignature, 0, len(cfg.Patterns))
	for _, s := range cfg.Patterns {
		re, err := regexp.Compile(s.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling mock signature %q: %w", s.Pattern, err)
		}
		compiled = append(compiled, compiledMockSignature{
			regex:       re,
			description: s.Description,
		})
	}

	// Determine test file handling
	skipTestFiles := cfg.ShouldSkipTestFiles()
	testFileSeverity := cfg.GetTestFileSeverity()

	// Scan each file
	for _, file := range files {
		isTest := isTestFile(file)

		// Skip test files if configured
		if isTest && skipTestFiles {
			result.Scanned++
			continue
		}

		// Determine severity for this file
		severity := SeverityWarning
		if isTest && testFileSeverity != "" {
			severity = testFileSeverity
		}

		violations, err := scanFileForMocks(file, compiled, severity)
		if err != nil {
			return nil, fmt.Errorf("scanning %s for mocks: %w", file, err)
		}
		result.Violations = append(result.Violations, violations...)
		result.Scanned++
	}

	return result, nil
}

// scanFileForMocks scans a single file for mock data signatures.
func scanFileForMocks(filePath string, signatures []compiledMockSignature, severity string) ([]Violation, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var violations []Violation
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, s := range signatures {
			if s.regex.MatchString(line) {
				msg := fmt.Sprintf("mock data signature %q found", s.regex.String())
				if s.description != "" {
					msg = fmt.Sprintf("%s: %s", msg, s.description)
				}
				violations = append(violations, Violation{
					Rule:     RuleMockData,
					Message:  msg,
					File:     filePath,
					Line:     lineNum,
					Severity: severity,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return violations, nil
}
