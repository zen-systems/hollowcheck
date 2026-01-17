package detect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zen-systems/hollowcheck/pkg/contract"
)

func testdataPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	return filepath.Join(wd, "..", "..", "testdata")
}

func TestDetectForbiddenPatterns(t *testing.T) {
	testdata := testdataPath(t)

	tests := []struct {
		name           string
		files          []string
		patterns       []contract.ForbiddenPattern
		wantViolations int
		wantErr        bool
	}{
		{
			name:           "no patterns",
			files:          []string{filepath.Join(testdata, "stub.go")},
			patterns:       nil,
			wantViolations: 0,
		},
		{
			name:  "detect TODO",
			files: []string{filepath.Join(testdata, "stub.go")},
			patterns: []contract.ForbiddenPattern{
				{Pattern: "TODO", Description: "work in progress"},
			},
			wantViolations: 1,
		},
		{
			name:  "detect multiple patterns",
			files: []string{filepath.Join(testdata, "stub.go")},
			patterns: []contract.ForbiddenPattern{
				{Pattern: "TODO"},
				{Pattern: "FIXME"},
				{Pattern: "HACK"},
				{Pattern: "XXX"},
			},
			wantViolations: 4,
		},
		{
			name:  "detect panic not implemented",
			files: []string{filepath.Join(testdata, "stub.go")},
			patterns: []contract.ForbiddenPattern{
				{Pattern: `panic\("not implemented"\)`},
			},
			wantViolations: 1,
		},
		{
			name:  "clean file has no patterns",
			files: []string{filepath.Join(testdata, "clean.go")},
			patterns: []contract.ForbiddenPattern{
				{Pattern: "TODO"},
				{Pattern: "FIXME"},
				{Pattern: "HACK"},
			},
			wantViolations: 0,
		},
		{
			name:  "invalid regex",
			files: []string{filepath.Join(testdata, "clean.go")},
			patterns: []contract.ForbiddenPattern{
				{Pattern: "[invalid"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DetectForbiddenPatterns(tt.files, tt.patterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectForbiddenPatterns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if len(result.Violations) != tt.wantViolations {
				t.Errorf("DetectForbiddenPatterns() got %d violations, want %d", len(result.Violations), tt.wantViolations)
				for _, v := range result.Violations {
					t.Logf("  - %s:%d %s", v.File, v.Line, v.Message)
				}
			}
		})
	}
}

func TestDetectMockData(t *testing.T) {
	testdata := testdataPath(t)
	skipFalse := false

	tests := []struct {
		name           string
		files          []string
		config         *contract.MockSignaturesConfig
		wantViolations int
		wantErr        bool
	}{
		{
			name:           "no signatures",
			files:          []string{filepath.Join(testdata, "mock.go")},
			config:         nil,
			wantViolations: 0,
		},
		{
			name:  "detect example.com",
			files: []string{filepath.Join(testdata, "mock.go")},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: `example\.com`, Description: "placeholder domain"},
				},
				SkipTestFiles: &skipFalse,
			},
			wantViolations: 3, // lines 15, 24, 40
		},
		{
			name:  "detect sequential IDs",
			files: []string{filepath.Join(testdata, "mock.go")},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: "12345|00000|11111"},
				},
				SkipTestFiles: &skipFalse,
			},
			wantViolations: 3,
		},
		{
			name:  "detect lorem ipsum",
			files: []string{filepath.Join(testdata, "mock.go")},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: "lorem ipsum"},
				},
				SkipTestFiles: &skipFalse,
			},
			wantViolations: 2, // comment on line 45 and string on line 47
		},
		{
			name:  "detect foo bar placeholder names",
			files: []string{filepath.Join(testdata, "mock.go")},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: `"foo"|"bar"`},
				},
				SkipTestFiles: &skipFalse,
			},
			wantViolations: 2,
		},
		{
			name:  "clean file has no mocks",
			files: []string{filepath.Join(testdata, "clean.go")},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: `example\.com`},
					{Pattern: "12345|00000"},
					{Pattern: "lorem ipsum"},
				},
				SkipTestFiles: &skipFalse,
			},
			wantViolations: 0,
		},
		{
			name:  "detect changeme password",
			files: []string{filepath.Join(testdata, "mock.go")},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: "changeme"},
				},
				SkipTestFiles: &skipFalse,
			},
			wantViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DetectMockData(tt.files, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectMockData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if len(result.Violations) != tt.wantViolations {
				t.Errorf("DetectMockData() got %d violations, want %d", len(result.Violations), tt.wantViolations)
				for _, v := range result.Violations {
					t.Logf("  - %s:%d %s", v.File, v.Line, v.Message)
				}
			}
		})
	}
}

