// Package prose provides detection of quality issues in prose text.
package prose

import (
	"regexp"
	"strings"
)

// FillerCategory categorizes types of filler phrases.
type FillerCategory string

const (
	FillerHedge         FillerCategory = "hedge"
	FillerQualifier     FillerCategory = "qualifier"
	FillerRedundant     FillerCategory = "redundant"
	FillerVague         FillerCategory = "vague"
	FillerIntensifier   FillerCategory = "intensifier"
	FillerMetaDiscourse FillerCategory = "meta_discourse"
)

// FillerPattern defines a filler phrase pattern with metadata.
type FillerPattern struct {
	Pattern     *regexp.Regexp
	Category    FillerCategory
	Description string
	Weight      float64 // How heavily this should count (1.0 = normal)
}

// fillerPatterns contains all recognized filler patterns.
var fillerPatterns = []FillerPattern{
	// Hedging phrases - weaken statements unnecessarily
	{regexp.MustCompile(`(?i)\b(I think|I believe|I feel|in my opinion)\b`), FillerHedge, "hedging phrase", 1.0},
	{regexp.MustCompile(`(?i)\b(it seems|it appears|it would seem)\b`), FillerHedge, "hedging phrase", 1.0},
	{regexp.MustCompile(`(?i)\b(sort of|kind of|more or less)\b`), FillerHedge, "vague qualifier", 1.0},
	{regexp.MustCompile(`(?i)\b(arguably|perhaps|maybe|possibly|potentially)\b`), FillerHedge, "uncertainty marker", 0.8},
	{regexp.MustCompile(`(?i)\b(somewhat|fairly|rather|quite)\b`), FillerHedge, "weak qualifier", 0.7},

	// Redundant phrases - add no meaning
	{regexp.MustCompile(`(?i)\b(in order to)\b`), FillerRedundant, "redundant phrase (use 'to')", 1.0},
	{regexp.MustCompile(`(?i)\b(due to the fact that)\b`), FillerRedundant, "redundant phrase (use 'because')", 1.0},
	{regexp.MustCompile(`(?i)\b(in the event that)\b`), FillerRedundant, "redundant phrase (use 'if')", 1.0},
	{regexp.MustCompile(`(?i)\b(at this point in time|at the present time)\b`), FillerRedundant, "redundant phrase (use 'now')", 1.0},
	{regexp.MustCompile(`(?i)\b(for all intents and purposes)\b`), FillerRedundant, "redundant phrase (use 'essentially')", 1.0},
	{regexp.MustCompile(`(?i)\b(it is important to note that|it should be noted that)\b`), FillerRedundant, "meta-phrase (just state it)", 1.0},
	{regexp.MustCompile(`(?i)\b(needless to say)\b`), FillerRedundant, "if needless, don't say it", 1.0},
	{regexp.MustCompile(`(?i)\b(as a matter of fact)\b`), FillerRedundant, "redundant phrase", 0.8},
	{regexp.MustCompile(`(?i)\b(in the final analysis)\b`), FillerRedundant, "redundant phrase (use 'finally' or omit)", 1.0},
	{regexp.MustCompile(`(?i)\b(first and foremost)\b`), FillerRedundant, "redundant phrase (use 'first')", 0.8},
	{regexp.MustCompile(`(?i)\b(each and every)\b`), FillerRedundant, "redundant phrase (use 'each' or 'every')", 0.8},
	{regexp.MustCompile(`(?i)\b(basic fundamentals|basic essentials)\b`), FillerRedundant, "redundant phrase", 1.0},
	{regexp.MustCompile(`(?i)\b(past history|future plans)\b`), FillerRedundant, "redundant phrase", 1.0},
	{regexp.MustCompile(`(?i)\b(end result|final outcome)\b`), FillerRedundant, "redundant phrase", 0.8},
	{regexp.MustCompile(`(?i)\b(completely eliminate|totally destroy)\b`), FillerRedundant, "redundant intensifier", 0.8},
	{regexp.MustCompile(`(?i)\b(close proximity)\b`), FillerRedundant, "redundant phrase (use 'near')", 1.0},

	// Vague phrases - lack precision
	{regexp.MustCompile(`(?i)\b(and so on|and so forth|etc\.?)\b`), FillerVague, "vague trailing phrase", 0.9},
	{regexp.MustCompile(`(?i)\b(things like|stuff like)\b`), FillerVague, "vague reference", 1.0},
	{regexp.MustCompile(`(?i)\b(various|several|numerous|a number of)\b`), FillerVague, "vague quantifier", 0.6},
	{regexp.MustCompile(`(?i)\b(very|really|extremely|incredibly|absolutely)\b`), FillerVague, "vague intensifier", 0.5},
	{regexp.MustCompile(`(?i)\b(pretty much|basically|essentially)\b`), FillerVague, "vague qualifier", 0.7},
	{regexp.MustCompile(`(?i)\b(the thing is|the fact is)\b`), FillerVague, "vague phrase", 0.9},

	// Intensifiers - often weaken rather than strengthen
	{regexp.MustCompile(`(?i)\b(literally)\b`), FillerIntensifier, "overused intensifier", 0.8},
	{regexp.MustCompile(`(?i)\b(actually|obviously|clearly|definitely)\b`), FillerIntensifier, "presumptuous intensifier", 0.6},

	// Meta-discourse - talking about the writing instead of the subject
	{regexp.MustCompile(`(?i)\b(as mentioned (earlier|above|before|previously))\b`), FillerMetaDiscourse, "meta-reference", 0.7},
	{regexp.MustCompile(`(?i)\b(as we will see|as we have seen)\b`), FillerMetaDiscourse, "meta-reference", 0.8},
	{regexp.MustCompile(`(?i)\b(let's|let us)\s+(take a look at|examine|consider|discuss)\b`), FillerMetaDiscourse, "meta-phrase", 0.9},
	{regexp.MustCompile(`(?i)\b(in this (article|document|paper|section))\b`), FillerMetaDiscourse, "meta-reference", 0.5},
	{regexp.MustCompile(`(?i)\b(we will now|I will now|let me now)\b`), FillerMetaDiscourse, "meta-phrase", 0.9},
}

// FillerMatch represents a detected filler phrase.
type FillerMatch struct {
	Text        string
	Line        int
	Column      int
	Category    FillerCategory
	Description string
	Weight      float64
}

// DetectFillers scans text for filler phrases and returns matches.
func DetectFillers(text string) []FillerMatch {
	var matches []FillerMatch

	lines := strings.Split(text, "\n")
	for lineNum, line := range lines {
		for _, pattern := range fillerPatterns {
			locs := pattern.Pattern.FindAllStringIndex(line, -1)
			for _, loc := range locs {
				matches = append(matches, FillerMatch{
					Text:        line[loc[0]:loc[1]],
					Line:        lineNum + 1,
					Column:      loc[0] + 1,
					Category:    pattern.Category,
					Description: pattern.Description,
					Weight:      pattern.Weight,
				})
			}
		}
	}

	return matches
}

// FillerScore calculates a filler score for the text.
// Returns a score from 0 (no fillers) to 100 (heavily filled).
func FillerScore(text string, matches []FillerMatch) float64 {
	if len(text) == 0 {
		return 0
	}

	// Calculate weighted match count
	var weightedCount float64
	for _, m := range matches {
		weightedCount += m.Weight
	}

	// Normalize by word count
	words := len(strings.Fields(text))
	if words == 0 {
		return 0
	}

	// Fillers per 100 words, capped at 100
	score := (weightedCount / float64(words)) * 100 * 10 // Scale factor
	if score > 100 {
		score = 100
	}

	return score
}
