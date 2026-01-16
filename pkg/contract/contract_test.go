package contract

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "minimal valid contract",
			input: `
version: "1.0"
name: "test-contract"
`,
			wantErr: false,
		},
		{
			name: "full contract",
			input: `
version: "1.0"
name: "full-contract"
description: "A complete contract example"
required_files:
  - path: "main.go"
    required: true
required_symbols:
  - name: "ProcessData"
    kind: "function"
    file: "processor.go"
forbidden_patterns:
  - pattern: "TODO"
    description: "No TODOs allowed"
mock_signatures:
  patterns:
    - pattern: "example\\.com"
      description: "No example.com URLs"
complexity:
  - symbol: "ProcessData"
    min_complexity: 5
required_tests:
  - name: "TestProcessData"
coverage_threshold: 80.0
`,
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			input:   `version: [invalid`,
			wantErr: true,
		},
		{
			name: "unknown field with strict parsing",
			input: `
version: "1.0"
name: "test"
unknown_field: "value"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	threshold80 := 80.0
	thresholdNeg := -1.0
	threshold150 := 150.0

	tests := []struct {
		name       string
		contract   *Contract
		wantErr    bool
		errContain string
	}{
		{
			name: "valid minimal contract",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
			},
			wantErr: false,
		},
		{
			name: "missing version",
			contract: &Contract{
				Name: "test",
			},
			wantErr:    true,
			errContain: "version",
		},
		{
			name: "missing name",
			contract: &Contract{
				Version: "1.0",
			},
			wantErr:    true,
			errContain: "name",
		},
		{
			name: "empty required file path",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
				RequiredFiles: []RequiredFile{
					{Path: "", Required: true},
				},
			},
			wantErr:    true,
			errContain: "required_files[0].path",
		},
		{
			name: "invalid symbol kind",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
				RequiredSymbols: []RequiredSymbol{
					{Name: "Foo", Kind: "invalid", File: "foo.go"},
				},
			},
			wantErr:    true,
			errContain: "invalid kind",
		},
		{
			name: "empty symbol name",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
				RequiredSymbols: []RequiredSymbol{
					{Name: "", Kind: SymbolFunction, File: "foo.go"},
				},
			},
			wantErr:    true,
			errContain: "required_symbols[0].name",
		},
		{
			name: "empty symbol file",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
				RequiredSymbols: []RequiredSymbol{
					{Name: "Foo", Kind: SymbolFunction, File: ""},
				},
			},
			wantErr:    true,
			errContain: "required_symbols[0].file",
		},
		{
			name: "invalid forbidden pattern regex",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
				ForbiddenPatterns: []ForbiddenPattern{
					{Pattern: "[invalid"},
				},
			},
			wantErr:    true,
			errContain: "invalid regex",
		},
		{
			name: "empty forbidden pattern",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
				ForbiddenPatterns: []ForbiddenPattern{
					{Pattern: ""},
				},
			},
			wantErr:    true,
			errContain: "forbidden_patterns[0].pattern",
		},
		{
			name: "invalid mock signature regex",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
				MockSignatures: &MockSignaturesConfig{
					Patterns: []MockSignature{
						{Pattern: "(unclosed"},
					},
				},
			},
			wantErr:    true,
			errContain: "invalid regex",
		},
		{
			name: "complexity with zero min",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
				Complexity: []ComplexityRequirement{
					{Symbol: "Foo", MinComplexity: 0},
				},
			},
			wantErr:    true,
			errContain: "min_complexity must be at least 1",
		},
		{
			name: "empty complexity symbol",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
				Complexity: []ComplexityRequirement{
					{Symbol: "", MinComplexity: 5},
				},
			},
			wantErr:    true,
			errContain: "complexity[0].symbol",
		},
		{
			name: "empty required test name",
			contract: &Contract{
				Version: "1.0",
				Name:    "test",
				RequiredTests: []RequiredTest{
					{Name: ""},
				},
			},
			wantErr:    true,
			errContain: "required_tests[0].name",
		},
		{
			name: "valid coverage threshold",
			contract: &Contract{
				Version:           "1.0",
				Name:              "test",
				CoverageThreshold: &threshold80,
			},
			wantErr: false,
		},
		{
			name: "negative coverage threshold",
			contract: &Contract{
				Version:           "1.0",
				Name:              "test",
				CoverageThreshold: &thresholdNeg,
			},
			wantErr:    true,
			errContain: "coverage_threshold",
		},
		{
			name: "coverage threshold over 100",
			contract: &Contract{
				Version:           "1.0",
				Name:              "test",
				CoverageThreshold: &threshold150,
			},
			wantErr:    true,
			errContain: "coverage_threshold",
		},
		{
			name: "valid full contract",
			contract: &Contract{
				Version:     "1.0",
				Name:        "full-test",
				Description: "A full test contract",
				RequiredFiles: []RequiredFile{
					{Path: "main.go", Required: true},
					{Path: "README.md", Required: false},
				},
				RequiredSymbols: []RequiredSymbol{
					{Name: "ProcessData", Kind: SymbolFunction, File: "processor.go"},
					{Name: "Config", Kind: SymbolType, File: "config.go"},
				},
				ForbiddenPatterns: []ForbiddenPattern{
					{Pattern: `TODO`, Description: "No TODOs"},
					{Pattern: `FIXME`, Description: "No FIXMEs"},
				},
				MockSignatures: &MockSignaturesConfig{
					Patterns: []MockSignature{
						{Pattern: `example\.com`, Description: "No example.com"},
					},
				},
				Complexity: []ComplexityRequirement{
					{Symbol: "ProcessData", MinComplexity: 5},
				},
				RequiredTests: []RequiredTest{
					{Name: "TestProcessData", File: "processor_test.go"},
				},
				CoverageThreshold: &threshold80,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.contract)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContain != "" {
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errContain)
				}
			}
		})
	}
}

func TestParseAndValidate(t *testing.T) {
	input := `
version: "1.0"
name: "integration-test"
description: "Tests parse and validate together"
required_files:
  - path: "main.go"
    required: true
required_symbols:
  - name: "Run"
    kind: "function"
    file: "main.go"
forbidden_patterns:
  - pattern: "TODO|FIXME|HACK"
    description: "No work-in-progress markers"
  - pattern: "NotImplementedError"
    description: "No stub implementations"
mock_signatures:
  patterns:
    - pattern: "test@example\\.com"
      description: "No test emails"
    - pattern: "12345|00000"
      description: "No sequential/fake IDs"
    - pattern: "lorem ipsum"
      description: "No placeholder text"
  skip_test_files: true
complexity:
  - symbol: "Run"
    file: "main.go"
    min_complexity: 3
required_tests:
  - name: "TestRun"
    file: "main_test.go"
coverage_threshold: 75.5
`

	c, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if err := Validate(c); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Verify parsed values
	if c.Version != "1.0" {
		t.Errorf("Version = %q, want %q", c.Version, "1.0")
	}
	if c.Name != "integration-test" {
		t.Errorf("Name = %q, want %q", c.Name, "integration-test")
	}
	if len(c.RequiredFiles) != 1 {
		t.Errorf("RequiredFiles count = %d, want 1", len(c.RequiredFiles))
	}
	if len(c.ForbiddenPatterns) != 2 {
		t.Errorf("ForbiddenPatterns count = %d, want 2", len(c.ForbiddenPatterns))
	}
	if c.MockSignatures == nil || len(c.MockSignatures.Patterns) != 3 {
		count := 0
		if c.MockSignatures != nil {
			count = len(c.MockSignatures.Patterns)
		}
		t.Errorf("MockSignatures.Patterns count = %d, want 3", count)
	}
	if c.CoverageThreshold == nil || *c.CoverageThreshold != 75.5 {
		t.Errorf("CoverageThreshold = %v, want 75.5", c.CoverageThreshold)
	}
}

func TestValidationErrorsFormat(t *testing.T) {
	errs := ValidationErrors{
		{Field: "version", Message: "required field is empty"},
		{Field: "name", Message: "required field is empty"},
	}

	errStr := errs.Error()
	if !strings.Contains(errStr, "version: required field is empty") {
		t.Errorf("Error string missing version error: %s", errStr)
	}
	if !strings.Contains(errStr, "name: required field is empty") {
		t.Errorf("Error string missing name error: %s", errStr)
	}
}
