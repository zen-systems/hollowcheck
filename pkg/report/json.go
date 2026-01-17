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
	Version         string                    `json:"version"`
	Path            string                    `json:"path"`
	Contract        string                    `json:"contract"`
	Score           int                       `json:"score"`
	Grade           string                    `json:"grade"`
	Threshold       int                       `json:"threshold"`
	Passed          bool                      `json:"passed"`
	Scanned         int                       `json:"files_scanned"`
	Violations      []JSONViolation           `json:"violations"`
	NewViolations   []JSONViolation           `json:"new_violations,omitempty"`
	BaselineRef     string                    `json:"baseline_ref,omitempty"`
	Suppressed      []JSONSuppressedViolation `json:"suppressed,omitempty"`
	SuppressedCount int                       `json:"suppressed_count"`
	Breakdown       []BreakdownEntry          `json:"breakdown"`
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

// JSONSuppressedViolation represents a suppressed violation in JSON format.
type JSONSuppressedViolation struct {
	Violation   JSONViolation   `json:"violation"`
	Suppression JSONSuppression `json:"suppression"`
}

// JSONSuppression represents a suppression directive in JSON format.
type JSONSuppression struct {
	Rule   string `json:"rule"`
	Reason string `json:"reason,omitempty"`
	File   string `json:"file"`
	Line   int    `json:"line"` // 0 for file-level
	Type   string `json:"type"` // "line", "next-line", "file"
}

// WriteJSON writes the lint result as JSON to the given writer.
func WriteJSON(w io.Writer, path, contractPath, version string, result *detect.DetectionResult, hollowness score.HollownessScore) error {
	report := JSONReport{
		Version:         version,
		Path:            path,
		Contract:        contractPath,
		Score:           hollowness.Score,
		Grade:           hollowness.Grade,
		Threshold:       hollowness.Threshold,
		Passed:          hollowness.Passed,
		Scanned:         result.Scanned,
		SuppressedCount: result.SuppressedCount(),
		BaselineRef:     result.BaselineRef,
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

	// Convert new violations (baseline mode only)
	if len(result.NewViolations) > 0 {
		report.NewViolations = make([]JSONViolation, 0, len(result.NewViolations))
		for _, v := range result.NewViolations {
			report.NewViolations = append(report.NewViolations, JSONViolation{
				Rule:     v.Rule,
				Severity: v.Severity,
				File:     v.File,
				Line:     v.Line,
				Message:  v.Message,
			})
		}
	}

	// Convert suppressed violations
	if len(result.Suppressed) > 0 {
		report.Suppressed = make([]JSONSuppressedViolation, 0, len(result.Suppressed))
		for _, sv := range result.Suppressed {
			report.Suppressed = append(report.Suppressed, JSONSuppressedViolation{
				Violation: JSONViolation{
					Rule:     sv.Violation.Rule,
					Severity: sv.Violation.Severity,
					File:     sv.Violation.File,
					Line:     sv.Violation.Line,
					Message:  sv.Violation.Message,
				},
				Suppression: JSONSuppression{
					Rule:   sv.Suppression.Rule,
					Reason: sv.Suppression.Reason,
					File:   sv.Suppression.File,
					Line:   sv.Suppression.Line,
					Type:   string(sv.Suppression.Type),
				},
			})
		}
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
