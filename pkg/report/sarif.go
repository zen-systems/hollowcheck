package report

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"

	"github.com/zen-systems/hollowcheck/pkg/detect"
)

// SARIF schema and version constants.
const (
	SARIFVersion = "2.1.0"
	SARIFSchema  = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json"
	ToolName     = "hollowcheck"
	InfoURI      = "https://github.com/zen-systems/hollowcheck"
)

// SARIFReport is the top-level SARIF document.
type SARIFReport struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []SARIFRun `json:"runs"`
}

// SARIFRun represents a single analysis run.
type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

// SARIFTool describes the analysis tool.
type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

// SARIFDriver describes the tool's driver component.
type SARIFDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []SARIFRule `json:"rules"`
}

// SARIFRule describes a rule that can produce results.
type SARIFRule struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	ShortDescription SARIFMessage    `json:"shortDescription"`
	FullDescription  SARIFMessage    `json:"fullDescription,omitempty"`
	HelpURI          string          `json:"helpUri,omitempty"`
	DefaultConfig    SARIFRuleConfig `json:"defaultConfiguration"`
}

// SARIFRuleConfig specifies the default severity level for a rule.
type SARIFRuleConfig struct {
	Level string `json:"level"`
}

// SARIFResult represents a single finding.
type SARIFResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   SARIFMessage    `json:"message"`
	Locations []SARIFLocation `json:"locations"`
}

// SARIFLocation specifies where a result was found.
type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

// SARIFPhysicalLocation describes a physical location in a file.
type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifact `json:"artifactLocation"`
	Region           SARIFRegion   `json:"region"`
}

// SARIFArtifact identifies a file.
type SARIFArtifact struct {
	URI string `json:"uri"`
}

// SARIFRegion specifies a region within a file.
type SARIFRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
}

// SARIFMessage is a text message.
type SARIFMessage struct {
	Text string `json:"text"`
}

// ruleInfo contains metadata about a rule.
type ruleInfo struct {
	name             string
	shortDescription string
	fullDescription  string
	helpURI          string
	defaultLevel     string
}

// knownRules maps rule IDs to their metadata.
var knownRules = map[string]ruleInfo{
	detect.RuleForbiddenPattern: {
		name:             "ForbiddenPattern",
		shortDescription: "Detects forbidden patterns like TODO, FIXME, panic(\"not implemented\")",
		fullDescription:  "Identifies code patterns that indicate incomplete or placeholder implementations, such as TODO comments, FIXME markers, and panic statements.",
		helpURI:          InfoURI + "#forbidden-patterns",
		defaultLevel:     "error",
	},
	detect.RuleMockData: {
		name:             "MockData",
		shortDescription: "Detects mock/placeholder data like example.com, fake IDs",
		fullDescription:  "Identifies hardcoded placeholder values that should be replaced with real data or configuration, such as example.com domains, fake UUIDs, and test credentials.",
		helpURI:          InfoURI + "#mock-data",
		defaultLevel:     "warning",
	},
	detect.RuleMissingFile: {
		name:             "MissingFile",
		shortDescription: "Detects missing required files",
		fullDescription:  "Verifies that all files specified as required in the contract exist in the project.",
		helpURI:          InfoURI + "#required-files",
		defaultLevel:     "error",
	},
	detect.RuleMissingSymbol: {
		name:             "MissingSymbol",
		shortDescription: "Detects missing required symbols (functions, types)",
		fullDescription:  "Verifies that all symbols (functions, types, constants) specified as required in the contract are defined in the code.",
		helpURI:          InfoURI + "#required-symbols",
		defaultLevel:     "error",
	},
	detect.RuleLowComplexity: {
		name:             "LowComplexity",
		shortDescription: "Detects stub implementations with suspiciously low complexity",
		fullDescription:  "Identifies functions that have cyclomatic complexity below the expected threshold, suggesting they may be stub or placeholder implementations.",
		helpURI:          InfoURI + "#complexity",
		defaultLevel:     "error",
	},
	detect.RuleMissingTest: {
		name:             "MissingTest",
		shortDescription: "Detects missing required test functions",
		fullDescription:  "Verifies that all test functions specified as required in the contract exist.",
		helpURI:          InfoURI + "#required-tests",
		defaultLevel:     "warning",
	},
	// Prose rules
	"filler_phrase": {
		name:             "FillerPhrase",
		shortDescription: "Detects filler phrases that add no meaning",
		fullDescription:  "Identifies redundant phrases, hedging language, and filler words that dilute the clarity of prose.",
		helpURI:          InfoURI + "#prose-fillers",
		defaultLevel:     "warning",
	},
	"weasel_word": {
		name:             "WeaselWord",
		shortDescription: "Detects weasel words and vague language",
		fullDescription:  "Identifies anonymous authority claims, passive voice constructions, and other language patterns that reduce precision.",
		helpURI:          InfoURI + "#prose-weasels",
		defaultLevel:     "warning",
	},
	"low_density": {
		name:             "LowDensity",
		shortDescription: "Detects sections with low information density",
		fullDescription:  "Identifies text sections that have a low ratio of content words to total words, suggesting padding or filler content.",
		helpURI:          InfoURI + "#prose-density",
		defaultLevel:     "warning",
	},
	"prose_repetitive_opener": {
		name:             "RepetitiveOpener",
		shortDescription: "Detects repetitive sentence openers",
		fullDescription:  "Identifies when multiple sentences start with the same pattern, suggesting formulaic or AI-generated prose.",
		helpURI:          InfoURI + "#prose-structure",
		defaultLevel:     "warning",
	},
	"prose_middle_sag": {
		name:             "MiddleSag",
		shortDescription: "Detects middle sections with lower quality than intro/conclusion",
		fullDescription:  "Identifies when the middle of a document has significantly lower information density than the introduction and conclusion.",
		helpURI:          InfoURI + "#prose-structure",
		defaultLevel:     "error",
	},
	"prose_weak_transition": {
		name:             "WeakTransition",
		shortDescription: "Detects weak sentence transitions",
		fullDescription:  "Identifies sentences that start with weak transitional phrases like 'And', 'But', 'So' at the beginning.",
		helpURI:          InfoURI + "#prose-structure",
		defaultLevel:     "note",
	},
}

