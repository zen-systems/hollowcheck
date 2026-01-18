//go:build !simple

package languages

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"

	"github.com/zen-systems/hollowcheck/pkg/parser"
	"github.com/zen-systems/hollowcheck/pkg/parser/treesitter"
)

func init() {
	parser.Register(".c", NewCParser)
	parser.Register(".h", NewCParser)
}

// C tree-sitter queries
const (
	cSymbolQuery = `
		(function_definition declarator: (function_declarator declarator: (identifier) @func_name)) @function
		(declaration declarator: (function_declarator declarator: (identifier) @func_decl_name)) @function_decl
		(struct_specifier name: (type_identifier) @struct_name) @struct
		(enum_specifier name: (type_identifier) @enum_name) @enum
		(type_definition declarator: (type_identifier) @typedef_name) @typedef
	`

	cFunctionQuery = `
		(function_definition declarator: (function_declarator declarator: (identifier) @name)) @func
	`

	cComplexityQuery = `
		(if_statement) @branch
		(else_clause) @branch
		(for_statement) @branch
		(while_statement) @branch
		(do_statement) @branch
		(switch_statement) @branch
		(case_statement) @branch
		(conditional_expression) @branch
		(binary_expression operator: "&&") @branch
		(binary_expression operator: "||") @branch
	`
)

// NewCParser creates a C tree-sitter parser.
func NewCParser() parser.Parser {
	return treesitter.New(treesitter.Config{
		Language:     c.GetLanguage(),
		LanguageName: "c",
		SymbolQuery:  cSymbolQuery,
		SymbolCaptures: []treesitter.SymbolCapture{
			{NameCapture: "func_name", Kind: "function"},
			{NameCapture: "func_decl_name", Kind: "function"},
			{NameCapture: "struct_name", Kind: "type"},
			{NameCapture: "enum_name", Kind: "type"},
			{NameCapture: "typedef_name", Kind: "type"},
		},
		FunctionQuery:   cFunctionQuery,
		FunctionCapture: "func",
		FuncNameCapture: "name",
		ComplexityQuery: cComplexityQuery,
	})
}

// CLanguage returns the C tree-sitter language for external use.
func CLanguage() *sitter.Language {
	return c.GetLanguage()
}
