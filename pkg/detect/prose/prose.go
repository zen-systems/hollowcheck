package prose

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zen-systems/hollowcheck/pkg/contract"
	"github.com/zen-systems/hollowcheck/pkg/detect"
)

// Rule names for prose violations.
const (
	RuleFillerPhrase        = "filler_phrase"
	RuleWeaselWord          = "weasel_word"
	RuleLowDensity          = "low_density"
	RuleRepetitiveStructure = "repetitive_structure"
	RuleMiddleSag           = "middle_sag"
	RuleWeakTransition      = "weak_transition"
)

// Weights for different prose issues (used in scoring).
type ProseWeights struct {
	Filler    float64
	Weasel    float64
	Density   float64
	Structure float64
}

// DefaultWeights returns the default scoring weights.
func DefaultWeights() ProseWeights {
	return ProseWeights{
		Filler:    0.25,
		Weasel:    0.25,
		Density:   0.25,
		Structure: 0.25,
	}
}

// ProseResult holds the results of prose analysis.
type ProseResult struct {
	FillerMatches   []FillerMatch
	WeaselMatches   []WeaselMatch
	DensityScore    DensityScore
	StructureResult StructureAnalysis

	// Individual scores (0-100, higher is worse)
	FillerScore    float64
	WeaselScore    float64
	DensityPenalty float64
	StructurePenalty float64

	// Combined score
	OverallScore float64

	// Violations for integration with hollowcheck
	Violations []detect.Violation
}

// Detector implements the prose detection logic.
type Detector struct {
	Config  *contract.ProseConfig
	Weights ProseWeights
}

// NewDetector creates a new prose detector with the given configuration.
func NewDetector(cfg *contract.ProseConfig) *Detector {
	weights := DefaultWeights()
	if cfg != nil && cfg.Weights != nil {
		if cfg.Weights.Filler > 0 {
			weights.Filler = cfg.Weights.Filler
		}
		if cfg.Weights.Weasel > 0 {
			weights.Weasel = cfg.Weights.Weasel
		}
		if cfg.Weights.Density > 0 {
			weights.Density = cfg.Weights.Density
		}
		if cfg.Weights.Structure > 0 {
			weights.Structure = cfg.Weights.Structure
		}
	}

	return &Detector{
		Config:  cfg,
		Weights: weights,
	}
}

// AnalyzeText performs prose analysis on the given text.
func (d *Detector) AnalyzeText(text string) *ProseResult {
	result := &ProseResult{}

	// Detect fillers
	result.FillerMatches = DetectFillers(text)
	result.FillerScore = FillerScore(text, result.FillerMatches)

	// Detect weasels
	result.WeaselMatches = DetectWeasels(text)
	result.WeaselScore = WeaselScore(text, result.WeaselMatches)

	// Analyze density
	densityCfg := DefaultDensityConfig()
	if d.Config != nil && d.Config.Density != nil {
		if d.Config.Density.MinSectionWords > 0 {
			densityCfg.MinSectionWords = d.Config.Density.MinSectionWords
		}
		if d.Config.Density.LowThreshold > 0 {
			densityCfg.LowThreshold = d.Config.Density.LowThreshold
		}
		if d.Config.Density.HighThreshold > 0 {
			densityCfg.HighThreshold = d.Config.Density.HighThreshold
		}
	}
	result.DensityScore = AnalyzeDensity(text, densityCfg)
	result.DensityPenalty = DensityViolationScore(result.DensityScore)

	// Analyze structure
	result.StructureResult = AnalyzeStructure(text)
	result.StructurePenalty = result.StructureResult.Score

	// Calculate overall score (weighted average)
	result.OverallScore = result.FillerScore*d.Weights.Filler +
		result.WeaselScore*d.Weights.Weasel +
		result.DensityPenalty*d.Weights.Density +
		result.StructurePenalty*d.Weights.Structure

	return result
}

// AnalyzeFile analyzes a single file and returns prose analysis results.
func (d *Detector) AnalyzeFile(filePath string) (*ProseResult, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	result := d.AnalyzeText(string(content))

	// Convert matches to violations
	result.Violations = d.toViolations(filePath, result)

	return result, nil
}

