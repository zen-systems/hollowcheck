package score

import (
	"testing"

	"github.com/zen-systems/hollowcheck/pkg/contract"
	"github.com/zen-systems/hollowcheck/pkg/detect"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name           string
		violations     []detect.Violation
		wantScore      int
		wantGrade      string
		wantPassed     bool
		wantThreshold  int
	}{
		{
			name:          "clean code - no violations",
			violations:    nil,
			wantScore:     0,
			wantGrade:     "A",
			wantPassed:    true,
			wantThreshold: DefaultThreshold,
		},
		{
			name:          "empty violations slice",
			violations:    []detect.Violation{},
			wantScore:     0,
			wantGrade:     "A",
			wantPassed:    true,
			wantThreshold: DefaultThreshold,
		},
		{
			name: "single mock data warning",
			violations: []detect.Violation{
				{Rule: detect.RuleMockData, Severity: detect.SeverityWarning},
			},
			wantScore:     3,
			wantGrade:     "A",
			wantPassed:    true,
			wantThreshold: DefaultThreshold,
		},
		{
			name: "single TODO pattern",
			violations: []detect.Violation{
				{Rule: detect.RuleForbiddenPattern, Severity: detect.SeverityError},
			},
			wantScore:     10,
			wantGrade:     "A",
			wantPassed:    true,
			wantThreshold: DefaultThreshold,
		},
		{
			name: "single missing file - critical",
			violations: []detect.Violation{
				{Rule: detect.RuleMissingFile, Severity: detect.SeverityError},
			},
			wantScore:     20,
			wantGrade:     "B",
			wantPassed:    true,
			wantThreshold: DefaultThreshold,
		},
		{
			name: "moderately hollow - multiple issues",
			violations: []detect.Violation{
				{Rule: detect.RuleForbiddenPattern}, // 10
				{Rule: detect.RuleForbiddenPattern}, // 10
				{Rule: detect.RuleMockData},         // 3
				{Rule: detect.RuleMockData},         // 3
				{Rule: detect.RuleMissingTest},      // 5
			},
			wantScore:     31,
			wantGrade:     "C",
			wantPassed:    false,
			wantThreshold: DefaultThreshold,
		},
		{
			name: "very hollow - missing files and symbols",
			violations: []detect.Violation{
				{Rule: detect.RuleMissingFile},   // 20
				{Rule: detect.RuleMissingFile},   // 20
				{Rule: detect.RuleMissingSymbol}, // 15
			},
			wantScore:     55,
			wantGrade:     "D",
			wantPassed:    false,
			wantThreshold: DefaultThreshold,
		},
		{
			name: "extremely hollow - many critical issues",
			violations: []detect.Violation{
				{Rule: detect.RuleMissingFile},      // 20
				{Rule: detect.RuleMissingFile},      // 20
				{Rule: detect.RuleMissingSymbol},    // 15
				{Rule: detect.RuleMissingSymbol},    // 15
				{Rule: detect.RuleForbiddenPattern}, // 10
				{Rule: detect.RuleForbiddenPattern}, // 10
			},
			wantScore:     90,
			wantGrade:     "F",
			wantPassed:    false,
			wantThreshold: DefaultThreshold,
		},
		{
			name: "capped at 100",
			violations: []detect.Violation{
				{Rule: detect.RuleMissingFile},   // 20
				{Rule: detect.RuleMissingFile},   // 20
				{Rule: detect.RuleMissingFile},   // 20
				{Rule: detect.RuleMissingFile},   // 20
				{Rule: detect.RuleMissingFile},   // 20
				{Rule: detect.RuleMissingFile},   // 20 = 120 total, capped to 100
				{Rule: detect.RuleMissingSymbol}, // 15
			},
			wantScore:     100,
			wantGrade:     "F",
			wantPassed:    false,
			wantThreshold: DefaultThreshold,
		},
		{
			name: "grade boundary A/B at 10",
			violations: []detect.Violation{
				{Rule: detect.RuleForbiddenPattern}, // 10 exactly
			},
			wantScore:     10,
			wantGrade:     "A",
			wantPassed:    true,
			wantThreshold: DefaultThreshold,
		},
		{
			name: "grade boundary B at 11",
			violations: []detect.Violation{
				{Rule: detect.RuleForbiddenPattern}, // 10
				{Rule: detect.RuleMockData},         // 3 = 13
			},
			wantScore:     13,
			wantGrade:     "B",
			wantPassed:    true,
			wantThreshold: DefaultThreshold,
		},
		{
			name: "all violation types",
			violations: []detect.Violation{
				{Rule: detect.RuleMissingFile},      // 20
				{Rule: detect.RuleMissingSymbol},    // 15
				{Rule: detect.RuleForbiddenPattern}, // 10
				{Rule: detect.RuleLowComplexity},    // 10
				{Rule: detect.RuleMissingTest},      // 5
				{Rule: detect.RuleMockData},         // 3 = 63
			},
			wantScore:     63,
			wantGrade:     "D",
			wantPassed:    false,
			wantThreshold: DefaultThreshold,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &detect.DetectionResult{
				Violations: tt.violations,
			}
			score := Calculate(result, nil)

			if score.Score != tt.wantScore {
				t.Errorf("Score = %d, want %d", score.Score, tt.wantScore)
			}
			if score.Grade != tt.wantGrade {
				t.Errorf("Grade = %q, want %q", score.Grade, tt.wantGrade)
			}
			if score.Passed != tt.wantPassed {
				t.Errorf("Passed = %v, want %v", score.Passed, tt.wantPassed)
			}
			if score.Threshold != tt.wantThreshold {
				t.Errorf("Threshold = %d, want %d", score.Threshold, tt.wantThreshold)
			}
		})
	}
}

