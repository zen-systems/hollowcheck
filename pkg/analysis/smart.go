//go:build !simple

package analysis

import (
	"bufio"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zen-systems/hollowcheck/pkg/contract"
	"github.com/zen-systems/hollowcheck/pkg/parser"
)

// TreeSitterAnalyzer implements the Analyzer interface using tree-sitter for AST-based analysis.
// This is the full-featured implementation that provides:
//   - AST-based symbol detection
//   - Cyclomatic complexity analysis
//   - Forbidden pattern detection
//   - Mock data signature detection
//
// Note: This implementation requires CGO and tree-sitter bindings.
// For environments without CGO support, use SimpleAnalyzer instead.
type TreeSitterAnalyzer struct {
	baseDir string // Base directory for resolving relative paths
}

// NewTreeSitterAnalyzer creates a new TreeSitterAnalyzer instance.
func NewTreeSitterAnalyzer(baseDir string) *TreeSitterAnalyzer {
	return &TreeSitterAnalyzer{baseDir: baseDir}
}

// Name returns the analyzer engine name.
func (a *TreeSitterAnalyzer) Name() string {
	return "smart"
}

// Analyze scans the provided files for violations using AST-based analysis.
func (a *TreeSitterAnalyzer) Analyze(files map[string]string, c *contract.Contract) (*Result, error) {
	result := &Result{}

	// Pre-compile forbidden patterns
	forbiddenPatterns, err := a.compilePatterns(c.ForbiddenPatterns)
	if err != nil {
		return nil, fmt.Errorf("compiling forbidden patterns: %w", err)
	}

	// Pre-compile mock data patterns
	var mockPatterns []tsCompiledPattern
	if c.MockSignatures != nil && len(c.MockSignatures.Patterns) > 0 {
		mockPatterns, err = a.compileMockPatterns(c.MockSignatures.Patterns)
		if err != nil {
			return nil, fmt.Errorf("compiling mock patterns: %w", err)
		}
	}

	// Build maps for symbol and complexity checking
	foundSymbols := make(map[string][]symbolInfo)
	funcComplexities := make(map[string][]funcComplexity)

	// Process each file
	for filePath, content := range files {
		ext := filepath.Ext(filePath)

		// Extract symbols and calculate complexity
		var syms []symbolInfo
		var funcs []funcComplexity
		var parseErr error

		if ext == ".go" {
			// Use native go/ast for Go files
			syms, parseErr = a.extractSymbolsGo(filePath, content)
			if parseErr == nil {
				funcs, parseErr = a.calculateComplexitiesGo(filePath, content)
			}
		} else if p, ok := parser.ForExtension(ext); ok {
			// Use parser registry (tree-sitter) for other supported languages
			syms, parseErr = a.extractSymbolsWithParser(filePath, content, p)
			if parseErr == nil {
				funcs, parseErr = a.calculateComplexitiesWithParser(filePath, content, p, syms)
			}
		}

		if parseErr != nil {
			return nil, fmt.Errorf("parsing %s: %w", filePath, parseErr)
		}

		// Store symbols and complexities by relative path
		relPath := a.relativePath(filePath)
		if syms != nil {
			foundSymbols[relPath] = syms
		}
		if funcs != nil {
			funcComplexities[relPath] = funcs
		}

		// Check forbidden patterns
		violations := a.scanContentForPatterns(filePath, content, forbiddenPatterns, RuleForbiddenPattern, SeverityError)
		result.Violations = append(result.Violations, violations...)

		// Check mock data patterns
		if len(mockPatterns) > 0 {
			isTestFile := strings.HasSuffix(filePath, "_test.go") || strings.Contains(filePath, "/test/")

			if c.MockSignatures.ShouldSkipTestFiles() && isTestFile {
				// Skip test files
			} else {
				severity := SeverityWarning
				if isTestFile {
					testSeverity := c.MockSignatures.GetTestFileSeverity()
					if testSeverity != "" {
						severity = testSeverity
						mockViolations := a.scanContentForPatterns(filePath, content, mockPatterns, RuleMockData, severity)
						result.Violations = append(result.Violations, mockViolations...)
					}
				} else {
					mockViolations := a.scanContentForPatterns(filePath, content, mockPatterns, RuleMockData, severity)
					result.Violations = append(result.Violations, mockViolations...)
				}
			}
		}

		result.Scanned++
	}

	// Check required symbols
	symbolViolations := a.checkRequiredSymbols(foundSymbols, c.RequiredSymbols)
	result.Violations = append(result.Violations, symbolViolations...)

	// Check complexity requirements
	complexityViolations := a.checkComplexityRequirements(funcComplexities, c.Complexity)
	result.Violations = append(result.Violations, complexityViolations...)

	return result, nil
}

// symbolInfo holds information about a found symbol.
type symbolInfo struct {
	Name string
	Kind contract.SymbolKind
	File string
	Line int
}

// funcComplexity holds complexity information for a function.
type funcComplexity struct {
	Name       string
	Complexity int
	File       string
	Line       int
}

