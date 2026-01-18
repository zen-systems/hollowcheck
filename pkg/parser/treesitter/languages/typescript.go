//go:build !simple

package languages

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/zen-systems/hollowcheck/pkg/parser"
	"github.com/zen-systems/hollowcheck/pkg/parser/treesitter"
)

func init() {
	parser.Register(".ts", NewTypeScriptParser)
	parser.Register(".tsx", NewTypeScriptParser)
}

// TypeScript tree-sitter queries
const (
	tsSymbolQuery = `
		(function_declaration name: (identifier) @func_name) @function
		(method_definition name: (property_identifier) @method_name) @method
		(class_declaration name: (type_identifier) @class_name) @class
		(interface_declaration name: (type_identifier) @interface_name) @interface
		(type_alias_declaration name: (type_identifier) @type_name) @type
	`

	tsFunctionQuery = `
		(function_declaration name: (identifier) @name) @func
		(method_definition name: (property_identifier) @name) @func
		(arrow_function) @func
	`

	tsComplexityQuery = `
		(if_statement) @branch
		(for_statement) @branch
		(for_in_statement) @branch
		(while_statement) @branch
		(do_statement) @branch
		(switch_statement) @branch
		(switch_case) @branch
		(catch_clause) @branch
		(ternary_expression) @branch
		(binary_expression operator: "&&") @branch
		(binary_expression operator: "||") @branch
		(binary_expression operator: "??") @branch
	`
)

// NewTypeScriptParser creates a TypeScript tree-sitter parser.
func NewTypeScriptParser() parser.Parser {
	return treesitter.New(treesitter.Config{
		Language:     typescript.GetLanguage(),
		LanguageName: "typescript",
		SymbolQuery:  tsSymbolQuery,
		SymbolCaptures: []treesitter.SymbolCapture{
			{NameCapture: "func_name", Kind: "function"},
			{NameCapture: "method_name", Kind: "method"},
			{NameCapture: "class_name", Kind: "type"},
			{NameCapture: "interface_name", Kind: "type"},
			{NameCapture: "type_name", Kind: "type"},
		},
		FunctionQuery:   tsFunctionQuery,
		FunctionCapture: "func",
		FuncNameCapture: "name",
		ComplexityQuery: tsComplexityQuery,
	})
}

// TypeScriptLanguage returns the TypeScript tree-sitter language for external use.
func TypeScriptLanguage() *sitter.Language {
	return typescript.GetLanguage()
}