// mapSeverityToLevel converts hollowcheck severity to SARIF level.
func mapSeverityToLevel(severity string) string {
	switch severity {
	case detect.SeverityError:
		return "error"
	case detect.SeverityWarning:
		return "warning"
	case detect.SeverityInfo:
		return "note"
	default:
		return "warning"
	}
}

// getRuleInfo returns metadata for a rule, creating a default if unknown.
func getRuleInfo(ruleID string) ruleInfo {
	if info, ok := knownRules[ruleID]; ok {
		return info
	}

	// Default for unknown rules - convert snake_case to PascalCase
	name := toTitleCase(ruleID)
	return ruleInfo{
		name:             name,
		shortDescription: "Detected: " + ruleID,
		fullDescription:  "A " + ruleID + " violation was detected.",
		helpURI:          InfoURI,
		defaultLevel:     "warning",
	}
}

// toTitleCase converts a snake_case string to PascalCase.
func toTitleCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

// makeRelativePath converts an absolute path to a relative URI for SARIF.
// basePath should be the directory being scanned (or the file itself for single file scans).
func makeRelativePath(filePath, basePath string) string {
	if basePath == "" {
		return filePath
	}

	// If they're the same (single file scan), return just the filename
	if filePath == basePath {
		return filepath.Base(filePath)
	}

	rel, err := filepath.Rel(basePath, filePath)
	if err != nil {
		return filePath
	}

	// SARIF uses forward slashes
	return filepath.ToSlash(rel)
}

// WriteSARIF writes the lint result as SARIF 2.1.0 to the given writer.
func WriteSARIF(w io.Writer, basePath, version string, result *detect.DetectionResult) error {
	// Collect unique rules from violations
	ruleSet := make(map[string]bool)
	for _, v := range result.Violations {
		ruleSet[v.Rule] = true
	}

	// Build rules list
	rules := make([]SARIFRule, 0, len(ruleSet))
	for ruleID := range ruleSet {
		info := getRuleInfo(ruleID)
		rules = append(rules, SARIFRule{
			ID:               ruleID,
			Name:             info.name,
			ShortDescription: SARIFMessage{Text: info.shortDescription},
			FullDescription:  SARIFMessage{Text: info.fullDescription},
			HelpURI:          info.helpURI,
			DefaultConfig:    SARIFRuleConfig{Level: info.defaultLevel},
		})
	}

	// Build results list
	results := make([]SARIFResult, 0, len(result.Violations))
	for _, v := range result.Violations {
		results = append(results, SARIFResult{
			RuleID:  v.Rule,
			Level:   mapSeverityToLevel(v.Severity),
			Message: SARIFMessage{Text: v.Message},
			Locations: []SARIFLocation{
				{
					PhysicalLocation: SARIFPhysicalLocation{
						ArtifactLocation: SARIFArtifact{
							URI: makeRelativePath(v.File, basePath),
						},
						Region: SARIFRegion{
							StartLine: v.Line,
						},
					},
				},
			},
		})
	}

	report := SARIFReport{
		Version: SARIFVersion,
		Schema:  SARIFSchema,
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:           ToolName,
						Version:        version,
						InformationURI: InfoURI,
						Rules:          rules,
					},
				},
				Results: results,
			},
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
