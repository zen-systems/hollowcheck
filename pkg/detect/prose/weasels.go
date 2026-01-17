package prose

import (
	"regexp"
	"strings"
)

// WeaselCategory categorizes types of weasel words.
type WeaselCategory string

const (
	WeaselPassiveVoice      WeaselCategory = "passive_voice"
	WeaselAnonymousAuthority WeaselCategory = "anonymous_authority"
	WeaselNominalization    WeaselCategory = "nominalization"
	WeaselWeaselWord        WeaselCategory = "weasel_word"
	WeaselHedge             WeaselCategory = "hedge"
)

// WeaselPattern defines a weasel word pattern with metadata.
type WeaselPattern struct {
	Pattern     *regexp.Regexp
	Category    WeaselCategory
	Description string
	Weight      float64
}

// weaselPatterns contains all recognized weasel word patterns.
var weaselPatterns = []WeaselPattern{
	// Anonymous authority - cites sources without specifics
	{regexp.MustCompile(`(?i)\b(some (say|believe|argue|claim|think))\b`), WeaselAnonymousAuthority, "anonymous authority", 1.0},
	{regexp.MustCompile(`(?i)\b(experts (say|believe|agree|claim))\b`), WeaselAnonymousAuthority, "anonymous authority", 1.0},
	{regexp.MustCompile(`(?i)\b(many (people|experts|researchers|scientists))\b`), WeaselAnonymousAuthority, "vague attribution", 0.9},
	{regexp.MustCompile(`(?i)\b(it is (said|believed|thought|known|widely known))\b`), WeaselAnonymousAuthority, "anonymous authority", 1.0},
	{regexp.MustCompile(`(?i)\b(research (shows|suggests|indicates))\b`), WeaselAnonymousAuthority, "uncited research", 0.8},
	{regexp.MustCompile(`(?i)\b(studies (show|suggest|indicate))\b`), WeaselAnonymousAuthority, "uncited studies", 0.8},
	{regexp.MustCompile(`(?i)\b(according to (some|experts|research|studies))\b`), WeaselAnonymousAuthority, "vague attribution", 0.9},
	{regexp.MustCompile(`(?i)\b(critics (say|argue|claim))\b`), WeaselAnonymousAuthority, "anonymous critics", 1.0},
	{regexp.MustCompile(`(?i)\b(observers (note|say|believe))\b`), WeaselAnonymousAuthority, "anonymous observers", 1.0},
	{regexp.MustCompile(`(?i)\b(there is (evidence|research|data) (that|to suggest))\b`), WeaselAnonymousAuthority, "vague evidence claim", 0.9},
	{regexp.MustCompile(`(?i)\b(it has been (shown|proven|demonstrated))\b`), WeaselAnonymousAuthority, "anonymous proof", 1.0},

	// Passive voice patterns - obscures responsibility
	{regexp.MustCompile(`(?i)\b(it is|it was|it has been|it will be)\s+(said|believed|thought|argued|claimed|reported|suggested|recommended|decided|determined)\b`), WeaselPassiveVoice, "passive voice obscures actor", 0.8},
	{regexp.MustCompile(`(?i)\b(mistakes were made)\b`), WeaselPassiveVoice, "passive voice avoids responsibility", 1.0},
	{regexp.MustCompile(`(?i)\b(is|was|were|been|being)\s+(considered|thought|believed|seen|viewed|regarded)\s+(as|to be)\b`), WeaselPassiveVoice, "passive voice", 0.7},

	// Weasel words - create false precision or vagueness
	{regexp.MustCompile(`(?i)\b(up to|as many as|as much as)\s+\d+`), WeaselWeaselWord, "inflated maximum (could be zero)", 1.0},
	{regexp.MustCompile(`(?i)\b(more than|over)\s+\d+`), WeaselWeaselWord, "vague quantity (how much more?)", 0.6},
	{regexp.MustCompile(`(?i)\b(significant|substantial|considerable)\b`), WeaselWeaselWord, "vague magnitude", 0.7},
	{regexp.MustCompile(`(?i)\b(may|might|could)\s+(be|have|result)\b`), WeaselWeaselWord, "uncertain outcome", 0.5},
	{regexp.MustCompile(`(?i)\b(often|sometimes|rarely|frequently)\b`), WeaselWeaselWord, "vague frequency", 0.5},
	{regexp.MustCompile(`(?i)\b(most|many|some|few)\s+(people|users|developers|companies)\b`), WeaselWeaselWord, "vague quantifier", 0.7},
	{regexp.MustCompile(`(?i)\b(widely|generally|commonly|typically)\s+(accepted|used|known|believed)\b`), WeaselWeaselWord, "vague generalization", 0.8},
	{regexp.MustCompile(`(?i)\b(certain|particular|specific)\s+(cases|situations|circumstances)\b`), WeaselWeaselWord, "vague specificity", 0.6},

	// Nominalizations - verbs turned into nouns, making prose dense and vague
	{regexp.MustCompile(`(?i)\b(the (implementation|utilization|optimization|configuration|transformation) of)\b`), WeaselNominalization, "nominalization (use verb form)", 0.8},
	{regexp.MustCompile(`(?i)\b(the (establishment|development|creation|formation) of)\b`), WeaselNominalization, "nominalization (use verb form)", 0.8},
	{regexp.MustCompile(`(?i)\b(the (facilitation|maximization|minimization|prioritization) of)\b`), WeaselNominalization, "nominalization (use verb form)", 0.9},
	{regexp.MustCompile(`(?i)\b(make (an|a) (decision|determination|recommendation|assessment))\b`), WeaselNominalization, "nominalization (just 'decide', 'determine', etc.)", 0.9},
	{regexp.MustCompile(`(?i)\b(perform (an|a) (analysis|evaluation|investigation|examination))\b`), WeaselNominalization, "nominalization (just 'analyze', 'evaluate', etc.)", 0.9},
	{regexp.MustCompile(`(?i)\b(conduct (an|a) (review|study|survey|assessment))\b`), WeaselNominalization, "nominalization (just 'review', 'study', etc.)", 0.8},
	{regexp.MustCompile(`(?i)\b(provide (an|a) (explanation|description|overview|summary))\b`), WeaselNominalization, "nominalization (just 'explain', 'describe', etc.)", 0.8},

	// Hedge words
	{regexp.MustCompile(`(?i)\b(tends to|appears to|seems to)\b`), WeaselHedge, "hedge phrase", 0.7},
	{regexp.MustCompile(`(?i)\b(in (general|principle|theory))\b`), WeaselHedge, "hedge phrase", 0.6},
	{regexp.MustCompile(`(?i)\b(for the most part)\b`), WeaselHedge, "hedge phrase", 0.7},
	{regexp.MustCompile(`(?i)\b(to some (extent|degree))\b`), WeaselHedge, "hedge phrase", 0.8},
	{regexp.MustCompile(`(?i)\b(relatively|comparatively)\b`), WeaselHedge, "relative term without comparison", 0.6},
}

