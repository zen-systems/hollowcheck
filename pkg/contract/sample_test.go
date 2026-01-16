package contract

import (
	"path/filepath"
	"testing"
)

func TestParseSampleYAML(t *testing.T) {
	path := filepath.Join("..", "..", "contracts", "examples", "sample.yaml")
	c, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if err := Validate(c); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Verify expected values from sample.yaml
	if c.Name != "web-api-service" {
		t.Errorf("Name = %q, want %q", c.Name, "web-api-service")
	}
	if c.Version != "1.0" {
		t.Errorf("Version = %q, want %q", c.Version, "1.0")
	}
	if len(c.RequiredFiles) != 7 {
		t.Errorf("RequiredFiles count = %d, want 7", len(c.RequiredFiles))
	}
	if len(c.RequiredSymbols) != 7 {
		t.Errorf("RequiredSymbols count = %d, want 7", len(c.RequiredSymbols))
	}
	if len(c.ForbiddenPatterns) != 8 {
		t.Errorf("ForbiddenPatterns count = %d, want 8", len(c.ForbiddenPatterns))
	}
	if c.MockSignatures == nil || len(c.MockSignatures.Patterns) != 10 {
		count := 0
		if c.MockSignatures != nil {
			count = len(c.MockSignatures.Patterns)
		}
		t.Errorf("MockSignatures.Patterns count = %d, want 10", count)
	}
	if len(c.Complexity) != 3 {
		t.Errorf("Complexity count = %d, want 3", len(c.Complexity))
	}
	if len(c.RequiredTests) != 5 {
		t.Errorf("RequiredTests count = %d, want 5", len(c.RequiredTests))
	}
	if c.CoverageThreshold == nil || *c.CoverageThreshold != 80.0 {
		t.Errorf("CoverageThreshold = %v, want 80.0", c.CoverageThreshold)
	}
}
