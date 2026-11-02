// Package wordlist turns a kaikki.org (Wiktionary) extract plus an
// OpenSubtitles frequency list into a vocabulary deck ordered by how often the
// words actually get used.
package wordlist

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
)

// LevelRanks are the frequency ceilings standing in for CEFR levels. Wiktionary
// entries carry no CEFR tag, so a level here is a frequency band: every lemma
// down to that OpenSubtitles rank. It is a proxy, and a coarse one — treat the
// levels as vocabulary-size choices, not as an exam syllabus.
var LevelRanks = map[string]int{
	"a1": 600,
	"a2": 1500,
	"b1": 3000,
	"b2": 6000,
}

// DefaultPos is the set worth learning as single words: the four that carry most
// of a sentence. Function words and phrases are left out because they are learnt
// in context, not as an isolated translation pair.
var DefaultPos = []string{"noun", "verb", "adj", "adv"}

// posLabels tags a card with the part of speech under its German name.
var posLabels = map[string]string{
	"noun":    "nomen",
	"verb":    "verb",
	"adj":     "adjektiv",
	"adv":     "adverb",
	"phrase":  "redewendung",
	"prep":    "praeposition",
	"conj":    "konjunktion",
	"num":     "numerale",
	"interj":  "interjektion",
	"pron":    "pronomen",
	"det":     "artikel",
	"postp":   "postposition",
	"proverb": "sprichwort",
}

// metaGlosses describe an entry instead of translating it; a card built from one
// of these teaches nothing.
var metaGlosses = []string{
	"alternative form", "alternative spelling", "alternate spelling", "alternate form",
	"obsolete spelling", "misspelling", "plural of", "singular of", "inflection of",
	"form of", "abbreviation of", "initialism of", "short form of", "short for",
	"comparative of", "superlative of", "past participle of", "past tense of",
	"present participle of", "first-person", "second-person", "third-person",
	"genitive of", "dative of", "accusative of", "nominative of", "dated form of",
	"nonstandard spelling", "eye dialect", "literally ", "cf. ", "see also",
}

// metaTags mark a sense that is not a usable translation on its own. A sense
// tagged in-compounds only exists inside compounds — "ein" as "motion into
// something" is the prefix ein-, not the word — so a card built from it is junk.
var metaTags = map[string]bool{
	"obsolete": true, "archaic": true, "nonstandard": true, "misspelling": true,
	"form-of": true, "alt-of": true, "alternative": true,
	"error-unrecognized-form": true, "no-gloss": true, "in-compounds": true,
	// "haben" as "forms the perfect aspect" is grammar, not the word's meaning;
	// the lexical sense that follows it is the card a learner needs.
	"auxiliary": true,
}

// nonWordPos are morphemes, not words: a deck of prefixes teaches nothing.
var nonWordPos = map[string]bool{
	"prefix": true, "suffix": true, "infix": true, "circumfix": true,
	"combining-form": true, "morpheme": true, "character": true,
}

// functionPos are the closed-class parts of speech. Adverbs are deliberately not
// here: "gestern" and "immer" are vocabulary, even though "mit" and "aber" are
// also filed as adverbs.
var functionPos = map[string]bool{
	"pron": true, "det": true, "article": true, "prep": true, "conj": true,
	"particle": true, "num": true, "contraction": true, "postp": true,
}

// marginalPos are the parts of speech where sharing a spelling with a closed-class
// word means the reading is metalinguistic or grammatical rather than a word in
// its own right — "das Ich" (the ego), "Es" (E-flat), "mit" and "aber" as adverbs.
// Verbs are exempt on purpose: "sein" and "haben" share their spelling with
// function words and are still core vocabulary.
var marginalPos = map[string]bool{"noun": true, "adj": true, "adv": true}

// formTags mark a sense that exists only to point at another lemma.
var formTags = map[string]bool{"form-of": true, "alt-of": true, "alternative": true}

// Options controls what the builder keeps.
type Options struct {
	// Pos is the parts of speech to keep. Nil means DefaultPos.
	Pos []string
	// MaxRank drops lemmas rarer than this frequency rank. 0 keeps every ranked
	// lemma, however rare.
	MaxRank int
	// AllowUnranked keeps lemmas the frequency list has never seen; they sort
	// after the ranked ones.
	AllowUnranked bool
	// Limit caps the number of cards. 0 keeps all.
	Limit int
	// MaxGloss trims translations to this many characters. 0 means 120.
	MaxGloss int
	// Level, when set, tags every card with that level (a1..b2).
	Level string
}

