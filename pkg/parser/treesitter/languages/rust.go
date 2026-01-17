package languages

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/rust"

	"github.com/zen-systems/hollowcheck/pkg/parser"
	"github.com/zen-systems/hollowcheck/pkg/parser/treesitter"
)

func init() {
	parser.Register(".rs", NewRustParser)
}

// Rust tree-sitter queries
const (
	rustSymbolQuery = `
		(function_item name: (identifier) @func_name) @function
		(impl_item (declaration_list (function_item name: (identifier) @method_name))) @method
		(struct_item name: (type_identifier) @struct_name) @struct
		(enum_item name: (type_identifier) @enum_name) @enum
		(trait_item name: (type_identifier) @trait_name) @trait
		(type_item name: (type_identifier) @type_name) @type
		(const_item name: (identifier) @const_name) @const
	`

	rustFunctionQuery = `
		(function_item name: (identifier) @name) @func
	`

	rustComplexityQuery = `
		(if_expression) @branch
		(else_clause) @branch
		(for_expression) @branch
		(while_expression) @branch
		(loop_expression) @branch
		(match_expression) @branch
		(match_arm) @branch
		(binary_expression operator: "&&") @branch
		(binary_expression operator: "||") @branch
	`
)

// NewRustParser creates a Rust tree-sitter parser.
func NewRustParser() parser.Parser {
	return treesitter.New(treesitter.Config{
		Language:     rust.GetLanguage(),
		LanguageName: "rust",
		SymbolQuery:  rustSymbolQuery,
		SymbolCaptures: []treesitter.SymbolCapture{
			{NameCapture: "func_name", Kind: "function"},
			{NameCapture: "method_name", Kind: "method"},
			{NameCapture: "struct_name", Kind: "type"},
			{NameCapture: "enum_name", Kind: "type"},
			{NameCapture: "trait_name", Kind: "type"},
			{NameCapture: "type_name", Kind: "type"},
			{NameCapture: "const_name", Kind: "const"},
		},
		FunctionQuery:   rustFunctionQuery,
		FunctionCapture: "func",
		FuncNameCapture: "name",
		ComplexityQuery: rustComplexityQuery,
	})
}

// RustLanguage returns the Rust tree-sitter language for external use.
func RustLanguage() *sitter.Language {
	return rust.GetLanguage()
}
