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

func TestTypeScriptParserSymbols(t *testing.T) {
	p, ok := parser.ForExtension(".ts")
	if !ok {
		t.Fatal("TypeScript parser not registered")
	}

	source := []byte(`
function processData(input: string): void {
    console.log(input);
}

class Config {
    name: string = "";

    getName(): string {
        return this.name;
    }
}

interface Handler {
    handle(): void;
}

type Result = string | number;
`)

	symbols, err := p.ParseSymbols(source)
	if err != nil {
		t.Fatalf("ParseSymbols() error = %v", err)
	}

	found := make(map[string]string)
	for _, sym := range symbols {
		found[sym.Name] = sym.Kind
	}

	if kind, ok := found["processData"]; !ok || kind != "function" {
		t.Errorf("expected processData function, got %v", found["processData"])
	}
	if kind, ok := found["Config"]; !ok || kind != "type" {
		t.Errorf("expected Config type, got %v", found["Config"])
	}
	if kind, ok := found["Handler"]; !ok || kind != "type" {
		t.Errorf("expected Handler type, got %v", found["Handler"])
	}
}

func TestJavaScriptParserSymbols(t *testing.T) {
	p, ok := parser.ForExtension(".js")
	if !ok {
		t.Fatal("JavaScript parser not registered")
	}

	source := []byte(`
function processData(input) {
    console.log(input);
}

class Config {
    constructor() {
        this.name = "";
    }

    getName() {
        return this.name;
    }
}
`)

	symbols, err := p.ParseSymbols(source)
	if err != nil {
		t.Fatalf("ParseSymbols() error = %v", err)
	}

	found := make(map[string]string)
	for _, sym := range symbols {
		found[sym.Name] = sym.Kind
	}

	if kind, ok := found["processData"]; !ok || kind != "function" {
		t.Errorf("expected processData function, got %v", found["processData"])
	}
	if kind, ok := found["Config"]; !ok || kind != "type" {
		t.Errorf("expected Config type, got %v", found["Config"])
	}
}

func TestRustParserSymbols(t *testing.T) {
	p, ok := parser.ForExtension(".rs")
	if !ok {
		t.Fatal("Rust parser not registered")
	}

	source := []byte(`
fn process_data(input: &str) -> Result<(), Error> {
    Ok(())
}

struct Config {
    name: String,
}

enum Status {
    Active,
    Inactive,
}

const MAX_SIZE: usize = 100;
`)

	symbols, err := p.ParseSymbols(source)
	if err != nil {
		t.Fatalf("ParseSymbols() error = %v", err)
	}

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
	if kind, ok := found["Status"]; !ok || kind != "type" {
		t.Errorf("expected Status type, got %v", found["Status"])
	}
}

func TestCParserSymbols(t *testing.T) {
	p, ok := parser.ForExtension(".c")
	if !ok {
		t.Fatal("C parser not registered")
	}

	source := []byte(`
int process_data(const char* input) {
    return 0;
}

struct Config {
    char* name;
};

enum Status {
    ACTIVE,
    INACTIVE
};
`)

	symbols, err := p.ParseSymbols(source)
	if err != nil {
		t.Fatalf("ParseSymbols() error = %v", err)
	}

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
}

func TestCppParserSymbols(t *testing.T) {
	p, ok := parser.ForExtension(".cpp")
	if !ok {
		t.Fatal("C++ parser not registered")
	}

	source := []byte(`
int processData(const std::string& input) {
    return 0;
}

class Config {
public:
    std::string name;
};

struct Data {
    int value;
};

enum class Status {
    Active,
    Inactive
};
`)

	symbols, err := p.ParseSymbols(source)
	if err != nil {
		t.Fatalf("ParseSymbols() error = %v", err)
	}

	found := make(map[string]string)
	for _, sym := range symbols {
		found[sym.Name] = sym.Kind
	}

	if kind, ok := found["processData"]; !ok || kind != "function" {
		t.Errorf("expected processData function, got %v", found["processData"])
	}
	if kind, ok := found["Config"]; !ok || kind != "type" {
		t.Errorf("expected Config type, got %v", found["Config"])
	}
}

func TestJavaParserSymbols(t *testing.T) {
	p, ok := parser.ForExtension(".java")
	if !ok {
		t.Fatal("Java parser not registered")
	}

	source := []byte(`
public class Config {
    private String name;

    public Config() {
        this.name = "";
    }

    public String getName() {
        return this.name;
    }
}

interface Handler {
    void handle();
}

enum Status {
    ACTIVE,
    INACTIVE
}
`)

	symbols, err := p.ParseSymbols(source)
	if err != nil {
		t.Fatalf("ParseSymbols() error = %v", err)
	}

	found := make(map[string]string)
	for _, sym := range symbols {
		found[sym.Name] = sym.Kind
	}

	if kind, ok := found["Config"]; !ok || kind != "type" {
		t.Errorf("expected Config type, got %v", found["Config"])
	}
	if kind, ok := found["getName"]; !ok || kind != "method" {
		t.Errorf("expected getName method, got %v", found["getName"])
	}
}

func TestKotlinParserSymbols(t *testing.T) {
	p, ok := parser.ForExtension(".kt")
	if !ok {
		t.Fatal("Kotlin parser not registered")
	}

	source := []byte(`
fun processData(input: String): Unit {
    println(input)
}

class Config {
    var name: String = ""
}

object Singleton {
    val instance = "single"
}
`)

	symbols, err := p.ParseSymbols(source)
	if err != nil {
		t.Fatalf("ParseSymbols() error = %v", err)
	}

	found := make(map[string]string)
	for _, sym := range symbols {
		found[sym.Name] = sym.Kind
	}

	if kind, ok := found["processData"]; !ok || kind != "function" {
		t.Errorf("expected processData function, got %v", found["processData"])
	}
	if kind, ok := found["Config"]; !ok || kind != "type" {
		t.Errorf("expected Config type, got %v", found["Config"])
	}
}

func TestAllExtensionsRegistered(t *testing.T) {
	extensions := []struct {
		ext  string
		lang string
	}{
		{".go", "go"},
		{".py", "python"},
		{".ts", "typescript"},
		{".tsx", "typescript"},
		{".js", "javascript"},
		{".jsx", "javascript"},
		{".rs", "rust"},
		{".c", "c"},
		{".h", "c"},
		{".cpp", "cpp"},
		{".cc", "cpp"},
		{".cxx", "cpp"},
		{".hpp", "cpp"},
		{".java", "java"},
		{".kt", "kotlin"},
		{".kts", "kotlin"},
	}

	for _, tc := range extensions {
		t.Run(tc.ext, func(t *testing.T) {
			p, ok := parser.ForExtension(tc.ext)
			if !ok {
				t.Fatalf("expected %s extension to be registered", tc.ext)
			}
			if p.Language() != tc.lang {
				t.Errorf("Language() = %q, want %q", p.Language(), tc.lang)
			}
		})
	}
}
