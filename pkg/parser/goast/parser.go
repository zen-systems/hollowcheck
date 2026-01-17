// Package goast provides a Go parser using the standard go/ast package.
// This is the reference implementation and fallback for Go files.
package goast

import (
	"go/ast"
	goparser "go/parser"
	"go/token"

	"github.com/zen-systems/hollowcheck/pkg/parser"
)

// Parser implements parser.Parser using Go's standard AST package.
type Parser struct{}

// New creates a new Go AST parser.
func New() parser.Parser {
	return &Parser{}
}

// Language returns "go".
func (p *Parser) Language() string {
	return "go"
}

// ParseSymbols extracts symbols from Go source code.
func (p *Parser) ParseSymbols(source []byte) ([]parser.Symbol, error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, "", source, 0)
	if err != nil {
		return nil, err
	}

	var symbols []parser.Symbol

	ast.Inspect(f, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			kind := "function"
			if decl.Recv != nil {
				kind = "method"
			}
			symbols = append(symbols, parser.Symbol{
				Name: decl.Name.Name,
				Kind: kind,
				Line: fset.Position(decl.Pos()).Line,
			})

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					symbols = append(symbols, parser.Symbol{
						Name: s.Name.Name,
						Kind: "type",
						Line: fset.Position(s.Pos()).Line,
					})
				case *ast.ValueSpec:
					if decl.Tok == token.CONST {
						for _, name := range s.Names {
							symbols = append(symbols, parser.Symbol{
								Name: name.Name,
								Kind: "const",
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

// Complexity calculates cyclomatic complexity for a function.
func (p *Parser) Complexity(source []byte, symbolName string) (int, error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, "", source, 0)
	if err != nil {
		return 0, err
	}

	var complexity int
	var found bool

	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		if fn.Name.Name == symbolName {
			found = true
			complexity = calculateComplexity(fn)
			return false // stop searching
		}
		return true
	})

	if !found {
		return 0, nil
	}
	return complexity, nil
}

// calculateComplexity calculates cyclomatic complexity for a function.
func calculateComplexity(fn *ast.FuncDecl) int {
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
			if node.List != nil { // not default
				complexity++
			}
		case *ast.CommClause:
			if node.Comm != nil { // not default
				complexity++
			}
		case *ast.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		}
		return true
	})

	return complexity
}
