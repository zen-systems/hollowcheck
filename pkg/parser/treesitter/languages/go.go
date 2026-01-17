package languages

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"

	"github.com/zen-systems/hollowcheck/pkg/parser"
	"github.com/zen-systems/hollowcheck/pkg/parser/treesitter"
)

func init() {
	parser.Register(".go", NewGoParser)
}

// Go tree-sitter queries
const (
	goSymbolQuery = `
		(function_declaration name: (identifier) @func_name) @function
		(method_declaration name: (field_identifier) @method_name) @method
		(type_declaration (type_spec name: (type_identifier) @type_name)) @type
		(const_declaration (const_spec name: (identifier) @const_name)) @const
	`

	goFunctionQuery = `
		(function_declaration name: (identifier) @name) @func
		(method_declaration name: (field_identifier) @name) @func
	`

	goComplexityQuery = `
		(if_statement) @branch
		(for_statement) @branch
		(expression_switch_statement) @branch
		(type_switch_statement) @branch
		(select_statement) @branch
		(communication_case) @branch
		(expression_case) @branch
		(type_case) @branch
		(binary_expression operator: "&&") @branch
		(binary_expression operator: "||") @branch
	`
)

// NewGoParser creates a Go tree-sitter parser.
func NewGoParser() parser.Parser {
	return treesitter.New(treesitter.Config{
		Language:     golang.GetLanguage(),
		LanguageName: "go",
		SymbolQuery:  goSymbolQuery,
		SymbolCaptures: []treesitter.SymbolCapture{
			{NameCapture: "func_name", Kind: "function"},
			{NameCapture: "method_name", Kind: "method"},
			{NameCapture: "type_name", Kind: "type"},
			{NameCapture: "const_name", Kind: "const"},
		},
		FunctionQuery:   goFunctionQuery,
		FunctionCapture: "func",
		FuncNameCapture: "name",
		ComplexityQuery: goComplexityQuery,
	})
}

// GoLanguage returns the Go tree-sitter language for external use.
func GoLanguage() *sitter.Language {
	return golang.GetLanguage()
}
