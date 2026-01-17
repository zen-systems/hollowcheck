package detect

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/zen-systems/hollowcheck/pkg/contract"
	"github.com/zen-systems/hollowcheck/pkg/parser"
)

// symbolInfo holds information about a found symbol.
type symbolInfo struct {
	Name string
	Kind contract.SymbolKind
	File string
	Line int
}

// DetectMissingSymbols checks that all required symbols exist in the codebase.
func DetectMissingSymbols(baseDir string, files []string, symbols []contract.RequiredSymbol) (*DetectionResult, error) {
	result := &DetectionResult{}

	if len(symbols) == 0 {
		return result, nil
	}

	// Build a map of found symbols by file
	foundSymbols := make(map[string][]symbolInfo)

	for _, file := range files {
		ext := filepath.Ext(file)
		var syms []symbolInfo
		var err error

		if ext == ".go" {
			// Use native go/ast for Go files
			syms, err = extractSymbolsGo(file)
		} else if p, ok := parser.ForExtension(ext); ok {
			// Use parser registry for other supported languages
			syms, err = extractSymbolsWithParser(file, p)
		} else {
			continue // unsupported extension
		}

		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", file, err)
		}

		// Normalize file path relative to baseDir for matching
		relPath, err := filepath.Rel(baseDir, file)
		if err != nil {
			relPath = file
		}
		foundSymbols[relPath] = syms
		result.Scanned++
	}

	// Check each required symbol
	for _, req := range symbols {
		found := false
		syms, ok := foundSymbols[req.File]
		if ok {
			for _, sym := range syms {
				if sym.Name == req.Name && sym.Kind == req.Kind {
					found = true
					break
				}
			}
		}

		if !found {
			result.AddViolation(Violation{
				Rule:     RuleMissingSymbol,
				Message:  fmt.Sprintf("required %s %q not found", req.Kind, req.Name),
				File:     req.File,
				Line:     0,
				Severity: SeverityError,
			})
		}
	}

	return result, nil
}

// DetectMissingTests checks that all required test functions exist.
func DetectMissingTests(baseDir string, files []string, tests []contract.RequiredTest) (*DetectionResult, error) {
	result := &DetectionResult{}

	if len(tests) == 0 {
		return result, nil
	}

	// Build a map of found test functions by file
	foundTests := make(map[string][]string)

	// Only parse Go test files (test detection is Go-specific for now)
	for _, file := range files {
		if !strings.HasSuffix(file, "_test.go") {
			continue
		}

		syms, err := extractSymbolsGo(file)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", file, err)
		}

		relPath, err := filepath.Rel(baseDir, file)
		if err != nil {
			relPath = file
		}

		var testNames []string
		for _, sym := range syms {
			if sym.Kind == contract.SymbolFunction && strings.HasPrefix(sym.Name, "Test") {
				testNames = append(testNames, sym.Name)
			}
		}
		foundTests[relPath] = testNames
	}

	// Check each required test
	for _, req := range tests {
		found := false

		if req.File != "" {
			// Look in specific file
			testNames, ok := foundTests[req.File]
			if ok {
				for _, name := range testNames {
					if name == req.Name {
						found = true
						break
					}
				}
			}
		} else {
			// Look in any test file
			for _, testNames := range foundTests {
				for _, name := range testNames {
					if name == req.Name {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}

		if !found {
			file := req.File
			if file == "" {
				file = "(any test file)"
			}
			result.AddViolation(Violation{
				Rule:     RuleMissingTest,
				Message:  fmt.Sprintf("required test %q not found", req.Name),
				File:     file,
				Line:     0,
				Severity: SeverityError,
			})
		}
	}

	return result, nil
}

// extractSymbolsGo parses a Go file using go/ast and extracts all symbol definitions.
func extractSymbolsGo(filePath string) ([]symbolInfo, error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, err
	}

	var symbols []symbolInfo

	ast.Inspect(f, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			kind := contract.SymbolFunction
			if decl.Recv != nil {
				kind = contract.SymbolMethod
			}
			symbols = append(symbols, symbolInfo{
				Name: decl.Name.Name,
				Kind: kind,
				File: filePath,
				Line: fset.Position(decl.Pos()).Line,
			})

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					symbols = append(symbols, symbolInfo{
						Name: s.Name.Name,
						Kind: contract.SymbolType,
						File: filePath,
						Line: fset.Position(s.Pos()).Line,
					})
				case *ast.ValueSpec:
					if decl.Tok == token.CONST {
						for _, name := range s.Names {
							symbols = append(symbols, symbolInfo{
								Name: name.Name,
								Kind: contract.SymbolConst,
								File: filePath,
								Line: fset.Position(name.Pos()).Line,
							})
						}
					}
				}
			}
		}
		return true
	})

	return symbols, nil
}

// extractSymbolsWithParser uses the parser interface for non-Go files.
func extractSymbolsWithParser(filePath string, p parser.Parser) ([]symbolInfo, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	syms, err := p.ParseSymbols(source)
	if err != nil {
		return nil, err
	}

	result := make([]symbolInfo, len(syms))
	for i, sym := range syms {
		result[i] = symbolInfo{
			Name: sym.Name,
			Kind: contract.SymbolKind(sym.Kind),
			File: filePath,
			Line: sym.Line,
		}
	}

	return result, nil
}
