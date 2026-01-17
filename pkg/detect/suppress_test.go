package detect

import (
	"testing"
)

func TestParseSuppressions_Go(t *testing.T) {
	content := []byte(`// hollowcheck:ignore-file mock_data - test fixtures are intentional
package fixtures

// hollowcheck:ignore forbidden_pattern - legacy code, JIRA-1234
func oldHandler() {
	// TODO: refactor when we migrate
}

func anotherFunc() {
	// hollowcheck:ignore-next-line forbidden_pattern - temporary workaround
	// FIXME: this is broken
	doSomething()
}

func inlineSuppress() {
	x := "test" // hollowcheck:ignore mock_data - inline test value
}
`)

	suppressions := ParseSuppressions("/path/to/file.go", content)

	if len(suppressions) != 4 {
		t.Fatalf("expected 4 suppressions, got %d", len(suppressions))
	}

	// Test file-level suppression
	s := suppressions[0]
	if s.Type != SuppressionFile {
		t.Errorf("expected SuppressionFile, got %s", s.Type)
	}
	if s.Rule != "mock_data" {
		t.Errorf("expected rule 'mock_data', got %s", s.Rule)
	}
	if s.Reason != "test fixtures are intentional" {
		t.Errorf("expected reason 'test fixtures are intentional', got %q", s.Reason)
	}
	if s.Line != 0 {
		t.Errorf("expected line 0 for file suppression, got %d", s.Line)
	}

	// Test line suppression (on its own line, acts as next-line)
	s = suppressions[1]
	if s.Type != SuppressionNextLine {
		t.Errorf("expected SuppressionNextLine, got %s", s.Type)
	}
	if s.Rule != "forbidden_pattern" {
		t.Errorf("expected rule 'forbidden_pattern', got %s", s.Rule)
	}
	if s.Reason != "legacy code, JIRA-1234" {
		t.Errorf("expected reason 'legacy code, JIRA-1234', got %q", s.Reason)
	}

	// Test explicit next-line suppression
	s = suppressions[2]
	if s.Type != SuppressionNextLine {
		t.Errorf("expected SuppressionNextLine, got %s", s.Type)
	}
	if s.Rule != "forbidden_pattern" {
		t.Errorf("expected rule 'forbidden_pattern', got %s", s.Rule)
	}

	// Test inline suppression
	s = suppressions[3]
	if s.Type != SuppressionLine {
		t.Errorf("expected SuppressionLine, got %s", s.Type)
	}
	if s.Rule != "mock_data" {
		t.Errorf("expected rule 'mock_data', got %s", s.Rule)
	}
}

func TestParseSuppressions_Python(t *testing.T) {
	content := []byte(`# hollowcheck:ignore-file mock_data - test fixtures
import unittest

def test_something():
    # hollowcheck:ignore forbidden_pattern - test case
    # TODO: add more tests
    pass

def inline_test():
    x = "example"  # hollowcheck:ignore mock_data - test value
`)

	suppressions := ParseSuppressions("/path/to/file.py", content)

	if len(suppressions) != 3 {
		t.Fatalf("expected 3 suppressions, got %d", len(suppressions))
	}

	// Test file-level suppression
	s := suppressions[0]
	if s.Type != SuppressionFile {
		t.Errorf("expected SuppressionFile, got %s", s.Type)
	}
	if s.Rule != "mock_data" {
		t.Errorf("expected rule 'mock_data', got %s", s.Rule)
	}

	// Test line suppression
	s = suppressions[1]
	if s.Type != SuppressionNextLine {
		t.Errorf("expected SuppressionNextLine, got %s", s.Type)
	}
	if s.Rule != "forbidden_pattern" {
		t.Errorf("expected rule 'forbidden_pattern', got %s", s.Rule)
	}
}