// Card is one word/translation pair.
type Card struct {
	Front string
	Back  string
	Pos   string
	Rank  int
	Tags  []string
	// Demoted marks a card whose frequency rank comes from a different word that
	// happens to be spelled the same lowercased. It is kept, but ordered after
	// every properly ranked card.
	Demoted bool
}

// Stats reports what happened to the input, so a build can be audited instead of
// trusted.
type Stats struct {
	Entries    int
	Unparsable int
	NoSense    int
	WrongPos   int
	FormOf     int
	Unranked   int
	TooRare    int
	NoGloss    int
	Duplicate  int
	Demoted    int
	Kept       int
}

type sense struct {
	Glosses []string `json:"glosses"`
	Tags    []string `json:"tags"`
	FormOf  any      `json:"form_of"`
	AltOf   any      `json:"alt_of"`
}

type entry struct {
	Word   string  `json:"word"`
	Pos    string  `json:"pos"`
	Lang   string  `json:"lang"`
	Senses []sense `json:"senses"`
}

// LoadFrequency reads an OpenSubtitles-style word list ("word count" per line,
// most frequent first) and returns word -> 1-based rank.
func LoadFrequency(r io.Reader) (map[string]int, error) {
	ranks := make(map[string]int, 4096)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	rank := 0
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		rank++
		word := strings.ToLower(fields[0])
		if _, seen := ranks[word]; !seen {
			ranks[word] = rank
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read frequency list: %w", err)
	}
	return ranks, nil
}

// Build reads a kaikki JSONL extract and returns one card per lemma, ordered by
// frequency. Entries that are inflected forms, that only describe another word,
// or that belong to a part of speech we are not keeping are dropped and counted.
func Build(r io.Reader, ranks map[string]int, opts Options) ([]Card, Stats, error) {
	var stats Stats
	pos := opts.Pos
	if len(pos) == 0 {
		pos = DefaultPos
	}
	keepPos := make(map[string]bool, len(pos))
	for _, p := range pos {
		keepPos[normalizePos(p)] = true
	}
	maxGloss := opts.MaxGloss
	if maxGloss <= 0 {
		maxGloss = 120
	}

	byWord := make(map[string]Card, 4096)
	// Lowercase spellings already claimed by a closed-class lemma. A noun or
	// adjective sharing one ("Ich" from ich, "Sein" from sein) inherited its rank
	// from the other reading; it did not earn it.
	classShadow := make(map[string]bool, 1<<12)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		stats.Entries++
		var e entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			// A pretty-printed extract arrives here one line at a time; counting
			// beats letting the deck quietly shrink.
			stats.Unparsable++
			continue
		}
		if functionPos[normalizePos(e.Pos)] {
			classShadow[strings.ToLower(strings.TrimSpace(e.Word))] = true
		}
		card, reason := buildCard(e, isFormEntry(e), ranks, keepPos, opts, maxGloss)
		switch reason {
		case reasonNone:
		case reasonPos:
			stats.WrongPos++
			continue
		case reasonSense:
			stats.NoSense++
			continue
		case reasonForm:
			stats.FormOf++
			continue
		case reasonUnranked:
			stats.Unranked++
			continue
		case reasonRare:
			stats.TooRare++
			continue
		default:
			stats.NoGloss++
			continue
		}
		if prev, dup := byWord[card.Front]; dup {
			stats.Duplicate++
			if prev.Rank != 0 && prev.Rank <= card.Rank {
				continue
			}
		}
		byWord[card.Front] = card
	}
	if err := scanner.Err(); err != nil {
		return nil, stats, fmt.Errorf("read kaikki extract: %w", err)
	}

	// One card per headword, and "headword" is what the store matches on: deck
	// plus front, case-insensitively. Keeping both "Ja" and "ja" would put a card
	// in the file that the import then silently drops.
	groups := make(map[string][]Card, len(byWord))
	for _, c := range byWord {
		lower := strings.ToLower(c.Front)
		// Only a closed-class shadow demotes, and only for the marginal parts of
		// speech. Inflected forms are not a usable signal here: "Zeuge" and "Reifen"
		// share their spelling with the verb forms "zeuge" and "reifen" and are
		// ordinary nouns — demoting on that basis cost 490 real cards.
		c.Demoted = marginalPos[c.Pos] && classShadow[lower]
		groups[lower] = append(groups[lower], c)
	}

	cards := make([]Card, 0, len(groups))
	for _, candidates := range groups {
		if len(candidates) > 1 {
			stats.Duplicate += len(candidates) - 1
		}
		best := candidates[0]
		for _, c := range candidates[1:] {
			if betterCard(c, best) {
				best = c
			}
		}
		if best.Demoted {
			stats.Demoted++
		}
		cards = append(cards, best)
	}
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Demoted != cards[j].Demoted {
			return !cards[i].Demoted
		}
		if cards[i].Rank != cards[j].Rank {
			if cards[i].Rank == 0 {
				return false
			}
			if cards[j].Rank == 0 {
				return true
			}
			return cards[i].Rank < cards[j].Rank
		}
		return cards[i].Front < cards[j].Front
	})

	if opts.Limit > 0 && len(cards) > opts.Limit {
		cards = cards[:opts.Limit]
	}
	stats.Kept = len(cards)
	return cards, stats, nil
}

