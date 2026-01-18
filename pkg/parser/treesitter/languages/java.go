//go:build !simple

package languages

import (
	"github.com/smacker/go-tree-sitter/java"

	"github.com/zen-systems/hollowcheck/pkg/parser"
	"github.com/zen-systems/hollowcheck/pkg/parser/treesitter"
)

func init() {
	parser.Register(".java", NewJavaParser)
}

// Java tree-sitter queries
const (
	javaSymbolQuery = `
		(class_declaration name: (identifier) @class_name) @class
		(interface_declaration name: (identifier) @interface_name) @interface
		(enum_declaration name: (identifier) @enum_name) @enum
		(method_declaration name: (identifier) @method_name) @method
	`

	javaFunctionQuery = `
		(method_declaration name: (identifier) @name) @func
		(constructor_declaration name: (identifier) @name) @func
	`

	javaComplexityQuery = `
		(if_statement) @branch
		(else_clause) @branch
		(for_statement) @branch
		(enhanced_for_statement) @branch
		(while_statement) @branch
		(do_statement) @branch
		(switch_expression) @branch
		(switch_block_statement_group) @branch
		(catch_clause) @branch
		(ternary_expression) @branch
		(binary_expression operator: "&&") @branch
		(binary_expression operator: "||") @branch
	`
)

// NewJavaParser creates a Java tree-sitter parser.
func NewJavaParser() parser.Parser {
	return treesitter.New(treesitter.Config{
		Language:     java.GetLanguage(),
		LanguageName: "java",
		SymbolQuery:  javaSymbolQuery,
		SymbolCaptures: []treesitter.SymbolCapture{
			{NameCapture: "class_name", Kind: "type"},
			{NameCapture: "interface_name", Kind: "type"},
			{NameCapture: "enum_name", Kind: "type"},
			{NameCapture: "method_name", Kind: "method"},
		},
		FunctionQuery:   javaFunctionQuery,
		FunctionCapture: "func",
		FuncNameCapture: "name",
		ComplexityQuery: javaComplexityQuery,
	})
}
