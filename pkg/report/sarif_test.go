package report

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/zen-systems/hollowcheck/pkg/detect"
)

func TestWriteSARIF(t *testing.T) {
	result := &detect.DetectionResult{
		Scanned: 3,
		Violations: []detect.Violation{
			{
				Rule:     detect.RuleForbiddenPattern,
				Message:  "forbidden pattern \"TODO\" found: Work-in-progress marker",
				File:     "/home/user/project/pkg/client/client.go",
				Line:     47,
				Severity: detect.SeverityError,
			},
			{
				Rule:     detect.RuleMockData,
				Message:  "mock data pattern found: example.com domain",
				File:     "/home/user/project/pkg/config/config.go",
				Line:     15,
				Severity: detect.SeverityWarning,
			},
			{
				Rule:     detect.RuleMissingSymbol,
				Message:  "required symbol 'NewClient' not found",
				File:     "/home/user/project/pkg/client/client.go",
				Line:     0,
				Severity: detect.SeverityError,
			},
		},
	}

	var buf bytes.Buffer
	err := WriteSARIF(&buf, "/home/user/project", "0.1.0", result)
	if err != nil {
		t.Fatalf("WriteSARIF failed: %v", err)
	}

	// Parse the output
	var report SARIFReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("Failed to parse SARIF output: %v", err)
	}

	// Verify top-level structure
	if report.Version != SARIFVersion {
		t.Errorf("Version = %q, want %q", report.Version, SARIFVersion)
	}
	if report.Schema != SARIFSchema {
		t.Errorf("Schema = %q, want %q", report.Schema, SARIFSchema)
	}
	if len(report.Runs) != 1 {
		t.Fatalf("len(Runs) = %d, want 1", len(report.Runs))
	}

	run := report.Runs[0]

	// Verify tool info
	if run.Tool.Driver.Name != ToolName {
		t.Errorf("Tool.Driver.Name = %q, want %q", run.Tool.Driver.Name, ToolName)
	}
	if run.Tool.Driver.Version != "0.1.0" {
		t.Errorf("Tool.Driver.Version = %q, want %q", run.Tool.Driver.Version, "0.1.0")
	}
	if run.Tool.Driver.InformationURI != InfoURI {
		t.Errorf("Tool.Driver.InformationURI = %q, want %q", run.Tool.Driver.InformationURI, InfoURI)
	}

	// Verify rules were collected
	if len(run.Tool.Driver.Rules) != 3 {
		t.Errorf("len(Rules) = %d, want 3", len(run.Tool.Driver.Rules))
	}

	// Verify results count
	if len(run.Results) != 3 {
		t.Fatalf("len(Results) = %d, want 3", len(run.Results))
	}

	// Verify first result
	res := run.Results[0]
	if res.RuleID != detect.RuleForbiddenPattern {
		t.Errorf("Results[0].RuleID = %q, want %q", res.RuleID, detect.RuleForbiddenPattern)
	}
	if res.Level != "error" {
		t.Errorf("Results[0].Level = %q, want %q", res.Level, "error")
	}
	if len(res.Locations) != 1 {
		t.Fatalf("len(Results[0].Locations) = %d, want 1", len(res.Locations))
	}
	if res.Locations[0].PhysicalLocation.ArtifactLocation.URI != "pkg/client/client.go" {
		t.Errorf("Results[0].Location.URI = %q, want %q", res.Locations[0].PhysicalLocation.ArtifactLocation.URI, "pkg/client/client.go")
	}
	if res.Locations[0].PhysicalLocation.Region.StartLine != 47 {
		t.Errorf("Results[0].Location.StartLine = %d, want %d", res.Locations[0].PhysicalLocation.Region.StartLine, 47)
	}
}

