package report

import (
	"fmt"
	"io"
	"sort"

	"github.com/fatih/color"

	"github.com/zen-systems/hollowcheck/pkg/detect"
	"github.com/zen-systems/hollowcheck/pkg/score"
)

var (
	// Colors for output
	titleColor   = color.New(color.FgCyan, color.Bold)
	errorColor   = color.New(color.FgRed)
	warnColor    = color.New(color.FgYellow)
	successColor = color.New(color.FgGreen)
	boldColor    = color.New(color.Bold)
	dimColor     = color.New(color.Faint)
	fileColor    = color.New(color.FgBlue)
)

// WritePretty writes the lint result as human-readable colored output.
func WritePretty(w io.Writer, path, contractPath, version string, result *detect.DetectionResult, hollowness score.HollownessScore) {
	// Header
	fmt.Fprintln(w)
	titleColor.Fprintf(w, "  hollowcheck v%s\n", version)
	fmt.Fprintln(w)

	// Scan info
	dimColor.Fprintf(w, "  Scanning: ")
	fmt.Fprintln(w, path)
	dimColor.Fprintf(w, "  Contract: ")
	fmt.Fprintln(w, contractPath)
	fmt.Fprintln(w)

	// Result summary
	writeResultSummary(w, hollowness)
	fmt.Fprintln(w)

	// Violations
	if len(result.Violations) > 0 {
		writeViolations(w, result.Violations)
		fmt.Fprintln(w)
	}

	// Breakdown
	if len(hollowness.Breakdown) > 0 {
		writeBreakdown(w, hollowness)
		fmt.Fprintln(w)
	}

	// Final status line
	writeFinalStatus(w, hollowness)
	fmt.Fprintln(w)
}

func writeResultSummary(w io.Writer, hollowness score.HollownessScore) {
	if hollowness.Passed {
		successColor.Fprintf(w, "  ✓ PASS")
	} else {
		errorColor.Fprintf(w, "  ✗ FAIL")
	}

	fmt.Fprintf(w, "  Hollowness: ")
	writeColoredScore(w, hollowness.Score)
	fmt.Fprintf(w, "%%  Grade: ")
	writeColoredGrade(w, hollowness.Grade)
	fmt.Fprintln(w)
}

func writeColoredScore(w io.Writer, score int) {
	switch {
	case score <= 10:
		successColor.Fprintf(w, "%d", score)
	case score <= 25:
		color.New(color.FgGreen).Fprintf(w, "%d", score)
	case score <= 50:
		warnColor.Fprintf(w, "%d", score)
	case score <= 75:
		color.New(color.FgHiYellow).Fprintf(w, "%d", score)
	default:
		errorColor.Fprintf(w, "%d", score)
	}
}

func writeColoredGrade(w io.Writer, grade string) {
	switch grade {
	case "A":
		successColor.Fprintf(w, "%s", grade)
	case "B":
		color.New(color.FgGreen).Fprintf(w, "%s", grade)
	case "C":
		warnColor.Fprintf(w, "%s", grade)
	case "D":
		color.New(color.FgHiYellow).Fprintf(w, "%s", grade)
	default:
		errorColor.Fprintf(w, "%s", grade)
	}
}

func writeViolations(w io.Writer, violations []detect.Violation) {
	boldColor.Fprintf(w, "  Violations (%d):\n", len(violations))
	fmt.Fprintln(w)

	for _, v := range violations {
		writeSeverityTag(w, v.Severity)
		fmt.Fprintf(w, "   ")
		dimColor.Fprintf(w, "%-18s", v.Rule)
		fileColor.Fprintf(w, "%s", v.File)
		if v.Line > 0 {
			dimColor.Fprintf(w, ":%d", v.Line)
		}
		fmt.Fprintln(w)

		// Message on next line, indented
		fmt.Fprintf(w, "            %s\n", v.Message)
		fmt.Fprintln(w)
	}
}

func writeSeverityTag(w io.Writer, severity string) {
	switch severity {
	case detect.SeverityError:
		errorColor.Fprintf(w, "    ERROR ")
	case detect.SeverityWarning:
		warnColor.Fprintf(w, "    WARN  ")
	default:
		fmt.Fprintf(w, "    %-6s", severity)
	}
}

func writeBreakdown(w io.Writer, hollowness score.HollownessScore) {
	boldColor.Fprintln(w, "  Breakdown:")

	// Sort rules for consistent output
	rules := make([]string, 0, len(hollowness.Breakdown))
	for rule := range hollowness.Breakdown {
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool {
		// Sort by points descending
		return hollowness.Breakdown[rules[i]] > hollowness.Breakdown[rules[j]]
	})

	for _, rule := range rules {
		points := hollowness.Breakdown[rule]
		count := hollowness.ViolationCount(rule)
		fmt.Fprintf(w, "    %-20s %3d pts (%d violation", rule, points, count)
		if count != 1 {
			fmt.Fprint(w, "s")
		}
		fmt.Fprintln(w, ")")
	}
}

func writeFinalStatus(w io.Writer, hollowness score.HollownessScore) {
	dimColor.Fprintf(w, "  Threshold: %d", hollowness.Threshold)
	fmt.Fprintf(w, "  Score: ")
	writeColoredScore(w, hollowness.Score)
	fmt.Fprintf(w, "  ")

	if hollowness.Passed {
		successColor.Fprint(w, "PASSED")
	} else {
		errorColor.Fprint(w, "FAILED")
	}
	fmt.Fprintln(w)
}
