package detect

import (
	"testing"
)

func TestViolationKey(t *testing.T) {
	v := Violation{
		Rule:    "forbidden_pattern",
		File:    "/path/to/file.go",
		Line:    42,
		Message: "found TODO",
	}

	key := ViolationKey(v)
	expected := "forbidden_pattern|/path/to/file.go|found TODO"
	if key != expected {
		t.Errorf("ViolationKey() = %q, want %q", key, expected)
	}
}

func TestViolationsMatch(t *testing.T) {
	tests := []struct {
		name  string
		a     Violation
		b     Violation
		match bool
	}{
		{
			name: "identical violations match",
			a: Violation{
				Rule:    "forbidden_pattern",
				File:    "/path/to/file.go",
				Line:    42,
				Message: "found TODO",
			},
			b: Violation{
				Rule:    "forbidden_pattern",
				File:    "/path/to/file.go",
				Line:    42,
				Message: "found TODO",
			},
			match: true,
		},
		{
			name: "different line numbers still match",
			a: Violation{
				Rule:    "forbidden_pattern",
				File:    "/path/to/file.go",
				Line:    42,
				Message: "found TODO",
			},
			b: Violation{
				Rule:    "forbidden_pattern",
				File:    "/path/to/file.go",
				Line:    100, // Different line
				Message: "found TODO",
			},
			match: true,
		},
		{
			name: "different rules don't match",
			a: Violation{
				Rule:    "forbidden_pattern",
				File:    "/path/to/file.go",
				Line:    42,
				Message: "found TODO",
			},
			b: Violation{
				Rule:    "mock_data",
				File:    "/path/to/file.go",
				Line:    42,
				Message: "found TODO",
			},
			match: false,
		},
		{
			name: "different files don't match",
			a: Violation{
				Rule:    "forbidden_pattern",
				File:    "/path/to/file.go",
				Line:    42,
				Message: "found TODO",
			},
			b: Violation{
				Rule:    "forbidden_pattern",
				File:    "/path/to/other.go",
				Line:    42,
				Message: "found TODO",
			},
			match: false,
		},
		{
			name: "different messages don't match",
			a: Violation{
				Rule:    "forbidden_pattern",
				File:    "/path/to/file.go",
				Line:    42,
				Message: "found TODO",
			},
			b: Violation{
				Rule:    "forbidden_pattern",
				File:    "/path/to/file.go",
				Line:    42,
				Message: "found FIXME",
			},
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ViolationsMatch(tt.a, tt.b); got != tt.match {
				t.Errorf("ViolationsMatch() = %v, want %v", got, tt.match)
			}
		})
	}
}

func TestDetectionResult_IsBaselineMode(t *testing.T) {
	tests := []struct {
		name   string
		result DetectionResult
		want   bool
	}{
		{
			name:   "no baseline ref",
			result: DetectionResult{},
			want:   false,
		},
		{
			name: "with baseline ref",
			result: DetectionResult{
				BaselineRef: "main",
			},
			want: true,
		},
		{
			name: "empty baseline ref",
			result: DetectionResult{
				BaselineRef: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsBaselineMode(); got != tt.want {
				t.Errorf("IsBaselineMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectionResult_NewViolationCount(t *testing.T) {
	result := DetectionResult{
		NewViolations: []Violation{
			{Rule: "r1"},
			{Rule: "r2"},
			{Rule: "r3"},
		},
	}

	if got := result.NewViolationCount(); got != 3 {
		t.Errorf("NewViolationCount() = %d, want 3", got)
	}
}