func TestDetectMockDataTestFileAwareness(t *testing.T) {
	// Create a temporary test file with mock data
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "example_test.go")
	regularFile := filepath.Join(tmpDir, "example.go")

	content := []byte(`package example
var url = "https://example.com/api"
`)
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	if err := os.WriteFile(regularFile, content, 0644); err != nil {
		t.Fatalf("writing regular file: %v", err)
	}

	skipTrue := true
	skipFalse := false

	tests := []struct {
		name           string
		files          []string
		config         *contract.MockSignaturesConfig
		wantViolations int
		wantSeverity   string
	}{
		{
			name:  "skip test files by default",
			files: []string{testFile, regularFile},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: `example\.com`},
				},
				// SkipTestFiles defaults to true
			},
			wantViolations: 1, // only regular file
			wantSeverity:   SeverityWarning,
		},
		{
			name:  "skip test files explicitly",
			files: []string{testFile, regularFile},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: `example\.com`},
				},
				SkipTestFiles: &skipTrue,
			},
			wantViolations: 1, // only regular file
			wantSeverity:   SeverityWarning,
		},
		{
			name:  "include test files with reduced severity",
			files: []string{testFile, regularFile},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: `example\.com`},
				},
				SkipTestFiles:    &skipFalse,
				TestFileSeverity: "info",
			},
			wantViolations: 2, // both files
			wantSeverity:   SeverityInfo,
		},
		{
			name:  "include test files with warning severity",
			files: []string{testFile, regularFile},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: `example\.com`},
				},
				SkipTestFiles:    &skipFalse,
				TestFileSeverity: "warning",
			},
			wantViolations: 2, // both files
			wantSeverity:   SeverityWarning,
		},
		{
			name:  "only test file skipped",
			files: []string{testFile},
			config: &contract.MockSignaturesConfig{
				Patterns: []contract.MockSignature{
					{Pattern: `example\.com`},
				},
				SkipTestFiles: &skipTrue,
			},
			wantViolations: 0, // skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DetectMockData(tt.files, tt.config)
			if err != nil {
				t.Fatalf("DetectMockData() error = %v", err)
			}
			if len(result.Violations) != tt.wantViolations {
				t.Errorf("DetectMockData() got %d violations, want %d", len(result.Violations), tt.wantViolations)
				for _, v := range result.Violations {
					t.Logf("  - %s:%d [%s] %s", v.File, v.Line, v.Severity, v.Message)
				}
			}
			// Check severity for test file violations
			if tt.wantSeverity != "" && tt.wantViolations > 0 {
				for _, v := range result.Violations {
					if strings.HasSuffix(v.File, "_test.go") && v.Severity != tt.wantSeverity {
						t.Errorf("Test file violation severity = %q, want %q", v.Severity, tt.wantSeverity)
					}
				}
			}
		})
	}
}

func TestDetectMissingSymbols(t *testing.T) {
	testdata := testdataPath(t)

	tests := []struct {
		name           string
		symbols        []contract.RequiredSymbol
		wantViolations int
	}{
		{
			name:           "no required symbols",
			symbols:        nil,
			wantViolations: 0,
		},
		{
			name: "find existing function",
			symbols: []contract.RequiredSymbol{
				{Name: "ProcessData", Kind: contract.SymbolFunction, File: "stub.go"},
			},
			wantViolations: 0,
		},
		{
			name: "find existing type",
			symbols: []contract.RequiredSymbol{
				{Name: "StubConfig", Kind: contract.SymbolType, File: "stub.go"},
			},
			wantViolations: 0,
		},
		{
			name: "find existing const",
			symbols: []contract.RequiredSymbol{
				{Name: "DefaultTimeout", Kind: contract.SymbolConst, File: "stub.go"},
			},
			wantViolations: 0,
		},
		{
			name: "find existing method",
			symbols: []contract.RequiredSymbol{
				{Name: "Validate", Kind: contract.SymbolMethod, File: "clean.go"},
			},
			wantViolations: 0,
		},
		{
			name: "missing function",
			symbols: []contract.RequiredSymbol{
				{Name: "NonExistent", Kind: contract.SymbolFunction, File: "stub.go"},
			},
			wantViolations: 1,
		},
		{
			name: "wrong kind",
			symbols: []contract.RequiredSymbol{
				{Name: "ProcessData", Kind: contract.SymbolType, File: "stub.go"},
			},
			wantViolations: 1,
		},
		{
			name: "wrong file",
			symbols: []contract.RequiredSymbol{
				{Name: "ProcessData", Kind: contract.SymbolFunction, File: "clean.go"},
			},
			wantViolations: 1,
		},
		{
			name: "multiple symbols mixed",
			symbols: []contract.RequiredSymbol{
				{Name: "Config", Kind: contract.SymbolType, File: "clean.go"},
				{Name: "ProcessItems", Kind: contract.SymbolFunction, File: "clean.go"},
				{Name: "NonExistent", Kind: contract.SymbolFunction, File: "clean.go"},
			},
			wantViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []string{
				filepath.Join(testdata, "stub.go"),
				filepath.Join(testdata, "mock.go"),
				filepath.Join(testdata, "clean.go"),
			}
			result, err := DetectMissingSymbols(testdata, files, tt.symbols)
			if err != nil {
				t.Fatalf("DetectMissingSymbols() error = %v", err)
			}
			if len(result.Violations) != tt.wantViolations {
				t.Errorf("DetectMissingSymbols() got %d violations, want %d", len(result.Violations), tt.wantViolations)
				for _, v := range result.Violations {
					t.Logf("  - %s", v.Message)
				}
			}
		})
	}
}