// tsCompiledPattern holds a pre-compiled regex with its metadata.
type tsCompiledPattern struct {
	regex       *regexp.Regexp
	pattern     string // Original pattern string for display
	description string
}

// relativePath returns the path relative to baseDir, or the original path if not possible.
func (a *TreeSitterAnalyzer) relativePath(filePath string) string {
	if a.baseDir == "" {
		return filePath
	}
	relPath, err := filepath.Rel(a.baseDir, filePath)
	if err != nil {
		return filePath
	}
	return relPath
}

// compilePatterns pre-compiles forbidden pattern regexes.
func (a *TreeSitterAnalyzer) compilePatterns(patterns []contract.ForbiddenPattern) ([]tsCompiledPattern, error) {
	compiled := make([]tsCompiledPattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", p.Pattern, err)
		}
		compiled = append(compiled, tsCompiledPattern{
			regex:       re,
			pattern:     p.Pattern,
			description: p.Description,
		})
	}
	return compiled, nil
}

// compileMockPatterns pre-compiles mock signature regexes.
func (a *TreeSitterAnalyzer) compileMockPatterns(patterns []contract.MockSignature) ([]tsCompiledPattern, error) {
	compiled := make([]tsCompiledPattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid mock pattern %q: %w", p.Pattern, err)
		}
		compiled = append(compiled, tsCompiledPattern{
			regex:       re,
			pattern:     p.Pattern,
			description: p.Description,
		})
	}
	return compiled, nil
}

// scanContentForPatterns scans file content for pattern violations.
// Returns detailed violations including line number, column, matched text, and context.
func (a *TreeSitterAnalyzer) scanContentForPatterns(filePath, content string, patterns []tsCompiledPattern, rule, severity string) []Violation {
	if len(patterns) == 0 {
		return nil
	}

	var violations []Violation
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, p := range patterns {
			matches := p.regex.FindAllStringSubmatchIndex(line, -1)
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}

				startPos := match[0]
				endPos := match[1]

				if isInsideStringLiteralTS(line, startPos) {
					continue
				}

				// Extract the matched content
				matchedText := line[startPos:endPos]

				// Build the message
				msg := formatViolationMessageTS(rule, p.pattern, p.description, matchedText)

				// Create context (trimmed line for display)
				context := strings.TrimSpace(line)
				if len(context) > 120 {
					if startPos < 60 {
						context = context[:117] + "..."
					} else if startPos > len(context)-60 {
						context = "..." + context[len(context)-117:]
					} else {
						start := startPos - 55
						end := startPos + 60
						if end > len(context) {
							end = len(context)
						}
						context = "..." + context[start:end] + "..."
					}
				}

				violations = append(violations, Violation{
					Rule:     rule,
					Message:  msg,
					File:     filePath,
					Line:     lineNum,
					Column:   startPos + 1,
					Match:    matchedText,
					Context:  context,
					Severity: severity,
				})
			}
		}
	}

	return violations
}

// formatViolationMessageTS creates a human-readable violation message.
func formatViolationMessageTS(rule, pattern, description, matchedText string) string {
	var msg string

	switch rule {
	case RuleForbiddenPattern:
		msg = fmt.Sprintf("forbidden pattern found: %q", matchedText)
	case RuleMockData:
		msg = fmt.Sprintf("mock/placeholder data found: %q", matchedText)
	default:
		msg = fmt.Sprintf("pattern %q matched: %q", pattern, matchedText)
	}

	if description != "" {
		msg = fmt.Sprintf("%s - %s", msg, description)
	}

	return msg
}

// extractSymbolsGo parses Go source content and extracts all symbol definitions.
func (a *TreeSitterAnalyzer) extractSymbolsGo(filePath, content string) ([]symbolInfo, error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, filePath, content, 0)
	if err != nil {
		return nil, err
	}

	var symbols []symbolInfo

	ast.Inspect(f, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			kind := contract.SymbolFunction
			if decl.Recv != nil {
				kind = contract.SymbolMethod
			}
			symbols = append(symbols, symbolInfo{
				Name: decl.Name.Name,
				Kind: kind,
				File: filePath,
				Line: fset.Position(decl.Pos()).Line,
			})

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					symbols = append(symbols, symbolInfo{
						Name: s.Name.Name,
						Kind: contract.SymbolType,
						File: filePath,
						Line: fset.Position(s.Pos()).Line,
					})
				case *ast.ValueSpec:
					if decl.Tok == token.CONST {
						for _, name := range s.Names {
							symbols = append(symbols, symbolInfo{
								Name: name.Name,
								Kind: contract.SymbolConst,
								File: filePath,
								Line: fset.Position(name.Pos()).Line,
							})
						}
					}
				}
			}
		}
		return true
	})

	return symbols, nil
}

