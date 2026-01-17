package languages

import (
	"github.com/smacker/go-tree-sitter/cpp"

	"github.com/zen-systems/hollowcheck/pkg/parser"
	"github.com/zen-systems/hollowcheck/pkg/parser/treesitter"
)

func init() {
	parser.Register(".cpp", NewCppParser)
	parser.Register(".cc", NewCppParser)
	parser.Register(".cxx", NewCppParser)
	parser.Register(".hpp", NewCppParser)
}

// C++ tree-sitter queries
const (
	cppSymbolQuery = `
		(function_definition declarator: (function_declarator declarator: (identifier) @func_name)) @function
		(function_definition declarator: (function_declarator declarator: (qualified_identifier name: (identifier) @method_name))) @method
		(class_specifier name: (type_identifier) @class_name) @class
		(struct_specifier name: (type_identifier) @struct_name) @struct
		(enum_specifier name: (type_identifier) @enum_name) @enum
		(type_definition declarator: (type_identifier) @typedef_name) @typedef
	`

	cppFunctionQuery = `
		(function_definition declarator: (function_declarator declarator: (identifier) @name)) @func
		(function_definition declarator: (function_declarator declarator: (qualified_identifier name: (identifier) @name))) @func
	`

	cppComplexityQuery = `
		(if_statement) @branch
		(else_clause) @branch
		(for_statement) @branch
		(for_range_loop) @branch
		(while_statement) @branch
		(do_statement) @branch
		(switch_statement) @branch
		(case_statement) @branch
		(catch_clause) @branch
		(conditional_expression) @branch
		(binary_expression operator: "&&") @branch
		(binary_expression operator: "||") @branch
	`
)

// NewCppParser creates a C++ tree-sitter parser.
func NewCppParser() parser.Parser {
	return treesitter.New(treesitter.Config{
		Language:     cpp.GetLanguage(),
		LanguageName: "cpp",
		SymbolQuery:  cppSymbolQuery,
		SymbolCaptures: []treesitter.SymbolCapture{
			{NameCapture: "func_name", Kind: "function"},
			{NameCapture: "method_name", Kind: "method"},
			{NameCapture: "class_name", Kind: "type"},
			{NameCapture: "struct_name", Kind: "type"},
			{NameCapture: "enum_name", Kind: "type"},
			{NameCapture: "typedef_name", Kind: "type"},
			{NameCapture: "namespace_name", Kind: "type"},
		},
		FunctionQuery:   cppFunctionQuery,
		FunctionCapture: "func",
		FuncNameCapture: "name",
		ComplexityQuery: cppComplexityQuery,
	})
}