func TestDetectMissingFiles(t *testing.T) {
	testdata := testdataPath(t)

	tests := []struct {
		name           string
		files          []contract.RequiredFile
		wantViolations int
	}{
		{
			name:           "no required files",
			files:          nil,
			wantViolations: 0,
		},
		{
			name: "existing file",
			files: []contract.RequiredFile{
				{Path: "stub.go", Required: true},
			},
			wantViolations: 0,
		},
		{
			name: "missing file",
			files: []contract.RequiredFile{
				{Path: "nonexistent.go", Required: true},
			},
			wantViolations: 1,
		},
		{
			name: "optional missing file",
			files: []contract.RequiredFile{
				{Path: "nonexistent.go", Required: false},
			},
			wantViolations: 0,
		},
		{
			name: "mixed files",
			files: []contract.RequiredFile{
				{Path: "stub.go", Required: true},
				{Path: "clean.go", Required: true},
				{Path: "missing.go", Required: true},
				{Path: "optional.go", Required: false},
			},
			wantViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DetectMissingFiles(testdata, tt.files)
			if err != nil {
				t.Fatalf("DetectMissingFiles() error = %v", err)
			}
			if len(result.Violations) != tt.wantViolations {
				t.Errorf("DetectMissingFiles() got %d violations, want %d", len(result.Violations), tt.wantViolations)
				for _, v := range result.Violations {
					t.Logf("  - %s", v.Message)
				}
			}
		})
	}
}

func TestDetectLowComplexity(t *testing.T) {
	testdata := testdataPath(t)

	tests := []struct {
		name           string
		requirements   []contract.ComplexityRequirement
		wantViolations int
	}{
		{
			name:           "no requirements",
			requirements:   nil,
			wantViolations: 0,
		},
		{
			name: "stub function low complexity",
			requirements: []contract.ComplexityRequirement{
				{Symbol: "ProcessData", File: "stub.go", MinComplexity: 3},
			},
			wantViolations: 1, // stub returns nil, complexity 1
		},
		{
			name: "clean function adequate complexity",
			requirements: []contract.ComplexityRequirement{
				{Symbol: "ProcessItems", File: "clean.go", MinComplexity: 5},
			},
			wantViolations: 0, // has loops, ifs, should be >= 5
		},
		{
			name: "method complexity",
			requirements: []contract.ComplexityRequirement{
				{Symbol: "Validate", File: "clean.go", MinComplexity: 3},
			},
			wantViolations: 0, // has multiple ifs
		},
		{
			name: "symbol not found",
			requirements: []contract.ComplexityRequirement{
				{Symbol: "NonExistent", File: "clean.go", MinComplexity: 1},
			},
			wantViolations: 1,
		},
		{
			name: "high complexity requirement fails",
			requirements: []contract.ComplexityRequirement{
				{Symbol: "CalculateScore", File: "clean.go", MinComplexity: 20},
			},
			wantViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []string{
				filepath.Join(testdata, "stub.go"),
				filepath.Join(testdata, "mock.go"),
				filepath.Join(testdata, "clean.go"),
			}
			result, err := DetectLowComplexity(testdata, files, tt.requirements)
			if err != nil {
				t.Fatalf("DetectLowComplexity() error = %v", err)
			}
			if len(result.Violations) != tt.wantViolations {
				t.Errorf("DetectLowComplexity() got %d violations, want %d", len(result.Violations), tt.wantViolations)
				for _, v := range result.Violations {
					t.Logf("  - %s", v.Message)
				}
			}
		})
	}
}