// calculateComplexitiesGo parses Go source content and calculates cyclomatic complexity.
func (a *TreeSitterAnalyzer) calculateComplexitiesGo(filePath, content string) ([]funcComplexity, error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, filePath, content, 0)
	if err != nil {
		return nil, err
	}

	var funcs []funcComplexity

	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		complexity := a.calculateFuncComplexity(fn)
		funcs = append(funcs, funcComplexity{
			Name:       fn.Name.Name,
			Complexity: complexity,
			File:       filePath,
			Line:       fset.Position(fn.Pos()).Line,
		})

		return true
	})

	return funcs, nil
}

// calculateFuncComplexity calculates the cyclomatic complexity of a Go function.
func (a *TreeSitterAnalyzer) calculateFuncComplexity(fn *ast.FuncDecl) int {
	if fn.Body == nil {
		return 1
	}

	complexity := 1

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			if node.List != nil {
				complexity++
			}
		case *ast.CommClause:
			if node.Comm != nil {
				complexity++
			}
		case *ast.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		}
		return true
	})

	return complexity
}

// extractSymbolsWithParser uses the parser interface for non-Go files.
func (a *TreeSitterAnalyzer) extractSymbolsWithParser(filePath, content string, p parser.Parser) ([]symbolInfo, error) {
	syms, err := p.ParseSymbols([]byte(content))
	if err != nil {
		return nil, err
	}

	result := make([]symbolInfo, len(syms))
	for i, sym := range syms {
		result[i] = symbolInfo{
			Name: sym.Name,
			Kind: contract.SymbolKind(sym.Kind),
			File: filePath,
			Line: sym.Line,
		}
	}

	return result, nil
}

// calculateComplexitiesWithParser uses the parser interface for non-Go files.
func (a *TreeSitterAnalyzer) calculateComplexitiesWithParser(filePath, content string, p parser.Parser, symbols []symbolInfo) ([]funcComplexity, error) {
	var funcs []funcComplexity

	for _, sym := range symbols {
		if sym.Kind != contract.SymbolFunction && sym.Kind != contract.SymbolMethod {
			continue
		}

		complexity, err := p.Complexity([]byte(content), sym.Name)
		if err != nil {
			return nil, err
		}

		funcs = append(funcs, funcComplexity{
			Name:       sym.Name,
			Complexity: complexity,
			File:       filePath,
			Line:       sym.Line,
		})
	}

	return funcs, nil
}

// checkRequiredSymbols verifies that all required symbols exist.
func (a *TreeSitterAnalyzer) checkRequiredSymbols(foundSymbols map[string][]symbolInfo, requirements []contract.RequiredSymbol) []Violation {
	var violations []Violation

	for _, req := range requirements {
		found := false
		syms, ok := foundSymbols[req.File]
		if ok {
			for _, sym := range syms {
				if sym.Name == req.Name && sym.Kind == req.Kind {
					found = true
					break
				}
			}
		}

		if !found {
			violations = append(violations, Violation{
				Rule:     RuleMissingSymbol,
				Message:  fmt.Sprintf("required %s %q not found", req.Kind, req.Name),
				File:     req.File,
				Line:     0,
				Severity: SeverityError,
			})
		}
	}

	return violations
}

// checkComplexityRequirements verifies that functions meet minimum complexity requirements.
func (a *TreeSitterAnalyzer) checkComplexityRequirements(funcComplexities map[string][]funcComplexity, requirements []contract.ComplexityRequirement) []Violation {
	var violations []Violation

	for _, req := range requirements {
		found := false
		var actualComplexity int

		if req.File != "" {
			funcs, ok := funcComplexities[req.File]
			if ok {
				for _, f := range funcs {
					if f.Name == req.Symbol {
						found = true
						actualComplexity = f.Complexity
						break
					}
				}
			}
		} else {
			for _, funcs := range funcComplexities {
				for _, f := range funcs {
					if f.Name == req.Symbol {
						found = true
						actualComplexity = f.Complexity
						break
					}
				}
				if found {
					break
				}
			}
		}

		if !found {
			file := req.File
			if file == "" {
				file = "(any file)"
			}
			violations = append(violations, Violation{
				Rule:     RuleLowComplexity,
				Message:  fmt.Sprintf("symbol %q not found for complexity check", req.Symbol),
				File:     file,
				Line:     0,
				Severity: SeverityError,
			})
			continue
		}

		if actualComplexity < req.MinComplexity {
			file := req.File
			if file == "" {
				file = "(found in codebase)"
			}
			violations = append(violations, Violation{
				Rule:     RuleLowComplexity,
				Message:  fmt.Sprintf("symbol %q has complexity %d, minimum required is %d", req.Symbol, actualComplexity, req.MinComplexity),
				File:     file,
				Line:     0,
				Severity: SeverityWarning,
			})
		}
	}

	return violations
}

// isInsideStringLiteralTS checks if a position in a line falls within a string literal.
func isInsideStringLiteralTS(line string, pos int) bool {
	var inString bool
	var stringChar rune
	escaped := false

	for i, ch := range line {
		if i >= pos {
			return inString
		}

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' || ch == '\'' || ch == '`' {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				inString = false
			}
		}
	}

	return inString
}