// WeaselMatch represents a detected weasel word or phrase.
type WeaselMatch struct {
	Text        string
	Line        int
	Column      int
	Category    WeaselCategory
	Description string
	Weight      float64
}

// DetectWeasels scans text for weasel words and returns matches.
func DetectWeasels(text string) []WeaselMatch {
	var matches []WeaselMatch

	lines := strings.Split(text, "\n")
	for lineNum, line := range lines {
		for _, pattern := range weaselPatterns {
			locs := pattern.Pattern.FindAllStringIndex(line, -1)
			for _, loc := range locs {
				matches = append(matches, WeaselMatch{
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

// WeaselScore calculates a weasel score for the text.
// Returns a score from 0 (no weasels) to 100 (heavily weaselly).
func WeaselScore(text string, matches []WeaselMatch) float64 {
	if len(text) == 0 {
		return 0
	}

	// Calculate weighted match count
	var weightedCount float64
	for _, m := range matches {
		weightedCount += m.Weight
	}

	// Normalize by sentence count (approximated by periods)
	sentences := strings.Count(text, ".") + strings.Count(text, "!") + strings.Count(text, "?")
	if sentences == 0 {
		sentences = 1
	}

	// Weasels per sentence, capped at 100
	score := (weightedCount / float64(sentences)) * 25 // Scale factor
	if score > 100 {
		score = 100
	}

	return score
}
