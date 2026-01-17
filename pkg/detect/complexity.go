package detect

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"

	"github.com/zen-systems/hollowcheck/pkg/contract"
	"github.com/zen-systems/hollowcheck/pkg/parser"
)

// funcComplexity holds complexity information for a function.
type funcComplexity struct {
	Name       string
	Complexity int
	File       string
	Line       int
	IsMethod   bool
}

// DetectLowComplexity checks that functions meet minimum complexity requirements.
func DetectLowComplexity(baseDir string, files []string, requirements []contract.ComplexityRequirement) (*DetectionResult, error) {
	result := &DetectionResult{}

	if len(requirements) == 0 {
		return result, nil
	}

	// Build a map of function complexities by file
	funcsByFile := make(map[string][]funcComplexity)

	for _, file := range files {
		ext := filepath.Ext(file)
		var funcs []funcComplexity
		var err error

		if ext == ".go" {
			// Use native go/ast for Go files
			funcs, err = calculateComplexitiesGo(file)
		} else if p, ok := parser.ForExtension(ext); ok {
			// Use parser registry for other supported languages
			funcs, err = calculateComplexitiesWithParser(file, p)
		} else {
			continue // unsupported extension
		}

		if err != nil {
			return nil, fmt.Errorf("calculating complexity for %s: %w", file, err)
		}

		relPath, err := filepath.Rel(baseDir, file)
		if err != nil {
			relPath = file
		}
		funcsByFile[relPath] = funcs
		result.Scanned++
	}

	// Check each complexity requirement
	for _, req := range requirements {
		found := false
		var actualComplexity int

		if req.File != "" {
			// Look in specific file
			funcs, ok := funcsByFile[req.File]
			if ok {
				for _, f := range funcs {
					if f.Name == req.Symbol {
						found = true
						actualComplexity = f.Complexity
						break
					}
				}
			}
		} else {
			// Look in any file
			for _, funcs := range funcsByFile {
				for _, f := range funcs {
					if f.Name == req.Symbol {
						found = true
						actualComplexity = f.Complexity
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
				file = "(any file)"
			}
			result.AddViolation(Violation{
				Rule:     RuleLowComplexity,
				Message:  fmt.Sprintf("symbol %q not found for complexity check", req.Symbol),
				File:     file,
				Line:     0,
				Severity: SeverityError,
			})
			continue
		}

		if actualComplexity < req.MinComplexity {
			file := req.File
			if file == "" {
				file = "(found in codebase)"
			}
			result.AddViolation(Violation{
				Rule:     RuleLowComplexity,
				Message:  fmt.Sprintf("symbol %q has complexity %d, minimum required is %d", req.Symbol, actualComplexity, req.MinComplexity),
				File:     file,
				Line:     0,
				Severity: SeverityWarning,
			})
		}
	}

	return result, nil
}

// calculateComplexitiesGo parses a Go file and calculates cyclomatic complexity for all functions.
func calculateComplexitiesGo(filePath string) ([]funcComplexity, error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, err
	}

	var funcs []funcComplexity

	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		complexity := calculateFuncComplexity(fn)
		funcs = append(funcs, funcComplexity{
			Name:       fn.Name.Name,
			Complexity: complexity,
			File:       filePath,
			Line:       fset.Position(fn.Pos()).Line,
			IsMethod:   fn.Recv != nil,
		})

		return true
	})

	return funcs, nil
}

// calculateFuncComplexity calculates the cyclomatic complexity of a function.
// Cyclomatic complexity = E - N + 2P where E=edges, N=nodes, P=connected components
// Simplified: start at 1 and add 1 for each decision point.
func calculateFuncComplexity(fn *ast.FuncDecl) int {
	if fn.Body == nil {
		return 1
	}

	complexity := 1

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			// Each case in a switch adds a decision point
			// (except default, but we count it anyway for simplicity)
			if node.List != nil { // not default
				complexity++
			}
		case *ast.CommClause:
			// Each case in a select adds a decision point
			if node.Comm != nil { // not default
				complexity++
			}
		case *ast.BinaryExpr:
			// Logical operators add decision points
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		}
		return true
	})

	return complexity
}

// calculateComplexitiesWithParser uses the parser interface for non-Go files.
func calculateComplexitiesWithParser(filePath string, p parser.Parser) ([]funcComplexity, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Get all symbols to find functions
	symbols, err := p.ParseSymbols(source)
	if err != nil {
		return nil, err
	}

	var funcs []funcComplexity
	for _, sym := range symbols {
		if sym.Kind != "function" && sym.Kind != "method" {
			continue
		}

		complexity, err := p.Complexity(source, sym.Name)
		if err != nil {
			return nil, err
		}

		funcs = append(funcs, funcComplexity{
			Name:       sym.Name,
			Complexity: complexity,
			File:       filePath,
			Line:       sym.Line,
			IsMethod:   sym.Kind == "method",
		})
	}

	return funcs, nil
}
