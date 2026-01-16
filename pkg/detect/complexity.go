package detect

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/zen-systems/hollowcheck/pkg/contract"
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

	// Only parse Go files
	for _, file := range files {
		if !strings.HasSuffix(file, ".go") {
			continue
		}

		funcs, err := calculateComplexities(file)
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

// calculateComplexities parses a Go file and calculates cyclomatic complexity for all functions.
func calculateComplexities(filePath string) ([]funcComplexity, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
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
