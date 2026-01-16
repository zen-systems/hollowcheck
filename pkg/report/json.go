// Package report provides output formatting for hollowcheck results.
package report

import (
	"encoding/json"
	"io"

	"github.com/zen-systems/hollowcheck/pkg/detect"
	"github.com/zen-systems/hollowcheck/pkg/score"
)

// JSONReport represents the complete lint result in JSON format.
type JSONReport struct {
	Version    string           `json:"version"`
	Path       string           `json:"path"`
	Contract   string           `json:"contract"`
	Score      int              `json:"score"`
	Grade      string           `json:"grade"`
	Threshold  int              `json:"threshold"`
	Passed     bool             `json:"passed"`
	Scanned    int              `json:"files_scanned"`
	Violations []JSONViolation  `json:"violations"`
	Breakdown  []BreakdownEntry `json:"breakdown"`
}

// JSONViolation represents a single violation in JSON format.
type JSONViolation struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
}

// BreakdownEntry represents a category in the score breakdown.
type BreakdownEntry struct {
	Rule       string `json:"rule"`
	Points     int    `json:"points"`
	Violations int    `json:"violations"`
}

// WriteJSON writes the lint result as JSON to the given writer.
func WriteJSON(w io.Writer, path, contractPath, version string, result *detect.DetectionResult, hollowness score.HollownessScore) error {
	report := JSONReport{
		Version:   version,
		Path:      path,
		Contract:  contractPath,
		Score:     hollowness.Score,
		Grade:     hollowness.Grade,
		Threshold: hollowness.Threshold,
		Passed:    hollowness.Passed,
		Scanned:   result.Scanned,
	}

	// Convert violations
	report.Violations = make([]JSONViolation, 0, len(result.Violations))
	for _, v := range result.Violations {
		report.Violations = append(report.Violations, JSONViolation{
			Rule:     v.Rule,
			Severity: v.Severity,
			File:     v.File,
			Line:     v.Line,
			Message:  v.Message,
		})
	}

	// Build breakdown
	report.Breakdown = make([]BreakdownEntry, 0, len(hollowness.Breakdown))
	for rule, points := range hollowness.Breakdown {
		count := hollowness.ViolationCount(rule)
		report.Breakdown = append(report.Breakdown, BreakdownEntry{
			Rule:       rule,
			Points:     points,
			Violations: count,
		})
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
