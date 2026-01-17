package prose

import (
	"regexp"
	"strings"
)

// StructureIssue represents a detected structural problem.
type StructureIssue struct {
	Type        string
	Description string
	Line        int
	Lines       []int // For issues spanning multiple lines
	Weight      float64
}

// StructureAnalysis holds the results of structural analysis.
type StructureAnalysis struct {
	Issues              []StructureIssue
	RepetitiveOpeners   int
	MiddleSag           bool
	WeakTransitions     int
	ParagraphVariance   float64
	Score               float64 // 0-100, higher is worse
}

// Common sentence openers that become repetitive.
var repetitiveOpenerPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(this|these|those)\s+(is|are|was|were|will|would|can|could|should|has|have|had)`),
	regexp.MustCompile(`(?i)^(it|there)\s+(is|are|was|were|will|would|can|could|should|has|have|had)`),
	regexp.MustCompile(`(?i)^(the\s+\w+)\s+(is|are|was|were|will|would|can|could|should|has|have|had)`),
	regexp.MustCompile(`(?i)^(in\s+addition|additionally|furthermore|moreover|also)\b`),
	regexp.MustCompile(`(?i)^(first|second|third|finally|lastly)\b`),
	regexp.MustCompile(`(?i)^(however|therefore|thus|hence|consequently)\b`),
	regexp.MustCompile(`(?i)^(we|you|i)\s+(can|will|should|must|need|have)\b`),
}

// Weak transition patterns.
var weakTransitionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(and|but|so|or)\s+\w`),        // Starting with coordinating conjunctions
	regexp.MustCompile(`(?i)^(also|too)\s+\w`),             // Weak additions
	regexp.MustCompile(`(?i)^(anyway|anyhow)\b`),           // Dismissive transitions
	regexp.MustCompile(`(?i)^(as\s+I\s+said|as\s+noted)\b`), // Redundant back-references
}

// extractSentences splits text into sentences.
func extractSentences(text string) []string {
	// Simple sentence splitting - could be improved with NLP
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")

	// Split on sentence-ending punctuation followed by space and capital
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])

		// Check for sentence end
		if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' {
			// Look ahead for space + capital or end
			if i+1 >= len(runes) {
				sentences = append(sentences, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}

			// Skip whitespace
			j := i + 1
			for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t') {
				j++
			}

			// Check for capital letter (new sentence)
			if j < len(runes) && runes[j] >= 'A' && runes[j] <= 'Z' {
				sentences = append(sentences, strings.TrimSpace(current.String()))
				current.Reset()
				i = j - 1 // Back up so loop advances to j
			}
		}
	}

	// Add any remaining text
	if current.Len() > 0 {
		if s := strings.TrimSpace(current.String()); s != "" {
			sentences = append(sentences, s)
		}
	}

	return sentences
}

// extractParagraphs splits text into paragraphs.
func extractParagraphs(text string) []string {
	// Split on double newlines
	parts := regexp.MustCompile(`\n\s*\n`).Split(text, -1)
	var paragraphs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			paragraphs = append(paragraphs, p)
		}
	}
	return paragraphs
}

// detectRepetitiveOpeners finds sentences that start the same way.
func detectRepetitiveOpeners(sentences []string) []StructureIssue {
	var issues []StructureIssue

	// Track opener patterns and their occurrence lines
	type occurrence struct {
		pattern string
		lines   []int
	}
	openerCounts := make(map[string]*occurrence)

	for i, sent := range sentences {
		sent = strings.TrimSpace(sent)
		if sent == "" {
			continue
		}

		for _, pattern := range repetitiveOpenerPatterns {
			if match := pattern.FindString(sent); match != "" {
				key := pattern.String()
				if openerCounts[key] == nil {
					openerCounts[key] = &occurrence{pattern: match, lines: []int{}}
				}
				openerCounts[key].lines = append(openerCounts[key].lines, i+1)
				break
			}
		}
	}

	// Report patterns that appear too frequently
	for _, occ := range openerCounts {
		if len(occ.lines) >= 3 {
			issues = append(issues, StructureIssue{
				Type:        "repetitive_opener",
				Description: "sentences repeatedly start with similar pattern '" + occ.pattern + "'",
				Lines:       occ.lines,
				Weight:      float64(len(occ.lines)) * 0.5,
			})
		}
	}

	return issues
}