// betterCard decides which of two readings of the same spelling to keep: an
// undemoted card beats a demoted one, a better rank beats a worse one, and a
// lowercase spelling beats a capitalised noun built on it ("ja" over "Ja").
func betterCard(a, b Card) bool {
	if a.Demoted != b.Demoted {
		return !a.Demoted
	}
	aRank, bRank := a.Rank, b.Rank
	if aRank == 0 {
		aRank = int(^uint(0) >> 1)
	}
	if bRank == 0 {
		bRank = int(^uint(0) >> 1)
	}
	if aRank != bRank {
		return aRank < bRank
	}
	aLower := a.Front == strings.ToLower(a.Front)
	bLower := b.Front == strings.ToLower(b.Front)
	if aLower != bLower {
		return aLower
	}
	return a.Front < b.Front
}

const (
	reasonNone     = ""
	reasonPos      = "pos"
	reasonSense    = "sense"
	reasonForm     = "form"
	reasonUnranked = "unranked"
	reasonRare     = "rare"
	reasonGloss    = "gloss"
)

func buildCard(e entry, form bool, ranks map[string]int, keepPos map[string]bool, opts Options, maxGloss int) (Card, string) {
	word := strings.TrimSpace(e.Word)
	pos := normalizePos(e.Pos)
	if word == "" || !keepPos[pos] || nonWordPos[pos] {
		return Card{}, reasonPos
	}
	if len(e.Senses) == 0 {
		return Card{}, reasonSense
	}
	if form {
		return Card{}, reasonForm
	}

	rank := ranks[strings.ToLower(word)]
	if rank == 0 && !opts.AllowUnranked {
		return Card{}, reasonUnranked
	}
	if opts.MaxRank > 0 && rank > opts.MaxRank {
		return Card{}, reasonRare
	}

	back := pickGloss(e.Senses, word, maxGloss)
	if back == "" {
		return Card{}, reasonGloss
	}

	tags := []string{posLabel(pos)}
	if opts.Level != "" {
		tags = append(tags, strings.ToLower(strings.TrimSpace(opts.Level)))
	}
	return Card{Front: word, Back: back, Pos: pos, Rank: rank, Tags: tags}, reasonNone
}

// isFormEntry reports whether every sense of the entry points at another lemma:
// an inflected form, an alternative spelling, or a plural.
func isFormEntry(e entry) bool {
	for _, s := range e.Senses {
		if s.FormOf != nil || s.AltOf != nil || hasTag(s.Tags, formTags) {
			return true
		}
	}
	return false
}

// pickGloss returns the first sense that actually translates the word. Wiktionary
// lists senses in order of primacy, so the first usable one is the right one;
// merging further senses mostly adds noise ("free; unenslaved; unimprisoned").
func pickGloss(senses []sense, word string, maxGloss int) string {
	for _, s := range senses {
		if len(s.Glosses) == 0 || hasTag(s.Tags, metaTags) {
			continue
		}
		// A sense can open with a header ("As a copulative verb:") and carry the
		// translation in the gloss after it, so every gloss of the sense is tried.
		for _, raw := range s.Glosses {
			if strings.HasSuffix(strings.TrimSpace(raw), ":") {
				continue
			}
			gloss := cleanGloss(raw, maxGloss)
			if gloss == "" || isMetaGloss(gloss) || strings.EqualFold(gloss, word) {
				continue
			}
			return gloss
		}
	}
	return ""
}

