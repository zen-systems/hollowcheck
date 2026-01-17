// Package parser provides language-agnostic code parsing interfaces.
package parser

// Symbol represents a named code element (function, method, type, const).
type Symbol struct {
	Name string
	Kind string // "function", "method", "type", "const"
	File string
	Line int
}

// Parser parses source code to extract symbols and complexity metrics.
type Parser interface {
	// ParseSymbols extracts all symbols from source code.
	ParseSymbols(source []byte) ([]Symbol, error)

	// Complexity calculates cyclomatic complexity for a named symbol.
	// Returns 0 if symbol not found.
	Complexity(source []byte, symbolName string) (int, error)

	// Language returns the language this parser handles (e.g., "go", "python").
	Language() string
}
