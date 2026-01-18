package analysis

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/zen-systems/hollowcheck/pkg/contract"
)

// SimpleAnalyzer implements the Analyzer interface using regex-based pattern matching.
// This is a pure Go implementation that does not require CGO or tree-sitter.
// It provides a fallback for environments where CGO is not available (e.g., Windows without MinGW).
//
// Capabilities:
//   - Forbidden pattern detection (regex-based line scanning)
//   - Mock data signature detection
//
// Limitations (compared to SmartAnalyzer):
//   - No AST-based symbol detection
//   - No cyclomatic complexity analysis
//   - Pattern matching is line-based, not context-aware
type SimpleAnalyzer struct{}

// NewSimpleAnalyzer creates a new SimpleAnalyzer instance.
func NewSimpleAnalyzer() *SimpleAnalyzer {
	return &SimpleAnalyzer{}
}

// Name returns the analyzer engine name.
func (a *SimpleAnalyzer) Name() string {
	return "simple"
}

// Analyze scans the provided files for violations using regex-based pattern matching.
func (a *SimpleAnalyzer) Analyze(files map[string]string, c *contract.Contract) (*Result, error) {
	result := &Result{}

	// Pre-compile forbidden patterns
	forbiddenPatterns, err := compilePatterns(c.ForbiddenPatterns)
	if err != nil {
		return nil, fmt.Errorf("compiling forbidden patterns: %w", err)
	}

	// Pre-compile mock data patterns
	var mockPatterns []compiledPattern
	if c.MockSignatures != nil && len(c.MockSignatures.Patterns) > 0 {
		mockPatterns, err = compileMockPatterns(c.MockSignatures.Patterns)
		if err != nil {
			return nil, fmt.Errorf("compiling mock patterns: %w", err)
		}
	}

	// Scan each file
	for filePath, content := range files {
		// Check forbidden patterns
		violations := scanContentForPatterns(filePath, content, forbiddenPatterns, RuleForbiddenPattern, SeverityError)
		result.Violations = append(result.Violations, violations...)

		// Check mock data patterns (skip test files if configured)
		if len(mockPatterns) > 0 {
			isTestFile := strings.HasSuffix(filePath, "_test.go") || strings.Contains(filePath, "/test/")

			if c.MockSignatures.ShouldSkipTestFiles() && isTestFile {
				// Skip test files entirely
				continue
			}

			severity := SeverityWarning
			if isTestFile {
				testSeverity := c.MockSignatures.GetTestFileSeverity()
				if testSeverity == "" {
					continue // Skip
				}
				severity = testSeverity
			}

			mockViolations := scanContentForPatterns(filePath, content, mockPatterns, RuleMockData, severity)
			result.Violations = append(result.Violations, mockViolations...)
		}

		result.Scanned++
	}

	return result, nil
}

// compiledPattern holds a pre-compiled regex with its metadata.
type compiledPattern struct {
	regex       *regexp.Regexp
	description string
}

// compilePatterns pre-compiles forbidden pattern regexes.
func compilePatterns(patterns []contract.ForbiddenPattern) ([]compiledPattern, error) {
	compiled := make([]compiledPattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", p.Pattern, err)
		}
		compiled = append(compiled, compiledPattern{
			regex:       re,
			description: p.Description,
		})
	}
	return compiled, nil
}

// compileMockPatterns pre-compiles mock signature regexes.
func compileMockPatterns(patterns []contract.MockSignature) ([]compiledPattern, error) {
	compiled := make([]compiledPattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid mock pattern %q: %w", p.Pattern, err)
		}
		compiled = append(compiled, compiledPattern{
			regex:       re,
			description: p.Description,
		})
	}
	return compiled, nil
}

// scanContentForPatterns scans file content for pattern violations.
func scanContentForPatterns(filePath, content string, patterns []compiledPattern, rule, severity string) []Violation {
	if len(patterns) == 0 {
		return nil
	}

	var violations []Violation
	scanner := bufio.NewScanner(strings.NewReader(content))
	// Increase buffer size for minified files (1MB max line length)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, p := range patterns {
			// Find all matches with their positions
			matches := p.regex.FindAllStringIndex(line, -1)
			for _, match := range matches {
				// Skip if match is inside a string literal
				if isInsideStringLiteral(line, match[0]) {
					continue
				}

				msg := fmt.Sprintf("%s pattern %q found", ruleDescription(rule), p.regex.String())
				if p.description != "" {
					msg = fmt.Sprintf("%s: %s", msg, p.description)
				}
				violations = append(violations, Violation{
					Rule:     rule,
					Message:  msg,
					File:     filePath,
					Line:     lineNum,
					Severity: severity,
				})
			}
		}
	}

	return violations
}

// ruleDescription returns a human-readable description for a rule.
func ruleDescription(rule string) string {
	switch rule {
	case RuleForbiddenPattern:
		return "forbidden"
	case RuleMockData:
		return "mock data"
	default:
		return rule
	}
}

// isInsideStringLiteral checks if a position in a line falls within a string literal.
// Supports double-quoted, single-quoted, and backtick strings with escape handling.
func isInsideStringLiteral(line string, pos int) bool {
	// Track string state as we scan
	var inString bool
	var stringChar rune
	escaped := false

	for i, ch := range line {
		if i >= pos {
			return inString
		}

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		// Check for string delimiters
		if ch == '"' || ch == '\'' || ch == '`' {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				inString = false
			}
		}
	}

	return inString
}
