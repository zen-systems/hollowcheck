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
//
// Key Features:
//   - Language-agnostic: Works with ANY file type (Go, Python, Rust, C++, Java, etc.)
//   - No file extension filtering: Scans ALL files provided in the input map
//   - Detailed reporting: Includes line number, column, matched content, and context
//   - Efficient: Pre-compiles all regex patterns before scanning
//
// Capabilities:
//   - Forbidden pattern detection (regex-based line scanning)
//   - Mock data signature detection
//   - Universal placeholder detection (TODO, FIXME, etc.)
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
// This method is completely language-agnostic - it processes ANY file in the input map
// regardless of extension. The contract patterns determine what gets flagged.
func (a *SimpleAnalyzer) Analyze(files map[string]string, c *contract.Contract) (*Result, error) {
	result := &Result{}

	// Pre-compile all patterns for efficiency
	forbiddenPatterns, err := compilePatterns(c.ForbiddenPatterns)
	if err != nil {
		return nil, fmt.Errorf("compiling forbidden patterns: %w", err)
	}

	var mockPatterns []compiledPattern
	if c.MockSignatures != nil && len(c.MockSignatures.Patterns) > 0 {
		mockPatterns, err = compileMockPatterns(c.MockSignatures.Patterns)
		if err != nil {
			return nil, fmt.Errorf("compiling mock patterns: %w", err)
		}
	}

	// Scan every file in the map - NO extension filtering
	// This makes SimpleAnalyzer truly polyglot
	for filePath, content := range files {
		fileViolations := a.scanFile(filePath, content, forbiddenPatterns, mockPatterns, c)
		result.Violations = append(result.Violations, fileViolations...)
		result.Scanned++
	}

	return result, nil
}

// scanFile processes a single file for all pattern types.
func (a *SimpleAnalyzer) scanFile(filePath, content string, forbidden, mock []compiledPattern, c *contract.Contract) []Violation {
	var violations []Violation

	// Check forbidden patterns (always applies)
	if len(forbidden) > 0 {
		v := scanContentForPatterns(filePath, content, forbidden, RuleForbiddenPattern, SeverityError)
		violations = append(violations, v...)
	}

	// Check mock data patterns with test file handling
	if len(mock) > 0 {
		isTestFile := isTestFilePath(filePath)

		if c.MockSignatures != nil && c.MockSignatures.ShouldSkipTestFiles() && isTestFile {
			// Skip test files for mock detection
		} else {
			severity := SeverityWarning
			if isTestFile && c.MockSignatures != nil {
				testSeverity := c.MockSignatures.GetTestFileSeverity()
				if testSeverity == "" {
					// Skip entirely
				} else {
					severity = testSeverity
					v := scanContentForPatterns(filePath, content, mock, RuleMockData, severity)
					violations = append(violations, v...)
				}
			} else {
				v := scanContentForPatterns(filePath, content, mock, RuleMockData, severity)
				violations = append(violations, v...)
			}
		}
	}

	return violations
}

// isTestFilePath detects test files across multiple languages.
func isTestFilePath(path string) bool {
	lowerPath := strings.ToLower(path)

	// Go test files
	if strings.HasSuffix(lowerPath, "_test.go") {
		return true
	}

	// Python test files
	if strings.HasSuffix(lowerPath, "_test.py") || strings.HasSuffix(lowerPath, "test_.py") {
		return true
	}
	if strings.Contains(lowerPath, "/tests/") || strings.Contains(lowerPath, "/test/") {
		return true
	}

	// Java/Kotlin test files
	if strings.HasSuffix(lowerPath, "test.java") || strings.HasSuffix(lowerPath, "test.kt") {
		return true
	}
	if strings.Contains(lowerPath, "/src/test/") {
		return true
	}

	// JavaScript/TypeScript test files
	if strings.HasSuffix(lowerPath, ".test.js") || strings.HasSuffix(lowerPath, ".test.ts") ||
		strings.HasSuffix(lowerPath, ".spec.js") || strings.HasSuffix(lowerPath, ".spec.ts") {
		return true
	}
	if strings.Contains(lowerPath, "/__tests__/") {
		return true
	}

	// Rust test files
	if strings.Contains(lowerPath, "/tests/") {
		return true
	}

	return false
}

// compiledPattern holds a pre-compiled regex with its metadata.
type compiledPattern struct {
	regex       *regexp.Regexp
	pattern     string // Original pattern string for display
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
			pattern:     p.Pattern,
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
			pattern:     p.Pattern,
			description: p.Description,
		})
	}
	return compiled, nil
}

// scanContentForPatterns scans file content for pattern violations.
// Returns detailed violations including line number, column, matched text, and context.
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
			matches := p.regex.FindAllStringSubmatchIndex(line, -1)
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}

				startPos := match[0]
				endPos := match[1]

				// Skip if match is inside a string literal (configurable behavior)
				// This helps avoid false positives in string constants
				if isInsideStringLiteral(line, startPos) {
					continue
				}

				// Extract the matched content
				matchedText := line[startPos:endPos]

				// Build the message
				msg := formatViolationMessage(rule, p.pattern, p.description, matchedText)

				// Create context (trimmed line for display)
				context := strings.TrimSpace(line)
				if len(context) > 120 {
					// Truncate long lines but try to keep the match visible
					if startPos < 60 {
						context = context[:117] + "..."
					} else if startPos > len(context)-60 {
						context = "..." + context[len(context)-117:]
					} else {
						// Match is in the middle, show around it
						start := startPos - 55
						end := startPos + 60
						if end > len(context) {
							end = len(context)
						}
						context = "..." + context[start:end] + "..."
					}
				}

				violations = append(violations, Violation{
					Rule:     rule,
					Message:  msg,
					File:     filePath,
					Line:     lineNum,
					Column:   startPos + 1, // 1-indexed column
					Match:    matchedText,
					Context:  context,
					Severity: severity,
				})
			}
		}
	}

	return violations
}

// formatViolationMessage creates a human-readable violation message.
func formatViolationMessage(rule, pattern, description, matchedText string) string {
	var msg string

	switch rule {
	case RuleForbiddenPattern:
		msg = fmt.Sprintf("forbidden pattern found: %q", matchedText)
	case RuleMockData:
		msg = fmt.Sprintf("mock/placeholder data found: %q", matchedText)
	default:
		msg = fmt.Sprintf("pattern %q matched: %q", pattern, matchedText)
	}

	if description != "" {
		msg = fmt.Sprintf("%s - %s", msg, description)
	}

	return msg
}

// isInsideStringLiteral checks if a position in a line falls within a string literal.
// Supports double-quoted, single-quoted, and backtick strings with escape handling.
// Works across multiple languages (Go, Python, JavaScript, Rust, etc.).
func isInsideStringLiteral(line string, pos int) bool {
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

		if ch == '\\' && inString && stringChar != '`' {
			// Backticks (raw strings in Go/JS) don't use escape sequences
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
