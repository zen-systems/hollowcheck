package languages

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"

	"github.com/zen-systems/hollowcheck/pkg/parser"
	"github.com/zen-systems/hollowcheck/pkg/parser/treesitter"
)

func init() {
	parser.Register(".js", NewJavaScriptParser)
	parser.Register(".jsx", NewJavaScriptParser)
}

// JavaScript tree-sitter queries
const (
	jsSymbolQuery = `
		(function_declaration name: (identifier) @func_name) @function
		(method_definition name: (property_identifier) @method_name) @method
		(class_declaration name: (identifier) @class_name) @class
	`

	jsFunctionQuery = `
		(function_declaration name: (identifier) @name) @func
		(method_definition name: (property_identifier) @name) @func
		(arrow_function) @func
	`

	jsComplexityQuery = `
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

// NewJavaScriptParser creates a JavaScript tree-sitter parser.
func NewJavaScriptParser() parser.Parser {
	return treesitter.New(treesitter.Config{
		Language:     javascript.GetLanguage(),
		LanguageName: "javascript",
		SymbolQuery:  jsSymbolQuery,
		SymbolCaptures: []treesitter.SymbolCapture{
			{NameCapture: "func_name", Kind: "function"},
			{NameCapture: "method_name", Kind: "method"},
			{NameCapture: "class_name", Kind: "type"},
		},
		FunctionQuery:   jsFunctionQuery,
		FunctionCapture: "func",
		FuncNameCapture: "name",
		ComplexityQuery: jsComplexityQuery,
	})
}

// JavaScriptLanguage returns the JavaScript tree-sitter language for external use.
func JavaScriptLanguage() *sitter.Language {
	return javascript.GetLanguage()
}
