package prose

import (
	"regexp"
	"strings"
	"unicode"
)

// Section represents a section of text with density information.
type Section struct {
	Title      string
	StartLine  int
	EndLine    int
	Text       string
	WordCount  int
	SentCount  int
	Density    float64 // Information density score
}

// DensityScore holds the overall density analysis results.
type DensityScore struct {
	Sections            []Section
	OverallDensity      float64
	LowDensitySections  int
	HighDensitySections int
	VarianceScore       float64 // How uneven the density is across sections
}

// DensityConfig holds configuration for density analysis.
type DensityConfig struct {
	MinSectionWords int     // Minimum words for a section to be analyzed
	LowThreshold    float64 // Below this is "low density"
	HighThreshold   float64 // Above this is "high density"
}

// DefaultDensityConfig returns sensible defaults.
func DefaultDensityConfig() DensityConfig {
	return DensityConfig{
		MinSectionWords: 20,
		LowThreshold:    0.3,
		HighThreshold:   0.8,
	}
}

// headingPatterns identifies section headers.
var headingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^#{1,6}\s+.+`),              // Markdown headers
	regexp.MustCompile(`^[A-Z][A-Za-z0-9\s]+:$`),    // Colon-terminated headers
	regexp.MustCompile(`^\d+\.\s+[A-Z]`),            // Numbered sections
	regexp.MustCompile(`^[A-Z][A-Z\s]+$`),           // ALL CAPS headers
	regexp.MustCompile(`^={3,}$|^-{3,}$`),           // Underlined headers (following line)
}

// contentWords are common words that don't carry much meaning.
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true, "may": true, "might": true,
	"shall": true, "can": true, "must": true,
	"to": true, "of": true, "in": true, "for": true, "on": true, "with": true,
	"at": true, "by": true, "from": true, "as": true, "into": true, "through": true,
	"during": true, "before": true, "after": true, "above": true, "below": true,
	"between": true, "under": true, "over": true,
	"this": true, "that": true, "these": true, "those": true, "it": true,
	"its": true, "their": true, "they": true, "them": true,
	"he": true, "she": true, "him": true, "her": true, "his": true, "hers": true,
	"we": true, "us": true, "our": true, "you": true, "your": true,
	"i": true, "me": true, "my": true,
	"who": true, "what": true, "which": true, "when": true, "where": true, "why": true, "how": true,
	"all": true, "each": true, "every": true, "both": true, "few": true, "more": true,
	"most": true, "other": true, "some": true, "such": true, "no": true, "not": true,
	"only": true, "same": true, "so": true, "than": true, "too": true, "very": true,
	"just": true, "also": true, "now": true, "then": true, "here": true, "there": true,
}

// isHeading checks if a line is a section header.
func isHeading(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	for _, pattern := range headingPatterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

// extractTitle extracts the header text from a line.
func extractTitle(line string) string {
	line = strings.TrimSpace(line)
	// Remove markdown header markers
	if strings.HasPrefix(line, "#") {
		line = strings.TrimLeft(line, "#")
		return strings.TrimSpace(line)
	}
	// Remove trailing colon
	line = strings.TrimSuffix(line, ":")
	return strings.TrimSpace(line)
}

// countSentences approximates sentence count.
func countSentences(text string) int {
	// Count sentence-ending punctuation
	count := 0
	for _, r := range text {
		if r == '.' || r == '!' || r == '?' {
			count++
		}
	}
	if count == 0 {
		count = 1 // At least one "sentence"
	}
	return count
}

// contentWordRatio calculates the ratio of content words to total words.
func contentWordRatio(text string) float64 {
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return 0
	}

	contentWords := 0
	for _, word := range words {
		// Clean punctuation
		word = strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		if word == "" {
			continue
		}
		if !stopWords[word] {
			contentWords++
		}
	}

	return float64(contentWords) / float64(len(words))
}

// calculateDensity computes an information density score for text.
// Higher scores indicate more information-dense text.
func calculateDensity(text string) float64 {
	wordCount := len(strings.Fields(text))
	if wordCount == 0 {
		return 0
	}

	sentCount := countSentences(text)
	contentRatio := contentWordRatio(text)

	// Average words per sentence (ideal range: 15-25)
	wordsPerSent := float64(wordCount) / float64(sentCount)
	sentenceScore := 1.0
	if wordsPerSent < 10 {
		sentenceScore = wordsPerSent / 10.0 // Penalize very short sentences
	} else if wordsPerSent > 30 {
		sentenceScore = 30.0 / wordsPerSent // Penalize very long sentences
	}

	// Combine factors
	// Content ratio weighted more heavily as it directly measures information
	density := (contentRatio * 0.7) + (sentenceScore * 0.3)

	return density
}

// AnalyzeDensity splits text into sections and analyzes information density.
func AnalyzeDensity(text string, cfg DensityConfig) DensityScore {
	lines := strings.Split(text, "\n")

	var sections []Section
	var currentSection *Section
	var currentLines []string
	startLine := 1

	// Helper to finalize current section
	finishSection := func(endLine int) {
		if currentSection == nil {
			return
		}
		currentSection.Text = strings.Join(currentLines, "\n")
		currentSection.EndLine = endLine
		currentSection.WordCount = len(strings.Fields(currentSection.Text))
		currentSection.SentCount = countSentences(currentSection.Text)
		if currentSection.WordCount >= cfg.MinSectionWords {
			currentSection.Density = calculateDensity(currentSection.Text)
			sections = append(sections, *currentSection)
		}
		currentSection = nil
		currentLines = nil
	}

	for i, line := range lines {
		lineNum := i + 1

		if isHeading(line) {
			finishSection(lineNum - 1)
			currentSection = &Section{
				Title:     extractTitle(line),
				StartLine: lineNum,
			}
			startLine = lineNum + 1
			continue
		}

		if currentSection == nil {
			// Create implicit first section
			currentSection = &Section{
				Title:     "(introduction)",
				StartLine: startLine,
			}
		}
		currentLines = append(currentLines, line)
	}

	// Finish last section
	finishSection(len(lines))

	// Calculate overall metrics
	result := DensityScore{Sections: sections}

	if len(sections) == 0 {
		return result
	}

	var totalDensity float64
	var densities []float64
	for _, s := range sections {
		totalDensity += s.Density
		densities = append(densities, s.Density)

		if s.Density < cfg.LowThreshold {
			result.LowDensitySections++
		} else if s.Density > cfg.HighThreshold {
			result.HighDensitySections++
		}
	}

	result.OverallDensity = totalDensity / float64(len(sections))

	// Calculate variance
	if len(densities) > 1 {
		var sumSqDiff float64
		for _, d := range densities {
			diff := d - result.OverallDensity
			sumSqDiff += diff * diff
		}
		result.VarianceScore = sumSqDiff / float64(len(densities))
	}

	return result
}

// DensityViolationScore returns a score 0-100 based on density issues.
// Low density and high variance are penalized.
func DensityViolationScore(ds DensityScore) float64 {
	if len(ds.Sections) == 0 {
		return 0
	}

	// Penalty for low overall density
	densityPenalty := 0.0
	if ds.OverallDensity < 0.4 {
		densityPenalty = (0.4 - ds.OverallDensity) * 100
	}

	// Penalty for variance (inconsistent quality)
	variancePenalty := ds.VarianceScore * 100

	// Penalty for low-density sections
	lowDensityPenalty := float64(ds.LowDensitySections) / float64(len(ds.Sections)) * 50

	score := densityPenalty*0.4 + variancePenalty*0.3 + lowDensityPenalty*0.3
	if score > 100 {
		score = 100
	}

	return score
}
