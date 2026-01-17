package detect

import (
	"bufio"
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
)

// SuppressionType indicates how the suppression applies.
type SuppressionType string

const (
	SuppressionLine     SuppressionType = "line"      // Applies to same line
	SuppressionNextLine SuppressionType = "next-line" // Applies to next line
	SuppressionFile     SuppressionType = "file"      // Applies to entire file
)

// Suppression represents an inline suppression directive.
type Suppression struct {
	Rule   string          // Rule to suppress (e.g., "forbidden_pattern")
	Reason string          // Human-readable reason
	File   string          // File containing the suppression
	Line   int             // Line number (0 for file-level)
	Type   SuppressionType // How the suppression applies
}

// SuppressedViolation represents a violation that was suppressed.
type SuppressedViolation struct {
	Violation   Violation
	Suppression Suppression
}

// suppressionPattern matches hollowcheck suppression comments.
// Supports: // hollowcheck:ignore[-file|-next-line] <rule> - <reason>
var suppressionPatterns = []*regexp.Regexp{
	// Go/JS/TS style: // hollowcheck:...
	regexp.MustCompile(`//\s*hollowcheck:(ignore(?:-file|-next-line)?)\s+(\S+)\s*(?:-\s*(.*))?`),
	// Python/Shell style: # hollowcheck:...
	regexp.MustCompile(`#\s*hollowcheck:(ignore(?:-file|-next-line)?)\s+(\S+)\s*(?:-\s*(.*))?`),
	// Block comment style: /* hollowcheck:... */
	regexp.MustCompile(`/\*\s*hollowcheck:(ignore(?:-file|-next-line)?)\s+(\S+)\s*(?:-\s*(.*?))?\s*\*/`),
	// HTML comment style: <!-- hollowcheck:... -->
	regexp.MustCompile(`<!--\s*hollowcheck:(ignore(?:-file|-next-line)?)\s+(\S+)\s*(?:-\s*(.*?))?\s*-->`),
}

// languageCommentPrefixes maps file extensions to comment prefixes for validation.
var languageCommentPrefixes = map[string][]string{
	".go":   {"//", "/*"},
	".js":   {"//", "/*"},
	".ts":   {"//", "/*"},
	".tsx":  {"//", "/*"},
	".jsx":  {"//", "/*"},
	".py":   {"#"},
	".rb":   {"#"},
	".sh":   {"#"},
	".bash": {"#"},
	".yaml": {"#"},
	".yml":  {"#"},
	".c":    {"//", "/*"},
	".cpp":  {"//", "/*"},
	".h":    {"//", "/*"},
	".hpp":  {"//", "/*"},
	".java": {"//", "/*"},
	".kt":   {"//", "/*"},
	".rs":   {"//", "/*"},
	".md":   {"<!--"},
	".html": {"<!--"},
	".xml":  {"<!--"},
}

// ParseSuppressions scans file content for suppression directives.
func ParseSuppressions(filePath string, content []byte) []Suppression {
	var suppressions []Suppression

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	inPackageBlock := true // Track if we're before the first non-comment code

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check if we've passed the header section (for file-level suppressions)
		if inPackageBlock && !isCommentOrEmpty(trimmed, filePath) {
			inPackageBlock = false
		}

		// Try each suppression pattern
		for _, pattern := range suppressionPatterns {
			matches := pattern.FindStringSubmatch(line)
			if matches == nil {
				continue
			}

			directive := matches[1] // "ignore", "ignore-file", or "ignore-next-line"
			rule := matches[2]
			reason := ""
			if len(matches) > 3 {
				reason = strings.TrimSpace(matches[3])
			}

			suppression := Suppression{
				Rule:   rule,
				Reason: reason,
				File:   filePath,
				Line:   lineNum,
			}

			switch directive {
			case "ignore-file":
				// File-level suppressions must be at the top of the file
				if !inPackageBlock && lineNum > 10 {
					// Allow some flexibility but warn implicitly by not applying
					continue
				}
				suppression.Type = SuppressionFile
				suppression.Line = 0 // File-level
			case "ignore-next-line":
				suppression.Type = SuppressionNextLine
				// The suppression applies to the next line
			case "ignore":
				suppression.Type = SuppressionLine
				// Check if this is on its own line or inline with code
				// If the line only contains the comment, it acts like next-line
				if isOnlyComment(trimmed) {
					suppression.Type = SuppressionNextLine
				}
			}

			suppressions = append(suppressions, suppression)
			break // Only one suppression per line
		}
	}

	return suppressions
}

// isCommentOrEmpty checks if a line is a comment or empty for the given file type.
func isCommentOrEmpty(line, filePath string) bool {
	if line == "" {
		return true
	}

	ext := filepath.Ext(filePath)
	prefixes, ok := languageCommentPrefixes[ext]
	if !ok {
		// Unknown file type, be conservative
		prefixes = []string{"//", "#", "/*", "<!--"}
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	return false
}

// isOnlyComment checks if the line contains only a comment (no code).
func isOnlyComment(line string) bool {
	// Check for common comment-only patterns
	return strings.HasPrefix(line, "//") ||
		strings.HasPrefix(line, "#") ||
		strings.HasPrefix(line, "/*") ||
		strings.HasPrefix(line, "<!--") ||
		(strings.HasPrefix(line, "*") && !strings.HasPrefix(line, "*/"))
}

// MatchesSuppression checks if a violation is suppressed by the given suppression.
func MatchesSuppression(v Violation, s Suppression) bool {
	// Must be same file
	if v.File != s.File {
		return false
	}

	// Must match rule (or suppression is for all rules with "*")
	if s.Rule != "*" && s.Rule != v.Rule {
		return false
	}

	switch s.Type {
	case SuppressionFile:
		// File-level suppression matches any line in the file
		return true
	case SuppressionLine:
		// Line-level suppression matches the same line
		return v.Line == s.Line
	case SuppressionNextLine:
		// Next-line suppression matches the line after the comment
		return v.Line == s.Line+1
	}

	return false
}

// FilterSuppressed separates violations into active and suppressed based on suppressions.
func FilterSuppressed(violations []Violation, suppressions []Suppression) (active []Violation, suppressed []SuppressedViolation) {
	for _, v := range violations {
		wasSuppressed := false
		for _, s := range suppressions {
			if MatchesSuppression(v, s) {
				suppressed = append(suppressed, SuppressedViolation{
					Violation:   v,
					Suppression: s,
				})
				wasSuppressed = true
				break
			}
		}
		if !wasSuppressed {
			active = append(active, v)
		}
	}
	return active, suppressed
}

// CollectSuppressions reads all files and collects their suppressions.
func CollectSuppressions(files []string, readFile func(string) ([]byte, error)) (map[string][]Suppression, error) {
	result := make(map[string][]Suppression)

	for _, file := range files {
		content, err := readFile(file)
		if err != nil {
			// Skip files we can't read
			continue
		}

		suppressions := ParseSuppressions(file, content)
		if len(suppressions) > 0 {
			result[file] = suppressions
		}
	}

	return result, nil
}
