package languages

import (
	"github.com/smacker/go-tree-sitter/kotlin"

	"github.com/zen-systems/hollowcheck/pkg/parser"
	"github.com/zen-systems/hollowcheck/pkg/parser/treesitter"
)

func init() {
	parser.Register(".kt", NewKotlinParser)
	parser.Register(".kts", NewKotlinParser)
}

// Kotlin tree-sitter queries
const (
	kotlinSymbolQuery = `
		(function_declaration (simple_identifier) @func_name) @function
		(class_declaration (type_identifier) @class_name) @class
		(object_declaration (type_identifier) @object_name) @object
	`

	kotlinFunctionQuery = `
		(function_declaration (simple_identifier) @name) @func
	`

	kotlinComplexityQuery = `
		(if_expression) @branch
		(when_expression) @branch
		(when_entry) @branch
		(for_statement) @branch
		(while_statement) @branch
		(do_while_statement) @branch
		(catch_block) @branch
		(conjunction_expression) @branch
		(disjunction_expression) @branch
		(elvis_expression) @branch
	`
)

// NewKotlinParser creates a Kotlin tree-sitter parser.
func NewKotlinParser() parser.Parser {
	return treesitter.New(treesitter.Config{
		Language:     kotlin.GetLanguage(),
		LanguageName: "kotlin",
		SymbolQuery:  kotlinSymbolQuery,
		SymbolCaptures: []treesitter.SymbolCapture{
			{NameCapture: "func_name", Kind: "function"},
			{NameCapture: "class_name", Kind: "type"},
			{NameCapture: "object_name", Kind: "type"},
		},
		FunctionQuery:   kotlinFunctionQuery,
		FunctionCapture: "func",
		FuncNameCapture: "name",
		ComplexityQuery: kotlinComplexityQuery,
	})
}