func TestParseSuppressions_JavaScript(t *testing.T) {
	content := []byte(`// hollowcheck:ignore-file mock_data - test fixtures
const fixtures = {
    // hollowcheck:ignore-next-line forbidden_pattern - known issue
    // TODO: fix this later
    user: { id: "user-123" }
};

/* hollowcheck:ignore mock_data - block comment */
const testEmail = "test@example.com";
`)

	suppressions := ParseSuppressions("/path/to/file.js", content)

	if len(suppressions) != 3 {
		t.Fatalf("expected 3 suppressions, got %d", len(suppressions))
	}

	// Test file-level suppression
	if suppressions[0].Type != SuppressionFile {
		t.Errorf("expected SuppressionFile, got %s", suppressions[0].Type)
	}

	// Test next-line suppression
	if suppressions[1].Type != SuppressionNextLine {
		t.Errorf("expected SuppressionNextLine, got %s", suppressions[1].Type)
	}

	// Test block comment suppression
	if suppressions[2].Type != SuppressionNextLine {
		t.Errorf("expected SuppressionNextLine for block comment on its own line, got %s", suppressions[2].Type)
	}
}

func TestMatchesSuppression_LineSuppression(t *testing.T) {
	violation := Violation{
		Rule:     RuleForbiddenPattern,
		File:     "/path/to/file.go",
		Line:     10,
		Message:  "TODO found",
		Severity: SeverityError,
	}

	suppression := Suppression{
		Rule: RuleForbiddenPattern,
		File: "/path/to/file.go",
		Line: 10,
		Type: SuppressionLine,
	}

	if !MatchesSuppression(violation, suppression) {
		t.Error("expected violation to match line suppression")
	}

	// Wrong line
	suppression.Line = 11
	if MatchesSuppression(violation, suppression) {
		t.Error("expected violation NOT to match suppression on different line")
	}
}

func TestMatchesSuppression_NextLineSuppression(t *testing.T) {
	violation := Violation{
		Rule:     RuleForbiddenPattern,
		File:     "/path/to/file.go",
		Line:     11,
		Message:  "TODO found",
		Severity: SeverityError,
	}

	suppression := Suppression{
		Rule: RuleForbiddenPattern,
		File: "/path/to/file.go",
		Line: 10, // Suppression on line 10, applies to line 11
		Type: SuppressionNextLine,
	}

	if !MatchesSuppression(violation, suppression) {
		t.Error("expected violation to match next-line suppression")
	}

	// Wrong line
	violation.Line = 12
	if MatchesSuppression(violation, suppression) {
		t.Error("expected violation NOT to match next-line suppression for wrong line")
	}
}

func TestMatchesSuppression_FileSuppression(t *testing.T) {
	violation := Violation{
		Rule:     RuleMockData,
		File:     "/path/to/file.go",
		Line:     100,
		Message:  "mock data found",
		Severity: SeverityWarning,
	}

	suppression := Suppression{
		Rule: RuleMockData,
		File: "/path/to/file.go",
		Line: 0,
		Type: SuppressionFile,
	}

	if !MatchesSuppression(violation, suppression) {
		t.Error("expected violation to match file suppression")
	}

	// Any line in the file should match
	violation.Line = 1
	if !MatchesSuppression(violation, suppression) {
		t.Error("expected violation on line 1 to match file suppression")
	}
}

func TestMatchesSuppression_WildcardRule(t *testing.T) {
	violation := Violation{
		Rule:     RuleForbiddenPattern,
		File:     "/path/to/file.go",
		Line:     10,
		Message:  "TODO found",
		Severity: SeverityError,
	}

	suppression := Suppression{
		Rule: "*", // Wildcard matches any rule
		File: "/path/to/file.go",
		Line: 10,
		Type: SuppressionLine,
	}

	if !MatchesSuppression(violation, suppression) {
		t.Error("expected violation to match wildcard suppression")
	}
}

func TestMatchesSuppression_WrongFile(t *testing.T) {
	violation := Violation{
		Rule:     RuleForbiddenPattern,
		File:     "/path/to/file.go",
		Line:     10,
		Message:  "TODO found",
		Severity: SeverityError,
	}

	suppression := Suppression{
		Rule: RuleForbiddenPattern,
		File: "/path/to/other.go", // Different file
		Line: 10,
		Type: SuppressionLine,
	}

	if MatchesSuppression(violation, suppression) {
		t.Error("expected violation NOT to match suppression in different file")
	}
}

