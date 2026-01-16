package detect

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/zen-systems/hollowcheck/pkg/contract"
)

// compiledPattern holds a pre-compiled regex with its metadata.
type compiledPattern struct {
	regex       *regexp.Regexp
	description string
}

// DetectForbiddenPatterns scans files for forbidden patterns defined in the contract.
func DetectForbiddenPatterns(files []string, patterns []contract.ForbiddenPattern) (*DetectionResult, error) {
	result := &DetectionResult{}

	if len(patterns) == 0 {
		return result, nil
	}

	// Pre-compile all patterns
	compiled := make([]compiledPattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling pattern %q: %w", p.Pattern, err)
		}
		compiled = append(compiled, compiledPattern{
			regex:       re,
			description: p.Description,
		})
	}

	// Scan each file
	for _, file := range files {
		violations, err := scanFileForPatterns(file, compiled)
		if err != nil {
			return nil, fmt.Errorf("scanning %s: %w", file, err)
		}
		result.Violations = append(result.Violations, violations...)
		result.Scanned++
	}

	return result, nil
}

// scanFileForPatterns scans a single file for forbidden patterns.
func scanFileForPatterns(filePath string, patterns []compiledPattern) ([]Violation, error) {
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

		for _, p := range patterns {
			if p.regex.MatchString(line) {
				msg := fmt.Sprintf("forbidden pattern %q found", p.regex.String())
				if p.description != "" {
					msg = fmt.Sprintf("%s: %s", msg, p.description)
				}
				violations = append(violations, Violation{
					Rule:     RuleForbiddenPattern,
					Message:  msg,
					File:     filePath,
					Line:     lineNum,
					Severity: SeverityError,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return violations, nil
}
