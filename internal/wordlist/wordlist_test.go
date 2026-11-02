package wordlist

import (
	"strings"
	"testing"
)

func ranksOf(words ...string) map[string]int {
	out := make(map[string]int, len(words))
	for i, w := range words {
		out[strings.ToLower(w)] = i + 1
	}
	return out
}

func build(t *testing.T, entries string, ranks map[string]int, opts Options) ([]Card, Stats) {
	t.Helper()
	cards, stats, err := Build(strings.NewReader(entries), ranks, opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return cards, stats
}

func TestBuildKeepsUsableSenseOnly(t *testing.T) {
	entries := `{"word":"frei","pos":"adj","senses":[{"tags":["obsolete"],"glosses":["obsolete form of freien"]},{"glosses":["(Austria) free; unenslaved; unimprisoned"]},{"glosses":["independent"]}]}`

	cards, _ := build(t, entries, ranksOf("frei"), Options{})
	if len(cards) != 1 {
		t.Fatalf("cards = %v", cards)
	}
	// The obsolete sense is skipped, the usage label is stripped, and the chain
	// of alternatives is cut to two.
	if cards[0].Back != "free; unenslaved" {
		t.Fatalf("back = %q", cards[0].Back)
	}
}

func TestBuildDropsInflectedForms(t *testing.T) {
	entries := `{"word":"ist","pos":"verb","senses":[{"form_of":[{"word":"sein"}],"glosses":["third-person singular present of sein"]}]}
{"word":"Haus","pos":"noun","senses":[{"glosses":["house"]}]}`

	cards, stats := build(t, entries, ranksOf("ist", "Haus"), Options{})
	if len(cards) != 1 || cards[0].Front != "Haus" {
		t.Fatalf("cards = %v", cards)
	}
	if stats.FormOf != 1 {
		t.Fatalf("stats.FormOf = %d, want 1", stats.FormOf)
	}
}

func TestBuildDropsMetaGlossWithoutFormOfField(t *testing.T) {
	// A sense can describe another lemma through its gloss alone, with no
	// form_of object to check.
	entries := `{"word":"Häuser","pos":"noun","senses":[{"glosses":["plural of Haus"]}]}`
	cards, stats := build(t, entries, ranksOf("Häuser"), Options{})
	if len(cards) != 0 {
		t.Fatalf("cards = %v, want none", cards)
	}
	if stats.NoGloss != 1 {
		t.Fatalf("stats.NoGloss = %d, want 1", stats.NoGloss)
	}
}

func TestBuildFiltersPartOfSpeech(t *testing.T) {
	entries := `{"word":"der","pos":"det","senses":[{"glosses":["the"]}]}
{"word":"laufen","pos":"verb","senses":[{"glosses":["to run"]}]}`

	cards, stats := build(t, entries, ranksOf("der", "laufen"), Options{})
	if len(cards) != 1 || cards[0].Front != "laufen" {
		t.Fatalf("cards = %v", cards)
	}
	if stats.WrongPos != 1 {
		t.Fatalf("stats.WrongPos = %d, want 1", stats.WrongPos)
	}

	cards, _ = build(t, entries, ranksOf("der", "laufen"), Options{Pos: []string{"det"}})
	if len(cards) != 1 || cards[0].Front != "der" {
		t.Fatalf("custom -pos ignored: %v", cards)
	}
}

func TestBuildOrdersByFrequencyAndAppliesCeiling(t *testing.T) {
	entries := `{"word":"selten","pos":"adj","senses":[{"glosses":["rare"]}]}
{"word":"Haus","pos":"noun","senses":[{"glosses":["house"]}]}
{"word":"Zeit","pos":"noun","senses":[{"glosses":["time"]}]}`

	cards, _ := build(t, entries, ranksOf("Haus", "Zeit", "selten"), Options{})
	if len(cards) != 3 || cards[0].Front != "Haus" || cards[1].Front != "Zeit" {
		t.Fatalf("order = %v", cards)
	}

	cards, stats := build(t, entries, ranksOf("Haus", "Zeit", "selten"), Options{MaxRank: 2})
	if len(cards) != 2 {
		t.Fatalf("ceiling not applied: %v", cards)
	}
	if stats.TooRare != 1 {
		t.Fatalf("stats.TooRare = %d, want 1", stats.TooRare)
	}
}

func TestBuildDropsUnrankedUnlessAsked(t *testing.T) {
	entries := `{"word":"Kaugummiautomat","pos":"noun","senses":[{"glosses":["gum machine"]}]}`

	cards, stats := build(t, entries, ranksOf("Haus"), Options{})
	if len(cards) != 0 || stats.Unranked != 1 {
		t.Fatalf("unranked kept: cards=%v stats=%+v", cards, stats)
	}

	cards, _ = build(t, entries, ranksOf("Haus"), Options{AllowUnranked: true})
	if len(cards) != 1 {
		t.Fatalf("AllowUnranked ignored: %v", cards)
	}
}

func TestBuildKeepsOneCardPerHeadword(t *testing.T) {
	entries := `{"word":"Bank","pos":"noun","senses":[{"glosses":["bench"]}]}
{"word":"Bank","pos":"noun","senses":[{"glosses":["financial institution"]}]}`

	cards, stats := build(t, entries, ranksOf("Bank"), Options{})
	if len(cards) != 1 {
		t.Fatalf("cards = %v", cards)
	}
	if cards[0].Back != "bench" {
		t.Fatalf("first sense should win, got %q", cards[0].Back)
	}
	if stats.Duplicate != 1 {
		t.Fatalf("stats.Duplicate = %d", stats.Duplicate)
	}
}

func TestBuildPreservesUmlautsAndTags(t *testing.T) {
	entries := `{"word":"schön","pos":"adj","senses":[{"glosses":["beautiful"]}]}`
	cards, _ := build(t, entries, ranksOf("schön"), Options{Level: "B1"})
	if len(cards) != 1 || cards[0].Front != "schön" {
		t.Fatalf("cards = %v", cards)
	}
	wantTags := []string{"adjektiv", "b1"}
	if len(cards[0].Tags) != len(wantTags) {
		t.Fatalf("tags = %v", cards[0].Tags)
	}
	for i, want := range wantTags {
		if cards[0].Tags[i] != want {
			t.Fatalf("tags = %v, want %v", cards[0].Tags, wantTags)
		}
	}
}

func TestBuildTrimsLongGloss(t *testing.T) {
	long := "a very long and windy definition that no learner needs on a card front; " +
		"with an equally long second alternative that should be cut away entirely"
	entries := `{"word":"Ding","pos":"noun","senses":[{"glosses":["` + long + `"]}]}`

	cards, _ := build(t, entries, ranksOf("Ding"), Options{MaxGloss: 40})
	if len(cards) != 1 {
		t.Fatalf("cards = %v", cards)
	}
	if len(cards[0].Back) > 40 {
		t.Fatalf("gloss not trimmed: %q (%d chars)", cards[0].Back, len(cards[0].Back))
	}
	if strings.HasSuffix(cards[0].Back, " ") || strings.HasSuffix(cards[0].Back, ",") {
		t.Fatalf("trim left trailing punctuation: %q", cards[0].Back)
	}
}

func TestBuildDoesNotDemoteOnInflectedFormShadowsAlone(t *testing.T) {
	// "Zeuge" (witness) and the verb form "zeuge" share a spelling, and so do
	// "Reifen" and "reifen". Treating that as a shadow demoted hundreds of real
	// nouns, so only closed-class shadows demote.
	entries := `{"word":"zeugen","pos":"verb","senses":[{"glosses":["to testify"]}]}
{"word":"zeuge","pos":"verb","senses":[{"form_of":[{"word":"zeugen"}],"glosses":["first-person singular present of zeugen"]}]}
{"word":"Zeuge","pos":"noun","senses":[{"glosses":["witness"]}]}`

	cards, stats := build(t, entries, ranksOf("zeugen", "zeuge"), Options{})
	if len(cards) != 2 {
		t.Fatalf("cards = %+v", cards)
	}
	for _, c := range cards {
		if c.Demoted {
			t.Fatalf("%s demoted, but an inflected form is not a reliable shadow", c.Front)
		}
	}
	if stats.Demoted != 0 {
		t.Fatalf("stats.Demoted = %d, want 0", stats.Demoted)
	}
}

func TestBuildKeepsCapitalisedNounsWithoutAShadow(t *testing.T) {
	// No lowercase "haus" lemma exists, so "Haus" keeps the rank it earned.
	entries := `{"word":"Haus","pos":"noun","senses":[{"glosses":["house"]}]}`
	cards, _ := build(t, entries, ranksOf("Haus"), Options{})
	if len(cards) != 1 || cards[0].Demoted {
		t.Fatalf("cards = %+v, want one undemoted card", cards)
	}
}

func TestBuildSkipsCompoundOnlySenses(t *testing.T) {
	// "ein" as an adverb only means "into" as the prefix ein-, and kaikki says so
	// with an in-compounds tag.
	entries := `{"word":"ein","pos":"adv","senses":[{"tags":["in-compounds"],"glosses":["indicating motion into something"]}]}`
	cards, stats := build(t, entries, ranksOf("ein"), Options{})
	if len(cards) != 0 {
		t.Fatalf("cards = %v, want none", cards)
	}
	if stats.NoGloss != 1 {
		t.Fatalf("stats.NoGloss = %d, want 1", stats.NoGloss)
	}
}

func TestBuildDemotesNominalisedFunctionWords(t *testing.T) {
	// "Ich" (the ego) inherits the rank of the pronoun "ich", which is the most
	// frequent token in the language. It is a real word, so it stays, but it must
	// not lead an A1 deck.
	entries := `{"word":"ich","pos":"pron","senses":[{"glosses":["I"]}]}
{"word":"Ich","pos":"noun","senses":[{"glosses":["ego"]}]}
{"word":"Haus","pos":"noun","senses":[{"glosses":["house"]}]}`

	cards, stats := build(t, entries, ranksOf("ich", "Haus"), Options{})
	if len(cards) != 2 {
		t.Fatalf("cards = %v", cards)
	}
	if cards[0].Front != "Haus" || cards[1].Front != "Ich" {
		t.Fatalf("order = %v, want Haus then Ich", []string{cards[0].Front, cards[1].Front})
	}
	if !cards[1].Demoted {
		t.Fatalf("Ich not marked demoted")
	}
	if stats.Demoted != 1 {
		t.Fatalf("stats.Demoted = %d, want 1", stats.Demoted)
	}
}

func TestBuildKeepsVerbsThatShareAFunctionWordSpelling(t *testing.T) {
	// "sein" is both a possessive pronoun and the core verb; demoting the verb
	// would gut the deck.
	entries := `{"word":"sein","pos":"det","senses":[{"glosses":["his"]}]}
{"word":"sein","pos":"verb","senses":[{"glosses":["to be"]}]}`

	cards, stats := build(t, entries, ranksOf("sein"), Options{})
	if len(cards) != 1 || cards[0].Pos != "verb" || cards[0].Demoted {
		t.Fatalf("cards = %+v", cards)
	}
	if stats.Demoted != 0 {
		t.Fatalf("stats.Demoted = %d, want 0", stats.Demoted)
	}
}

func TestBuildCountsUnparsableLines(t *testing.T) {
	// One JSON object per line is the kaikki contract; a pretty-printed file is a
	// mistake the build should report instead of silently shrinking the deck.
	entries := "not json at all\n{\"word\":\"Haus\",\"pos\":\"noun\",\"senses\":[{\"glosses\":[\"house\"]}]}"
	cards, stats := build(t, entries, ranksOf("Haus"), Options{})
	if len(cards) != 1 {
		t.Fatalf("cards = %v", cards)
	}
	if stats.Unparsable != 1 {
		t.Fatalf("stats.Unparsable = %d, want 1", stats.Unparsable)
	}
}

func TestBuildStopsAtLimit(t *testing.T) {
	entries := `{"word":"eins","pos":"num","senses":[{"glosses":["one"]}]}
{"word":"zwei","pos":"num","senses":[{"glosses":["two"]}]}
{"word":"drei","pos":"num","senses":[{"glosses":["three"]}]}`
	cards, stats := build(t, entries, ranksOf("eins", "zwei", "drei"), Options{Pos: []string{"num"}, Limit: 2})
	if len(cards) != 2 || stats.Kept != 2 {
		t.Fatalf("cards = %v, stats.Kept = %d", cards, stats.Kept)
	}
}

func TestLoadFrequency(t *testing.T) {
	ranks, err := LoadFrequency(strings.NewReader("ich 5890279\nsie 3806767\nich 100\n"))
	if err != nil {
		t.Fatalf("LoadFrequency: %v", err)
	}
	if ranks["ich"] != 1 || ranks["sie"] != 2 {
		t.Fatalf("ranks = %v", ranks)
	}
}

func TestLevelLimit(t *testing.T) {
	if LevelLimit("B2") != 6000 || LevelLimit("a1") != 600 || LevelLimit("c1") != 0 {
		t.Fatalf("level mapping changed: a1=%d b2=%d c1=%d", LevelLimit("a1"), LevelLimit("B2"), LevelLimit("C1"))
	}
}

func TestBuildMinRankKeepsBandsDisjoint(t *testing.T) {
	entries := `{"word":"eins","pos":"adj","senses":[{"glosses":["one"]}]}
{"word":"zwei","pos":"adj","senses":[{"glosses":["two"]}]}
{"word":"drei","pos":"adj","senses":[{"glosses":["three"]}]}`
	ranks := ranksOf("eins", "zwei", "drei")

	cards, stats := build(t, entries, ranks, Options{MinRank: 2, MaxRank: 3})
	if len(cards) != 2 || cards[0].Front != "zwei" || cards[1].Front != "drei" {
		t.Fatalf("band 2-3 = %v", cards)
	}
	if stats.TooCommon != 1 {
		t.Fatalf("stats.TooCommon = %d, want 1", stats.TooCommon)
	}
}

func TestLevelFloorMakesPerLevelBands(t *testing.T) {
	want := map[string]int{"a1": 0, "a2": 600, "b1": 1500, "b2": 3000, "c1": 0}
	for level, floor := range want {
		if got := LevelFloor(level); got != floor {
			t.Fatalf("LevelFloor(%q) = %d, want %d", level, got, floor)
		}
	}
	// Each band starts one rank past the previous level's ceiling, so the four
	// decks together hold every word exactly once.
	order := []string{"a1", "a2", "b1", "b2"}
	for i := 1; i < len(order); i++ {
		if LevelFloor(order[i]) != LevelLimit(order[i-1]) {
			t.Fatalf("%s does not start where the previous level ends", order[i])
		}
	}
}

func TestBuildKeepsEntryWithOneFormSense(t *testing.T) {
	// The real kaikki entry for "Wasser" ends with an abbreviation sense
	// ("clipping of Mineralwasser"). Treating any form sense as a form entry
	// dropped the whole word from every deck.
	entries := `{"word":"Wasser","pos":"noun","senses":[` +
		`{"glosses":["water (H\u2082O)"],"tags":["neuter"]},` +
		`{"glosses":["abbreviation of Mineralwasser"],"tags":["alt-of","clipping"]}]}`

	cards, stats := build(t, entries, ranksOf("Wasser"), Options{})
	if len(cards) != 1 || cards[0].Front != "Wasser" {
		t.Fatalf("cards = %v, want the noun", cards)
	}
	if cards[0].Back != "water" {
		t.Fatalf("back = %q, want the first translation", cards[0].Back)
	}
	if stats.FormOf != 0 {
		t.Fatalf("stats.FormOf = %d, want 0", stats.FormOf)
	}
}

func TestBuildStillDropsPureFormEntries(t *testing.T) {
	entries := `{"word":"ist","pos":"verb","senses":[` +
		`{"form_of":[{"word":"sein"}],"glosses":["third-person singular present of sein"]}]}`

	cards, stats := build(t, entries, ranksOf("ist"), Options{})
	if len(cards) != 0 {
		t.Fatalf("cards = %v, want none", cards)
	}
	if stats.FormOf != 1 {
		t.Fatalf("stats.FormOf = %d, want 1", stats.FormOf)
	}
}
