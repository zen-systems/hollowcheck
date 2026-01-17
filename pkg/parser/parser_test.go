package parser_test

import (
	"testing"

	"github.com/zen-systems/hollowcheck/pkg/parser"
	_ "github.com/zen-systems/hollowcheck/pkg/parser/treesitter/languages"
)

func TestRegistry(t *testing.T) {
	t.Run("Go parser registered", func(t *testing.T) {
		p, ok := parser.ForExtension(".go")
		if !ok {
			t.Fatal("expected .go extension to be registered")
		}
		if p.Language() != "go" {
			t.Errorf("Language() = %q, want %q", p.Language(), "go")
		}
	})

	t.Run("Python parser registered", func(t *testing.T) {
		p, ok := parser.ForExtension(".py")
		if !ok {
			t.Fatal("expected .py extension to be registered")
		}
		if p.Language() != "python" {
			t.Errorf("Language() = %q, want %q", p.Language(), "python")
		}
	})

	t.Run("unsupported extension returns false", func(t *testing.T) {
		_, ok := parser.ForExtension(".xyz")
		if ok {
			t.Error("expected .xyz extension to not be registered")
		}
	})

	t.Run("SupportedExtensions includes registered languages", func(t *testing.T) {
		exts := parser.SupportedExtensions()
		hasGo := false
		hasPy := false
		for _, ext := range exts {
			if ext == ".go" {
				hasGo = true
			}
			if ext == ".py" {
				hasPy = true
			}
		}
		if !hasGo {
			t.Error("SupportedExtensions() missing .go")
		}
		if !hasPy {
			t.Error("SupportedExtensions() missing .py")
		}
	})
}

func TestGoParserSymbols(t *testing.T) {
	p, ok := parser.ForExtension(".go")
	if !ok {
		t.Fatal("Go parser not registered")
	}

	source := []byte(`package example

func ProcessData(input string) error {
	return nil
}

type Config struct {
	Name string
}

const DefaultTimeout = 30
`)

	symbols, err := p.ParseSymbols(source)
	if err != nil {
		t.Fatalf("ParseSymbols() error = %v", err)
	}

	// Check we found expected symbols
	found := make(map[string]string)
	for _, sym := range symbols {
		found[sym.Name] = sym.Kind
	}

	if kind, ok := found["ProcessData"]; !ok || kind != "function" {
		t.Errorf("expected ProcessData function, got %v", found["ProcessData"])
	}
	if kind, ok := found["Config"]; !ok || kind != "type" {
		t.Errorf("expected Config type, got %v", found["Config"])
	}
	if kind, ok := found["DefaultTimeout"]; !ok || kind != "const" {
		t.Errorf("expected DefaultTimeout const, got %v", found["DefaultTimeout"])
	}
}

func TestGoParserComplexity(t *testing.T) {
	p, ok := parser.ForExtension(".go")
	if !ok {
		t.Fatal("Go parser not registered")
	}

	source := []byte(`package example

func Simple() {}

func WithBranches(x int) int {
	if x > 0 {
		return x
	} else if x < 0 {
		return -x
	}
	return 0
}
`)

	// Simple function should have complexity 1
	c, err := p.Complexity(source, "Simple")
	if err != nil {
		t.Fatalf("Complexity(Simple) error = %v", err)
	}
	if c != 1 {
		t.Errorf("Complexity(Simple) = %d, want 1", c)
	}

	// WithBranches should have complexity >= 3 (base + 2 if statements)
	c, err = p.Complexity(source, "WithBranches")
	if err != nil {
		t.Fatalf("Complexity(WithBranches) error = %v", err)
	}
	if c < 3 {
		t.Errorf("Complexity(WithBranches) = %d, want >= 3", c)
	}

	// Non-existent function returns 0
	c, err = p.Complexity(source, "NonExistent")
	if err != nil {
		t.Fatalf("Complexity(NonExistent) error = %v", err)
	}
	if c != 0 {
		t.Errorf("Complexity(NonExistent) = %d, want 0", c)
	}
}

func TestPythonParserSymbols(t *testing.T) {
	p, ok := parser.ForExtension(".py")
	if !ok {
		t.Fatal("Python parser not registered")
	}

	source := []byte(`
def process_data(input):
    return None

class Config:
    def __init__(self):
        self.name = ""

def helper():
    pass
`)

	symbols, err := p.ParseSymbols(source)
	if err != nil {
		t.Fatalf("ParseSymbols() error = %v", err)
	}

	// Check we found expected symbols
	found := make(map[string]string)
	for _, sym := range symbols {
		found[sym.Name] = sym.Kind
	}

	if kind, ok := found["process_data"]; !ok || kind != "function" {
		t.Errorf("expected process_data function, got %v", found["process_data"])
	}
	if kind, ok := found["Config"]; !ok || kind != "type" {
		t.Errorf("expected Config type, got %v", found["Config"])
	}
	if kind, ok := found["helper"]; !ok || kind != "function" {
		t.Errorf("expected helper function, got %v", found["helper"])
	}
}
