package score

import (
	"testing"

	"github.com/zen-systems/hollowcheck/pkg/detect"
)

func TestCalculateForNewViolations(t *testing.T) {
	tests := []struct {
		name      string
		result    *detect.DetectionResult
		threshold int
		wantScore int
		wantPass  bool
	}{
		{
			name: "no new violations passes",
			result: &detect.DetectionResult{
				Violations:    []detect.Violation{{Rule: "forbidden_pattern"}},
				NewViolations: []detect.Violation{},
			},
			threshold: 0,
			wantScore: 0,
			wantPass:  true,
		},
		{
			name: "new violation fails with threshold 0",
			result: &detect.DetectionResult{
				Violations: []detect.Violation{{Rule: "forbidden_pattern"}},
				NewViolations: []detect.Violation{
					{Rule: "forbidden_pattern", Message: "new issue"},
				},
			},
			threshold: 0,
			wantScore: PointsForbiddenPattern,
			wantPass:  false,
		},
		{
			name: "new violation passes with higher threshold",
			result: &detect.DetectionResult{
				Violations: []detect.Violation{{Rule: "forbidden_pattern"}},
				NewViolations: []detect.Violation{
					{Rule: "forbidden_pattern", Message: "new issue"},
				},
			},
			threshold: 20,
			wantScore: PointsForbiddenPattern,
			wantPass:  true,
		},
		{
			name: "multiple new violations accumulate",
			result: &detect.DetectionResult{
				NewViolations: []detect.Violation{
					{Rule: "forbidden_pattern"},
					{Rule: "mock_data"},
					{Rule: "missing_test"},
				},
			},
			threshold: 10,
			wantScore: PointsForbiddenPattern + PointsMockData + PointsMissingTest, // 10 + 3 + 5 = 18
			wantPass:  false,
		},
		{
			name: "score caps at 100",
			result: &detect.DetectionResult{
				NewViolations: []detect.Violation{
					{Rule: "missing_file"},
					{Rule: "missing_file"},
					{Rule: "missing_file"},
					{Rule: "missing_file"},
					{Rule: "missing_file"},
					{Rule: "missing_file"}, // 6 * 20 = 120
				},
			},
			threshold: 50,
			wantScore: 100,
			wantPass:  false,
		},
		{
			name: "negative threshold treated as 0",
			result: &detect.DetectionResult{
				NewViolations: []detect.Violation{},
			},
			threshold: -10,
			wantScore: 0,
			wantPass:  true,
		},
		{
			name: "existing violations don't count",
			result: &detect.DetectionResult{
				Violations: []detect.Violation{
					{Rule: "missing_file"},
					{Rule: "missing_symbol"},
					{Rule: "forbidden_pattern"},
				},
				NewViolations: []detect.Violation{},
			},
			threshold: 0,
			wantScore: 0,
			wantPass:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateForNewViolations(tt.result, tt.threshold)

			if got.Score != tt.wantScore {
				t.Errorf("CalculateForNewViolations().Score = %d, want %d", got.Score, tt.wantScore)
			}

			if got.Passed != tt.wantPass {
				t.Errorf("CalculateForNewViolations().Passed = %v, want %v", got.Passed, tt.wantPass)
			}
		})
	}
}

func TestCalculateForNewViolations_Breakdown(t *testing.T) {
	result := &detect.DetectionResult{
		NewViolations: []detect.Violation{
			{Rule: "forbidden_pattern"},
			{Rule: "forbidden_pattern"},
			{Rule: "mock_data"},
		},
	}

	score := CalculateForNewViolations(result, 50)

	// Check breakdown
	if points, ok := score.Breakdown["forbidden_pattern"]; !ok || points != 20 {
		t.Errorf("Breakdown[forbidden_pattern] = %d, want 20", points)
	}

	if points, ok := score.Breakdown["mock_data"]; !ok || points != 3 {
		t.Errorf("Breakdown[mock_data] = %d, want 3", points)
	}

	// Check violation count
	if count := score.ViolationCount("forbidden_pattern"); count != 2 {
		t.Errorf("ViolationCount(forbidden_pattern) = %d, want 2", count)
	}
}