// detectWeakTransitions finds sentences that start with weak transitions.
func detectWeakTransitions(sentences []string) []StructureIssue {
	var issues []StructureIssue

	for i, sent := range sentences {
		sent = strings.TrimSpace(sent)
		if sent == "" {
			continue
		}

		for _, pattern := range weakTransitionPatterns {
			if match := pattern.FindString(sent); match != "" {
				issues = append(issues, StructureIssue{
					Type:        "weak_transition",
					Description: "sentence starts with weak transition '" + match + "'",
					Line:        i + 1,
					Weight:      0.8,
				})
				break
			}
		}
	}

	return issues
}

// detectMiddleSag checks if the middle of the text has lower density.
func detectMiddleSag(paragraphs []string) (bool, StructureIssue) {
	if len(paragraphs) < 5 {
		return false, StructureIssue{}
	}

	// Calculate density for intro, middle, and conclusion sections
	third := len(paragraphs) / 3

	introText := strings.Join(paragraphs[:third], "\n")
	middleText := strings.Join(paragraphs[third:2*third], "\n")
	conclusionText := strings.Join(paragraphs[2*third:], "\n")

	introDensity := calculateDensity(introText)
	middleDensity := calculateDensity(middleText)
	conclusionDensity := calculateDensity(conclusionText)

	// Middle sag: middle is significantly less dense than intro and conclusion
	avgEndsDensity := (introDensity + conclusionDensity) / 2
	if middleDensity < avgEndsDensity*0.7 && middleDensity < 0.4 {
		return true, StructureIssue{
			Type:        "middle_sag",
			Description: "middle section has lower information density than introduction and conclusion",
			Weight:      2.0,
		}
	}

	return false, StructureIssue{}
}

// calculateParagraphVariance measures how uneven paragraph lengths are.
func calculateParagraphVariance(paragraphs []string) float64 {
	if len(paragraphs) < 2 {
		return 0
	}

	var lengths []int
	var total int
	for _, p := range paragraphs {
		length := len(strings.Fields(p))
		lengths = append(lengths, length)
		total += length
	}

	avg := float64(total) / float64(len(lengths))

	var sumSqDiff float64
	for _, l := range lengths {
		diff := float64(l) - avg
		sumSqDiff += diff * diff
	}

	variance := sumSqDiff / float64(len(lengths))
	// Normalize: high variance relative to mean is problematic
	if avg > 0 {
		return variance / (avg * avg) // Coefficient of variation squared
	}
	return 0
}

// AnalyzeStructure performs structural analysis on the text.
func AnalyzeStructure(text string) StructureAnalysis {
	result := StructureAnalysis{}

	sentences := extractSentences(text)
	paragraphs := extractParagraphs(text)

	// Detect issues
	openerIssues := detectRepetitiveOpeners(sentences)
	result.Issues = append(result.Issues, openerIssues...)
	result.RepetitiveOpeners = len(openerIssues)

	transitionIssues := detectWeakTransitions(sentences)
	result.Issues = append(result.Issues, transitionIssues...)
	result.WeakTransitions = len(transitionIssues)

	hasSag, sagIssue := detectMiddleSag(paragraphs)
	if hasSag {
		result.MiddleSag = true
		result.Issues = append(result.Issues, sagIssue)
	}

	result.ParagraphVariance = calculateParagraphVariance(paragraphs)

	// Calculate overall score
	var totalWeight float64
	for _, issue := range result.Issues {
		totalWeight += issue.Weight
	}

	// Normalize to 0-100
	// High variance also contributes
	varianceContribution := result.ParagraphVariance * 20

	result.Score = totalWeight*5 + varianceContribution
	if result.Score > 100 {
		result.Score = 100
	}

	return result
}