func TestCalculateFuncComplexity(t *testing.T) {
	testdata := testdataPath(t)
	files := []string{
		filepath.Join(testdata, "stub.go"),
		filepath.Join(testdata, "clean.go"),
	}

	// Map of expected minimum complexities
	expected := map[string]int{
		"ProcessData":    1, // stub: just returns nil
		"ValidateInput":  1, // stub: just returns true
		"HandleRequest":  1, // stub: just panics
		"ProcessItems":   8, // has for loop, multiple ifs, &&
		"CalculateScore": 6, // has for loop, multiple ifs
		"Validate":       5, // has multiple ifs
	}

	for _, file := range files {
		funcs, err := calculateComplexitiesGo(file)
		if err != nil {
			t.Fatalf("calculateComplexitiesGo(%s) error = %v", file, err)
		}

		for _, f := range funcs {
			if minExpected, ok := expected[f.Name]; ok {
				if f.Complexity < minExpected {
					t.Errorf("Function %s has complexity %d, expected at least %d", f.Name, f.Complexity, minExpected)
				}
			}
		}
	}
}

func TestDetectionResult(t *testing.T) {
	t.Run("Merge", func(t *testing.T) {
		r1 := &DetectionResult{
			Violations: []Violation{{Rule: "test1"}},
			Scanned:    5,
		}
		r2 := &DetectionResult{
			Violations: []Violation{{Rule: "test2"}, {Rule: "test3"}},
			Scanned:    3,
		}
		r1.Merge(r2)

		if len(r1.Violations) != 3 {
			t.Errorf("Merge() got %d violations, want 3", len(r1.Violations))
		}
		if r1.Scanned != 8 {
			t.Errorf("Merge() got scanned=%d, want 8", r1.Scanned)
		}
	})

	t.Run("HasErrors", func(t *testing.T) {
		tests := []struct {
			name       string
			violations []Violation
			want       bool
		}{
			{
				name:       "no violations",
				violations: nil,
				want:       false,
			},
			{
				name: "only warnings",
				violations: []Violation{
					{Severity: SeverityWarning},
				},
				want: false,
			},
			{
				name: "has errors",
				violations: []Violation{
					{Severity: SeverityWarning},
					{Severity: SeverityError},
				},
				want: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := &DetectionResult{Violations: tt.violations}
				if got := r.HasErrors(); got != tt.want {
					t.Errorf("HasErrors() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func TestRunner(t *testing.T) {
	testdata := testdataPath(t)

	skipFalse := false
	c := &contract.Contract{
		Version: "1.0",
		Name:    "test",
		RequiredFiles: []contract.RequiredFile{
			{Path: "stub.go", Required: true},
			{Path: "missing.go", Required: true},
		},
		ForbiddenPatterns: []contract.ForbiddenPattern{
			{Pattern: "TODO"},
		},
		MockSignatures: &contract.MockSignaturesConfig{
			Patterns: []contract.MockSignature{
				{Pattern: `example\.com`},
			},
			SkipTestFiles: &skipFalse,
		},
		RequiredSymbols: []contract.RequiredSymbol{
			{Name: "ProcessData", Kind: contract.SymbolFunction, File: "stub.go"},
			{Name: "NonExistent", Kind: contract.SymbolFunction, File: "stub.go"},
		},
		Complexity: []contract.ComplexityRequirement{
			{Symbol: "ProcessData", File: "stub.go", MinComplexity: 5},
		},
	}

	files := []string{
		filepath.Join(testdata, "stub.go"),
		filepath.Join(testdata, "mock.go"),
	}

	runner := NewRunner(testdata)
	result, err := runner.Run(files, c)
	if err != nil {
		t.Fatalf("Runner.Run() error = %v", err)
	}

	// Should have multiple violations:
	// - 1 missing file (missing.go)
	// - 1 forbidden pattern (TODO in stub.go)
	// - 2 mock data (example.com in mock.go)
	// - 1 missing symbol (NonExistent)
	// - 1 low complexity (ProcessData)
	if len(result.Violations) < 5 {
		t.Errorf("Runner.Run() got %d violations, want at least 5", len(result.Violations))
		for _, v := range result.Violations {
			t.Logf("  - [%s] %s: %s", v.Severity, v.Rule, v.Message)
		}
	}

	if !result.HasErrors() {
		t.Error("Runner.Run() expected to have errors")
	}
}