func TestMatchesSuppression_WrongRule(t *testing.T) {
	violation := Violation{
		Rule:     RuleForbiddenPattern,
		File:     "/path/to/file.go",
		Line:     10,
		Message:  "TODO found",
		Severity: SeverityError,
	}

	suppression := Suppression{
		Rule: RuleMockData, // Different rule
		File: "/path/to/file.go",
		Line: 10,
		Type: SuppressionLine,
	}

	if MatchesSuppression(violation, suppression) {
		t.Error("expected violation NOT to match suppression for different rule")
	}
}

func TestFilterSuppressed(t *testing.T) {
	violations := []Violation{
		{Rule: RuleForbiddenPattern, File: "/path/file.go", Line: 10, Severity: SeverityError},
		{Rule: RuleForbiddenPattern, File: "/path/file.go", Line: 20, Severity: SeverityError},
		{Rule: RuleMockData, File: "/path/file.go", Line: 30, Severity: SeverityWarning},
	}

	suppressions := []Suppression{
		{Rule: RuleForbiddenPattern, File: "/path/file.go", Line: 9, Type: SuppressionNextLine}, // Suppresses line 10
		{Rule: RuleMockData, File: "/path/file.go", Line: 0, Type: SuppressionFile},            // Suppresses all mock_data
	}

	active, suppressed := FilterSuppressed(violations, suppressions)

	if len(active) != 1 {
		t.Errorf("expected 1 active violation, got %d", len(active))
	}
	if len(suppressed) != 2 {
		t.Errorf("expected 2 suppressed violations, got %d", len(suppressed))
	}

	// The active violation should be the one on line 20
	if active[0].Line != 20 {
		t.Errorf("expected active violation to be on line 20, got %d", active[0].Line)
	}
}

func TestFilterSuppressed_NoSuppressions(t *testing.T) {
	violations := []Violation{
		{Rule: RuleForbiddenPattern, File: "/path/file.go", Line: 10, Severity: SeverityError},
	}

	active, suppressed := FilterSuppressed(violations, nil)

	if len(active) != 1 {
		t.Errorf("expected 1 active violation, got %d", len(active))
	}
	if len(suppressed) != 0 {
		t.Errorf("expected 0 suppressed violations, got %d", len(suppressed))
	}
}

func TestFilterSuppressed_AllSuppressed(t *testing.T) {
	violations := []Violation{
		{Rule: RuleForbiddenPattern, File: "/path/file.go", Line: 10, Severity: SeverityError},
	}

	suppressions := []Suppression{
		{Rule: "*", File: "/path/file.go", Line: 0, Type: SuppressionFile}, // Suppress everything
	}

	active, suppressed := FilterSuppressed(violations, suppressions)

	if len(active) != 0 {
		t.Errorf("expected 0 active violations, got %d", len(active))
	}
	if len(suppressed) != 1 {
		t.Errorf("expected 1 suppressed violation, got %d", len(suppressed))
	}
}

func TestParseSuppressions_NoReason(t *testing.T) {
	content := []byte(`// hollowcheck:ignore forbidden_pattern
func test() {
	// TODO: something
}
`)

	suppressions := ParseSuppressions("/path/to/file.go", content)

	if len(suppressions) != 1 {
		t.Fatalf("expected 1 suppression, got %d", len(suppressions))
	}

	if suppressions[0].Reason != "" {
		t.Errorf("expected empty reason, got %q", suppressions[0].Reason)
	}
}

func TestParseSuppressions_HTML(t *testing.T) {
	content := []byte(`<!-- hollowcheck:ignore-file filler_phrase - marketing copy -->
<html>
<body>
<!-- hollowcheck:ignore weasel_word - acceptable here -->
<p>Studies show this is effective.</p>
</body>
</html>
`)

	suppressions := ParseSuppressions("/path/to/file.html", content)

	if len(suppressions) != 2 {
		t.Fatalf("expected 2 suppressions, got %d", len(suppressions))
	}

	if suppressions[0].Type != SuppressionFile {
		t.Errorf("expected SuppressionFile, got %s", suppressions[0].Type)
	}
	if suppressions[0].Rule != "filler_phrase" {
		t.Errorf("expected rule 'filler_phrase', got %s", suppressions[0].Rule)
	}
}