func TestCalculateWithThreshold(t *testing.T) {
	tests := []struct {
		name       string
		violations []detect.Violation
		threshold  int
		wantPassed bool
	}{
		{
			name:       "clean code passes any threshold",
			violations: nil,
			threshold:  0,
			wantPassed: true,
		},
		{
			name: "threshold edge case - exactly at threshold passes",
			violations: []detect.Violation{
				{Rule: detect.RuleForbiddenPattern}, // 10
			},
			threshold:  10,
			wantPassed: true,
		},
		{
			name: "threshold edge case - one over fails",
			violations: []detect.Violation{
				{Rule: detect.RuleForbiddenPattern}, // 10
				{Rule: detect.RuleMockData},         // 3 = 13
			},
			threshold:  10,
			wantPassed: false,
		},
		{
			name: "strict threshold of 0",
			violations: []detect.Violation{
				{Rule: detect.RuleMockData}, // 3
			},
			threshold:  0,
			wantPassed: false,
		},
		{
			name: "lenient threshold of 50",
			violations: []detect.Violation{
				{Rule: detect.RuleMissingFile},   // 20
				{Rule: detect.RuleMissingSymbol}, // 15
				{Rule: detect.RuleMockData},      // 3 = 38
			},
			threshold:  50,
			wantPassed: true,
		},
		{
			name: "threshold of 100 passes everything",
			violations: []detect.Violation{
				{Rule: detect.RuleMissingFile}, // 20
				{Rule: detect.RuleMissingFile}, // 20
				{Rule: detect.RuleMissingFile}, // 20
				{Rule: detect.RuleMissingFile}, // 20
				{Rule: detect.RuleMissingFile}, // 20 = 100
			},
			threshold:  100,
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &detect.DetectionResult{
				Violations: tt.violations,
			}
			score := CalculateWithThreshold(result, tt.threshold)

			if score.Passed != tt.wantPassed {
				t.Errorf("Passed = %v, want %v (score=%d, threshold=%d)",
					score.Passed, tt.wantPassed, score.Score, tt.threshold)
			}
			if score.Threshold != tt.threshold {
				t.Errorf("Threshold = %d, want %d", score.Threshold, tt.threshold)
			}
		})
	}
}

func TestBreakdown(t *testing.T) {
	violations := []detect.Violation{
		{Rule: detect.RuleMissingFile},
		{Rule: detect.RuleMissingFile},
		{Rule: detect.RuleForbiddenPattern},
		{Rule: detect.RuleForbiddenPattern},
		{Rule: detect.RuleForbiddenPattern},
		{Rule: detect.RuleMockData},
	}

	result := &detect.DetectionResult{Violations: violations}
	score := Calculate(result, nil)

	expectedBreakdown := map[string]int{
		detect.RuleMissingFile:      40, // 2 * 20
		detect.RuleForbiddenPattern: 30, // 3 * 10
		detect.RuleMockData:         3,  // 1 * 3
	}

	for rule, wantPoints := range expectedBreakdown {
		if gotPoints := score.Breakdown[rule]; gotPoints != wantPoints {
			t.Errorf("Breakdown[%q] = %d, want %d", rule, gotPoints, wantPoints)
		}
	}

	// Total should be 73
	if score.Score != 73 {
		t.Errorf("Score = %d, want 73", score.Score)
	}
}