// toViolations converts prose analysis results to hollowcheck violations.
func (d *Detector) toViolations(filePath string, result *ProseResult) []detect.Violation {
	var violations []detect.Violation

	// Get thresholds from config
	fillerThreshold := 10
	weaselThreshold := 10
	if d.Config != nil {
		if d.Config.FillerThreshold > 0 {
			fillerThreshold = d.Config.FillerThreshold
		}
		if d.Config.WeaselThreshold > 0 {
			weaselThreshold = d.Config.WeaselThreshold
		}
	}

	// Filler violations (only report if above threshold)
	if len(result.FillerMatches) > fillerThreshold {
		// Report top violations
		for i, m := range result.FillerMatches {
			if i >= 20 { // Cap at 20 violations per category
				break
			}
			severity := detect.SeverityWarning
			if m.Weight >= 1.0 {
				severity = detect.SeverityError
			}
			violations = append(violations, detect.Violation{
				Rule:     RuleFillerPhrase,
				Message:  fmt.Sprintf("%s: \"%s\"", m.Description, m.Text),
				File:     filePath,
				Line:     m.Line,
				Severity: severity,
			})
		}
	}

	// Weasel violations
	if len(result.WeaselMatches) > weaselThreshold {
		for i, m := range result.WeaselMatches {
			if i >= 20 {
				break
			}
			severity := detect.SeverityWarning
			if m.Weight >= 1.0 {
				severity = detect.SeverityError
			}
			violations = append(violations, detect.Violation{
				Rule:     RuleWeaselWord,
				Message:  fmt.Sprintf("%s: \"%s\"", m.Description, m.Text),
				File:     filePath,
				Line:     m.Line,
				Severity: severity,
			})
		}
	}

	// Density violations
	for _, section := range result.DensityScore.Sections {
		if section.Density < 0.3 {
			violations = append(violations, detect.Violation{
				Rule:     RuleLowDensity,
				Message:  fmt.Sprintf("low information density in section '%s' (%.0f%%)", section.Title, section.Density*100),
				File:     filePath,
				Line:     section.StartLine,
				Severity: detect.SeverityWarning,
			})
		}
	}

	// Structure violations
	for _, issue := range result.StructureResult.Issues {
		var severity string
		switch issue.Type {
		case "middle_sag":
			severity = detect.SeverityError
		case "repetitive_opener":
			severity = detect.SeverityWarning
		default:
			severity = detect.SeverityInfo
		}

		line := issue.Line
		if line == 0 && len(issue.Lines) > 0 {
			line = issue.Lines[0]
		}

		violations = append(violations, detect.Violation{
			Rule:     "prose_" + issue.Type,
			Message:  issue.Description,
			File:     filePath,
			Line:     line,
			Severity: severity,
		})
	}

	return violations
}

// Detect implements the detect.Detector interface.
func (d *Detector) Detect(files []string, c *contract.Contract) (*detect.DetectionResult, error) {
	result := &detect.DetectionResult{}

	// Filter to prose files
	proseExts := map[string]bool{
		".md":   true,
		".txt":  true,
		".rst":  true,
		".adoc": true,
		".tex":  true,
		".html": true,
	}

	if d.Config != nil && len(d.Config.Extensions) > 0 {
		proseExts = make(map[string]bool)
		for _, ext := range d.Config.Extensions {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			proseExts[ext] = true
		}
	}

	for _, file := range files {
		ext := filepath.Ext(file)
		if !proseExts[ext] {
			continue
		}

		proseResult, err := d.AnalyzeFile(file)
		if err != nil {
			return nil, fmt.Errorf("analyzing %s: %w", file, err)
		}

		result.Violations = append(result.Violations, proseResult.Violations...)
		result.Scanned++
	}

	return result, nil
}

// DetectProsePatterns is a convenience function for detecting prose issues
// in a list of files, similar to DetectForbiddenPatterns.
func DetectProsePatterns(files []string, cfg *contract.ProseConfig) (*detect.DetectionResult, error) {
	detector := NewDetector(cfg)
	return detector.Detect(files, nil)
}

// ReadFileContent reads file content, handling large files gracefully.
func ReadFileContent(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(f)
	// Increase buffer for large lines
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return content.String(), nil
}
