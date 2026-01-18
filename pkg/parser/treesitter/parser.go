//go:build !simple

// Package treesitter provides a generic tree-sitter based parser.
package treesitter

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/zen-systems/hollowcheck/pkg/parser"
)

// SymbolCapture defines how to extract symbol info from query captures.
type SymbolCapture struct {
	NameCapture string // capture name for symbol name (e.g., "name")
	Kind        string // symbol kind (e.g., "function", "type")
}

// Config holds tree-sitter parser configuration for a language.
type Config struct {
	Language         *sitter.Language
	LanguageName     string
	SymbolQuery      string          // tree-sitter query for finding symbols
	SymbolCaptures   []SymbolCapture // how to map captures to symbols
	ComplexityQuery  string          // query for counting complexity branch points
	FunctionQuery    string          // query for finding function nodes by name
	FunctionCapture  string          // capture name for function node
	FuncNameCapture  string          // capture name for function name within FunctionQuery
}

// Parser implements parser.Parser using tree-sitter.
type Parser struct {
	config Config
	lang   *sitter.Language
	parser *sitter.Parser
}

// New creates a tree-sitter parser with the given configuration.
func New(cfg Config) *Parser {
	p := sitter.NewParser()
	p.SetLanguage(cfg.Language)
	return &Parser{
		config: cfg,
		lang:   cfg.Language,
		parser: p,
	}
}

// Language returns the language name.
func (p *Parser) Language() string {
	return p.config.LanguageName
}

// ParseSymbols extracts symbols from source code using the configured query.
func (p *Parser) ParseSymbols(source []byte) ([]parser.Symbol, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil, fmt.Errorf("parsing source: %w", err)
	}
	defer tree.Close()

	q, err := sitter.NewQuery([]byte(p.config.SymbolQuery), p.lang)
	if err != nil {
		return nil, fmt.Errorf("compiling symbol query: %w", err)
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, tree.RootNode())

	var symbols []parser.Symbol
	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, sc := range p.config.SymbolCaptures {
			sym := p.extractSymbol(match, q, sc, source)
			if sym.Name != "" {
				symbols = append(symbols, sym)
			}
		}
	}

	return symbols, nil
}

// extractSymbol extracts a symbol from a query match.
func (p *Parser) extractSymbol(match *sitter.QueryMatch, q *sitter.Query, sc SymbolCapture, source []byte) parser.Symbol {
	var sym parser.Symbol
	sym.Kind = sc.Kind

	for _, capture := range match.Captures {
		captureName := q.CaptureNameForId(capture.Index)
		if captureName == sc.NameCapture {
			sym.Name = capture.Node.Content(source)
			sym.Line = int(capture.Node.StartPoint().Row) + 1
		}
	}

	return sym
}

// Complexity calculates cyclomatic complexity for a symbol.
func (p *Parser) Complexity(source []byte, symbolName string) (int, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return 0, fmt.Errorf("parsing source: %w", err)
	}
	defer tree.Close()

	// Find the function node
	funcNode, err := p.findFunction(tree.RootNode(), source, symbolName)
	if err != nil {
		return 0, err
	}
	if funcNode == nil {
		return 0, nil // symbol not found
	}

	// Count complexity within the function
	return p.countComplexity(funcNode, source)
}

// findFunction locates a function/method node by name.
func (p *Parser) findFunction(root *sitter.Node, source []byte, name string) (*sitter.Node, error) {
	q, err := sitter.NewQuery([]byte(p.config.FunctionQuery), p.lang)
	if err != nil {
		return nil, fmt.Errorf("compiling function query: %w", err)
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, root)

	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}

		var funcNode *sitter.Node
		var funcName string

		for _, capture := range match.Captures {
			captureName := q.CaptureNameForId(capture.Index)
			if captureName == p.config.FunctionCapture {
				funcNode = capture.Node
			}
			if captureName == p.config.FuncNameCapture {
				funcName = capture.Node.Content(source)
			}
		}

		if funcName == name && funcNode != nil {
			return funcNode, nil
		}
	}

	return nil, nil
}

// countComplexity counts branch points within a node.
func (p *Parser) countComplexity(node *sitter.Node, source []byte) (int, error) {
	if p.config.ComplexityQuery == "" {
		return 1, nil // no query configured, return base complexity
	}

	q, err := sitter.NewQuery([]byte(p.config.ComplexityQuery), p.lang)
	if err != nil {
		return 0, fmt.Errorf("compiling complexity query: %w", err)
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, node)

	complexity := 1 // base complexity
	for {
		_, ok := qc.NextMatch()
		if !ok {
			break
		}
		complexity++
	}

	return complexity, nil
}