func TestGradeCalculation(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{0, "A"},
		{5, "A"},
		{10, "A"},
		{11, "B"},
		{20, "B"},
		{25, "B"},
		{26, "C"},
		{40, "C"},
		{50, "C"},
		{51, "D"},
		{60, "D"},
		{75, "D"},
		{76, "F"},
		{90, "F"},
		{100, "F"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := calculateGrade(tt.score)
			if got != tt.want {
				t.Errorf("calculateGrade(%d) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

func TestTotalPoints(t *testing.T) {
	tests := []struct {
		name       string
		violations []detect.Violation
		wantTotal  int
		wantScore  int
	}{
		{
			name:       "no violations",
			violations: nil,
			wantTotal:  0,
			wantScore:  0,
		},
		{
			name: "under cap",
			violations: []detect.Violation{
				{Rule: detect.RuleMissingFile},   // 20
				{Rule: detect.RuleMissingSymbol}, // 15
			},
			wantTotal: 35,
			wantScore: 35,
		},
		{
			name: "over cap - total differs from score",
			violations: []detect.Violation{
				{Rule: detect.RuleMissingFile}, // 20
				{Rule: detect.RuleMissingFile}, // 20
				{Rule: detect.RuleMissingFile}, // 20
				{Rule: detect.RuleMissingFile}, // 20
				{Rule: detect.RuleMissingFile}, // 20
				{Rule: detect.RuleMissingFile}, // 20 = 120
			},
			wantTotal: 120,
			wantScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &detect.DetectionResult{Violations: tt.violations}
			score := Calculate(result, nil)

			if got := score.TotalPoints(); got != tt.wantTotal {
				t.Errorf("TotalPoints() = %d, want %d", got, tt.wantTotal)
			}
			if score.Score != tt.wantScore {
				t.Errorf("Score = %d, want %d", score.Score, tt.wantScore)
			}
		})
	}
}

func TestViolationCount(t *testing.T) {
	violations := []detect.Violation{
		{Rule: detect.RuleMissingFile},
		{Rule: detect.RuleMissingFile},
		{Rule: detect.RuleMissingFile},
		{Rule: detect.RuleForbiddenPattern},
		{Rule: detect.RuleForbiddenPattern},
		{Rule: detect.RuleMockData},
	}

	result := &detect.DetectionResult{Violations: violations}
	score := Calculate(result, nil)

	tests := []struct {
		rule  string
		count int
	}{
		{detect.RuleMissingFile, 3},
		{detect.RuleForbiddenPattern, 2},
		{detect.RuleMockData, 1},
		{detect.RuleMissingSymbol, 0},
		{detect.RuleLowComplexity, 0},
		{detect.RuleMissingTest, 0},
	}

	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			if got := score.ViolationCount(tt.rule); got != tt.count {
				t.Errorf("ViolationCount(%q) = %d, want %d", tt.rule, got, tt.count)
			}
		})
	}
}

func TestCalculateWithContract(t *testing.T) {
	// Test that passing a contract doesn't break anything
	// (threshold handling via contract could be extended later)
	c := &contract.Contract{
		Version: "1.0",
		Name:    "test",
	}

	violations := []detect.Violation{
		{Rule: detect.RuleForbiddenPattern},
	}

	result := &detect.DetectionResult{Violations: violations}
	score := Calculate(result, c)

	if score.Score != 10 {
		t.Errorf("Score = %d, want 10", score.Score)
	}
	if score.Threshold != DefaultThreshold {
		t.Errorf("Threshold = %d, want %d", score.Threshold, DefaultThreshold)
	}
}

func TestPointWeights(t *testing.T) {
	// Verify documented point weights
	tests := []struct {
		rule   string
		points int
	}{
		{detect.RuleMissingFile, 20},
		{detect.RuleMissingSymbol, 15},
		{detect.RuleForbiddenPattern, 10},
		{detect.RuleLowComplexity, 10},
		{detect.RuleMissingTest, 5},
		{detect.RuleMockData, 3},
		{"unknown_rule", 0},
	}

	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			if got := getPoints(tt.rule); got != tt.points {
				t.Errorf("getPoints(%q) = %d, want %d", tt.rule, got, tt.points)
			}
		})
	}
}