func TestWriteSARIF_EmptyResult(t *testing.T) {
	result := &detect.DetectionResult{
		Scanned:    0,
		Violations: []detect.Violation{},
	}

	var buf bytes.Buffer
	err := WriteSARIF(&buf, "/home/user/project", "0.1.0", result)
	if err != nil {
		t.Fatalf("WriteSARIF failed: %v", err)
	}

	var report SARIFReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("Failed to parse SARIF output: %v", err)
	}

	if len(report.Runs) != 1 {
		t.Fatalf("len(Runs) = %d, want 1", len(report.Runs))
	}
	if len(report.Runs[0].Results) != 0 {
		t.Errorf("len(Results) = %d, want 0", len(report.Runs[0].Results))
	}
	if len(report.Runs[0].Tool.Driver.Rules) != 0 {
		t.Errorf("len(Rules) = %d, want 0", len(report.Runs[0].Tool.Driver.Rules))
	}
}

func TestMapSeverityToLevel(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{detect.SeverityError, "error"},
		{detect.SeverityWarning, "warning"},
		{detect.SeverityInfo, "note"},
		{"unknown", "warning"},
		{"", "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			got := mapSeverityToLevel(tt.severity)
			if got != tt.want {
				t.Errorf("mapSeverityToLevel(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

func TestGetRuleInfo(t *testing.T) {
	tests := []struct {
		ruleID   string
		wantName string
	}{
		{detect.RuleForbiddenPattern, "ForbiddenPattern"},
		{detect.RuleMockData, "MockData"},
		{detect.RuleMissingFile, "MissingFile"},
		{"filler_phrase", "FillerPhrase"},
		{"weasel_word", "WeaselWord"},
		{"unknown_rule_here", "UnknownRuleHere"}, // tests default case
	}

	for _, tt := range tests {
		t.Run(tt.ruleID, func(t *testing.T) {
			info := getRuleInfo(tt.ruleID)
			if info.name != tt.wantName {
				t.Errorf("getRuleInfo(%q).name = %q, want %q", tt.ruleID, info.name, tt.wantName)
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"forbidden_pattern", "ForbiddenPattern"},
		{"mock_data", "MockData"},
		{"single", "Single"},
		{"", ""},
		{"already_Title_Case", "AlreadyTitleCase"},
		{"UPPERCASE", "Uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toTitleCase(tt.input)
			if got != tt.want {
				t.Errorf("toTitleCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMakeRelativePath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		basePath string
		want     string
	}{
		{"nested file", "/home/user/project/pkg/client.go", "/home/user/project", "pkg/client.go"},
		{"root file", "/home/user/project/main.go", "/home/user/project", "main.go"},
		{"same file (single file scan)", "/home/user/project/file.go", "/home/user/project/file.go", "file.go"},
		{"empty base path", "/home/user/project/file.go", "", "/home/user/project/file.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeRelativePath(tt.filePath, tt.basePath)
			if got != tt.want {
				t.Errorf("makeRelativePath(%q, %q) = %q, want %q", tt.filePath, tt.basePath, got, tt.want)
			}
		})
	}
}

func TestWriteSARIF_ProseRules(t *testing.T) {
	result := &detect.DetectionResult{
		Scanned: 1,
		Violations: []detect.Violation{
			{
				Rule:     "filler_phrase",
				Message:  "hedging phrase: \"I think\"",
				File:     "/home/user/docs/README.md",
				Line:     10,
				Severity: detect.SeverityWarning,
			},
			{
				Rule:     "prose_weak_transition",
				Message:  "sentence starts with weak transition 'But'",
				File:     "/home/user/docs/README.md",
				Line:     25,
				Severity: detect.SeverityInfo,
			},
		},
	}

	var buf bytes.Buffer
	err := WriteSARIF(&buf, "/home/user/docs", "0.1.0", result)
	if err != nil {
		t.Fatalf("WriteSARIF failed: %v", err)
	}

	var report SARIFReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("Failed to parse SARIF output: %v", err)
	}

	run := report.Runs[0]

	// Check that prose rules are included
	if len(run.Tool.Driver.Rules) != 2 {
		t.Errorf("len(Rules) = %d, want 2", len(run.Tool.Driver.Rules))
	}

	// Check results have correct levels
	if len(run.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(run.Results))
	}

	if run.Results[0].Level != "warning" {
		t.Errorf("Results[0].Level = %q, want 'warning'", run.Results[0].Level)
	}
	if run.Results[1].Level != "note" {
		t.Errorf("Results[1].Level = %q, want 'note'", run.Results[1].Level)
	}
}
