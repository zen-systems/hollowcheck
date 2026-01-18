//go:build !simple

package analysis

// GetAnalyzer returns the default analyzer for the current build.
// When built without the "simple" tag, this returns a TreeSitterAnalyzer
// which provides full AST-based analysis capabilities.
//
// Usage:
//
//	analyzer := analysis.GetAnalyzer("/path/to/project")
//	result, err := analyzer.Analyze(files, contract)
func GetAnalyzer(baseDir string) Analyzer {
	return NewTreeSitterAnalyzer(baseDir)
}

// DefaultAnalyzerName returns the name of the default analyzer for this build.
func DefaultAnalyzerName() string {
	return "smart"
}

// IsCGOEnabled returns true if this build has CGO support.
func IsCGOEnabled() bool {
	return true
}
