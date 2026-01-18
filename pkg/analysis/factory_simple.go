//go:build simple

package analysis

// GetAnalyzer returns the default analyzer for the current build.
// When built with the "simple" tag (-tags simple), this returns a SimpleAnalyzer
// which provides regex-based analysis without requiring CGO.
//
// Usage:
//
//	analyzer := analysis.GetAnalyzer("/path/to/project")
//	result, err := analyzer.Analyze(files, contract)
func GetAnalyzer(baseDir string) Analyzer {
	return NewSimpleAnalyzer()
}

// DefaultAnalyzerName returns the name of the default analyzer for this build.
func DefaultAnalyzerName() string {
	return "simple"
}

// IsCGOEnabled returns true if this build has CGO support.
func IsCGOEnabled() bool {
	return false
}
