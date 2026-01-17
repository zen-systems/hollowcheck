package languages

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"

	"github.com/zen-systems/hollowcheck/pkg/parser"
	"github.com/zen-systems/hollowcheck/pkg/parser/treesitter"
)

func init() {
	parser.Register(".py", NewPythonParser)
}

// Python tree-sitter queries
const (
	pythonSymbolQuery = `
		(function_definition name: (identifier) @func_name) @function
		(class_definition name: (identifier) @class_name) @class
	`

	pythonFunctionQuery = `
		(function_definition name: (identifier) @name) @func
	`

	pythonComplexityQuery = `
		(if_statement) @branch
		(elif_clause) @branch
		(for_statement) @branch
		(while_statement) @branch
		(except_clause) @branch
		(with_statement) @branch
		(conditional_expression) @branch
		(boolean_operator operator: "and") @branch
		(boolean_operator operator: "or") @branch
		(list_comprehension) @branch
		(dictionary_comprehension) @branch
		(set_comprehension) @branch
		(generator_expression) @branch
	`
)

// NewPythonParser creates a Python tree-sitter parser.
func NewPythonParser() parser.Parser {
	return treesitter.New(treesitter.Config{
		Language:     python.GetLanguage(),
		LanguageName: "python",
		SymbolQuery:  pythonSymbolQuery,
		SymbolCaptures: []treesitter.SymbolCapture{
			{NameCapture: "func_name", Kind: "function"},
			{NameCapture: "class_name", Kind: "type"},
		},
		FunctionQuery:   pythonFunctionQuery,
		FunctionCapture: "func",
		FuncNameCapture: "name",
		ComplexityQuery: pythonComplexityQuery,
	})
}

// PythonLanguage returns the Python tree-sitter language for external use.
func PythonLanguage() *sitter.Language {
	return python.GetLanguage()
}
