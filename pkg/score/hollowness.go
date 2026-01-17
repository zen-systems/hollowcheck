// Package score calculates hollowness scores from detection results.
package score

import (
	"github.com/zen-systems/hollowcheck/pkg/contract"
	"github.com/zen-systems/hollowcheck/pkg/detect"
)

// Point weights for each violation type.
const (
	PointsMissingFile      = 20 // critical
	PointsMissingSymbol    = 15 // critical
	PointsForbiddenPattern = 10 // error
	PointsLowComplexity    = 10 // error
	PointsMissingTest      = 5  // warning
	PointsMockData         = 3  // warning

	// Prose-specific point weights
	PointsFillerPhrase        = 2 // warning
	PointsWeaselWord          = 3 // warning
	PointsLowDensity          = 5 // warning
	PointsRepetitiveStructure = 3 // warning
	PointsMiddleSag           = 8 // error
	PointsWeakTransition      = 2 // info
	PointsProseDefault        = 2 // default for prose issues
)

// DefaultThreshold is used when the contract doesn't specify one.
const DefaultThreshold = 25

// Grade thresholds.
const (
	GradeAMax = 10
	GradeBMax = 25
	GradeCMax = 50
	GradeDMax = 75
)

// HollownessScore represents the calculated hollowness score.
type HollownessScore struct {
	Score     int            // 0-100, higher = more hollow
	Grade     string         // "A" (0-10), "B" (11-25), "C" (26-50), "D" (51-75), "F" (76-100)
	Breakdown map[string]int // points by category
	Passed    bool           // true if score <= threshold
	Threshold int            // from contract or default 25
}

// Calculate computes the hollowness score from detection results.
func Calculate(result *detect.DetectionResult, c *contract.Contract) HollownessScore {
	breakdown := make(map[string]int)
	totalPoints := 0

	// Count violations by rule and calculate points
	for _, v := range result.Violations {
		points := getPoints(v.Rule)
		breakdown[v.Rule] += points
		totalPoints += points
	}

	// Cap at 100
	score := totalPoints
	if score > 100 {
		score = 100
	}

	// Determine threshold
	threshold := DefaultThreshold
	if c != nil && c.CoverageThreshold != nil {
		// Note: CoverageThreshold in contract is for coverage, but we repurpose
		// the concept. For hollowness, we might want a separate field.
		// For now, we use default unless contract specifies a hollowness threshold.
		// This could be extended to add a HollownessThreshold field to Contract.
		threshold = DefaultThreshold
	}

	return HollownessScore{
		Score:     score,
		Grade:     calculateGrade(score),
		Breakdown: breakdown,
		Passed:    score <= threshold,
		Threshold: threshold,
	}
}

// CalculateWithThreshold computes the hollowness score with a custom threshold.
func CalculateWithThreshold(result *detect.DetectionResult, threshold int) HollownessScore {
	breakdown := make(map[string]int)
	totalPoints := 0

	// Count violations by rule and calculate points
	for _, v := range result.Violations {
		points := getPoints(v.Rule)
		breakdown[v.Rule] += points
		totalPoints += points
	}

	// Cap at 100
	score := totalPoints
	if score > 100 {
		score = 100
	}

	return HollownessScore{
		Score:     score,
		Grade:     calculateGrade(score),
		Breakdown: breakdown,
		Passed:    score <= threshold,
		Threshold: threshold,
	}
}

// getPoints returns the point weight for a violation rule.
func getPoints(rule string) int {
	switch rule {
	case detect.RuleMissingFile:
		return PointsMissingFile
	case detect.RuleMissingSymbol:
		return PointsMissingSymbol
	case detect.RuleForbiddenPattern:
		return PointsForbiddenPattern
	case detect.RuleLowComplexity:
		return PointsLowComplexity
	case detect.RuleMissingTest:
		return PointsMissingTest
	case detect.RuleMockData:
		return PointsMockData
	// Prose rules
	case "filler_phrase":
		return PointsFillerPhrase
	case "weasel_word":
		return PointsWeaselWord
	case "low_density":
		return PointsLowDensity
	case "prose_repetitive_opener":
		return PointsRepetitiveStructure
	case "prose_middle_sag":
		return PointsMiddleSag
	case "prose_weak_transition":
		return PointsWeakTransition
	default:
		// For any unknown prose rules, return a default
		if len(rule) > 6 && rule[:6] == "prose_" {
			return PointsProseDefault
		}
		return 0
	}
}

// calculateGrade determines the letter grade from the score.
func calculateGrade(score int) string {
	switch {
	case score <= GradeAMax:
		return "A"
	case score <= GradeBMax:
		return "B"
	case score <= GradeCMax:
		return "C"
	case score <= GradeDMax:
		return "D"
	default:
		return "F"
	}
}

// TotalPoints returns the sum of all breakdown points (before capping).
func (s *HollownessScore) TotalPoints() int {
	total := 0
	for _, points := range s.Breakdown {
		total += points
	}
	return total
}

// ViolationCount returns the number of violations for a given rule.
func (s *HollownessScore) ViolationCount(rule string) int {
	points, ok := s.Breakdown[rule]
	if !ok {
		return 0
	}
	perViolation := getPoints(rule)
	if perViolation == 0 {
		return 0
	}
	return points / perViolation
}

// CalculateForNewViolations computes a score based only on new violations (baseline mode).
// The threshold defaults to 0 if not specified (any new violation fails).
func CalculateForNewViolations(result *detect.DetectionResult, threshold int) HollownessScore {
	breakdown := make(map[string]int)
	totalPoints := 0

	// Only count new violations
	for _, v := range result.NewViolations {
		points := getPoints(v.Rule)
		breakdown[v.Rule] += points
		totalPoints += points
	}

	// Cap at 100
	score := totalPoints
	if score > 100 {
		score = 100
	}

	// For baseline mode, default threshold is 0 (any new violation fails)
	if threshold < 0 {
		threshold = 0
	}

	return HollownessScore{
		Score:     score,
		Grade:     calculateGrade(score),
		Breakdown: breakdown,
		Passed:    score <= threshold,
		Threshold: threshold,
	}
}