// cleanGloss strips the dictionary apparatus — leading usage labels, chained
// alternatives, "see also" tails — and trims to a readable length.
func cleanGloss(raw string, maxGloss int) string {
	g := strings.Join(strings.Fields(raw), " ")
	for {
		trimmed := strings.TrimSpace(g)
		if strings.HasPrefix(trimmed, "(") {
			if end := strings.Index(trimmed, ")"); end >= 0 {
				g = trimmed[end+1:]
				continue
			}
		}
		if strings.HasPrefix(trimmed, "[") {
			if end := strings.Index(trimmed, "]"); end >= 0 {
				g = trimmed[end+1:]
				continue
			}
		}
		g = trimmed
		break
	}
	g = strings.Trim(g, " .,;:—–-")
	// Keep at most two alternatives: "free; unrestricted; licentious" teaches
	// less than "free; unrestricted". Separators inside parentheses do not count,
	// so "to have; to own (to possess; to hold)" stays one alternative.
	if parts := splitAlternatives(g); len(parts) > 2 {
		g = strings.Join(parts[:2], "; ")
	}
	g = trimGlossTails(g)
	if maxGloss > 0 && len(g) > maxGloss {
		if cut := strings.LastIndexAny(g[:maxGloss], " ,"); cut > maxGloss/2 {
			g = g[:cut]
		} else {
			g = g[:maxGloss]
		}
		// Truncation can leave a parenthetical open, so the tails are trimmed again.
		g = trimGlossTails(strings.TrimRight(g, " ,;"))
	}
	if !hasWordRune(g) {
		return ""
	}
	return g
}

// splitAlternatives splits on "; " outside parentheses.
func splitAlternatives(g string) []string {
	parts := make([]string, 0, 3)
	depth, start := 0, 0
	for i := 0; i < len(g); i++ {
		switch g[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ';':
			if depth == 0 && i+1 < len(g) && g[i+1] == ' ' {
				parts = append(parts, g[start:i])
				start = i + 2
				i++
			}
		}
	}
	return append(parts, g[start:])
}

// trimGlossTails drops trailing explanations — "too (excessively)", the half of a
// parenthetical the length cap cut in two — leaving the translation itself.
func trimGlossTails(g string) string {
	for {
		trimmed := strings.TrimRight(g, " ,;")
		switch {
		case strings.HasSuffix(trimmed, ")"):
			open := strings.LastIndex(trimmed, "(")
			if open <= 0 {
				return trimmed
			}
			rest := strings.TrimRight(trimmed[:open], " ,;")
			if !hasWordRune(rest) {
				return trimmed
			}
			g = rest
		case strings.Count(trimmed, "(") > strings.Count(trimmed, ")"):
			open := strings.LastIndex(trimmed, "(")
			if open <= 0 {
				return trimmed
			}
			g = trimmed[:open]
		default:
			return trimmed
		}
	}
}

// isMetaGloss reports whether a gloss describes the entry instead of translating it.
func isMetaGloss(g string) bool {
	low := strings.ToLower(g)
	for _, m := range metaGlosses {
		if strings.HasPrefix(low, m) || strings.Contains(low, "; "+m) || strings.Contains(low, " "+m) {
			return true
		}
	}
	return false
}

func hasTag(tags []string, set map[string]bool) bool {
	for _, t := range tags {
		if set[strings.ToLower(t)] {
			return true
		}
	}
	return false
}

// hasWordRune rejects glosses that carry no word, e.g. "†" or "—".
func hasWordRune(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func normalizePos(pos string) string {
	return strings.ToLower(strings.TrimSpace(pos))
}

func posLabel(pos string) string {
	if label, ok := posLabels[pos]; ok {
		return label
	}
	return pos
}

// LevelLimit maps a level name to its frequency ceiling, or 0 when unknown.
func LevelLimit(level string) int {
	return LevelRanks[strings.ToLower(strings.TrimSpace(level))]
}
