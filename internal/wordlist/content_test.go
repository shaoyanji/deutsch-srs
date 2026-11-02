package wordlist

import "testing"

// The verb and the relational adjective share a spelling and a frequency rank,
// so the winner used to be whichever line the extract listed first: "wollen"
// came out as "woolen" — the rare adjective — on one run and "to want" on the
// next.
func TestBuildPrefersTheVerbOverARelationalAdjective(t *testing.T) {
	adjFirst := `{"word":"wollen","pos":"adj","senses":[{"tags":["relational"],"glosses":["wool; woolen, woollen"]}]}
{"word":"wollen","pos":"verb","senses":[{"glosses":["to want; to wish; to desire"]}]}`
	verbFirst := `{"word":"wollen","pos":"verb","senses":[{"glosses":["to want; to wish; to desire"]}]}
{"word":"wollen","pos":"adj","senses":[{"tags":["relational"],"glosses":["wool; woolen, woollen"]}]}`

	for name, entries := range map[string]string{"adjective first": adjFirst, "verb first": verbFirst} {
		cards, _ := build(t, entries, ranksOf("wollen"), Options{})
		if len(cards) != 1 {
			t.Fatalf("%s: cards = %v", name, cards)
		}
		if cards[0].Pos != "verb" || cards[0].Back != "to want; to wish" {
			t.Fatalf("%s: card = %+v, want the verb", name, cards[0])
		}
	}
}

func TestBuildPrefersTheNounOverAPredicateOnlyAdjective(t *testing.T) {
	// "weg" at rank 129 is the adverb "away"; the adjective only exists as "er
	// ist weg" and was hiding der Weg (the path) behind the same rank.
	entries := `{"word":"weg","pos":"adj","senses":[{"tags":["predicative","informal"],"glosses":["unconscious; passed out"]}]}
{"word":"Weg","pos":"noun","senses":[{"glosses":["path, trail, track"]}]}`

	cards, _ := build(t, entries, ranksOf("weg"), Options{})
	if len(cards) != 1 {
		t.Fatalf("cards = %v", cards)
	}
	if cards[0].Front != "Weg" || cards[0].Back != "path, trail, track" {
		t.Fatalf("card = %+v, want der Weg", cards[0])
	}
}

func TestBuildPrefersTheNounOverARelationalAdjective(t *testing.T) {
	entries := `{"word":"plan","pos":"adj","senses":[{"tags":["relational"],"glosses":["planar, flat, level, smooth"]}]}
{"word":"Plan","pos":"noun","senses":[{"glosses":["plan, map"]}]}`

	cards, _ := build(t, entries, ranksOf("plan"), Options{})
	if len(cards) != 1 {
		t.Fatalf("cards = %v", cards)
	}
	if cards[0].Front != "Plan" {
		t.Fatalf("card = %+v, want der Plan", cards[0])
	}
}

func TestBuildContentOnlyDropsABorrowedRank(t *testing.T) {
	// "Eins" is a noun, but the rank belongs to the article "eins".
	entries := `{"word":"eins","pos":"article","senses":[{"glosses":["one"]}]}
{"word":"Eins","pos":"noun","senses":[{"glosses":["one (the digit)"]}]}`

	kept, stats := build(t, entries, ranksOf("eins"), Options{})
	if len(kept) != 1 || !kept[0].Demoted {
		t.Fatalf("without ContentOnly the card should stay, demoted: %+v", kept)
	}
	if stats.Demoted != 1 {
		t.Fatalf("stats.Demoted = %d", stats.Demoted)
	}

	dropped, stats := build(t, entries, ranksOf("eins"), Options{ContentOnly: true})
	if len(dropped) != 0 {
		t.Fatalf("ContentOnly kept %+v", dropped)
	}
	if stats.Shadowed != 1 {
		t.Fatalf("stats.Shadowed = %d, want 1", stats.Shadowed)
	}
}

func TestBuildExcludeDropsASpellingWhateverItsPartOfSpeech(t *testing.T) {
	entries := `{"word":"Ja","pos":"noun","senses":[{"glosses":["a yes"]}]}
{"word":"Haus","pos":"noun","senses":[{"glosses":["house"]}]}`

	cards, stats := build(t, entries, ranksOf("Ja", "Haus"), Options{Exclude: map[string]bool{"ja": true}})
	if len(cards) != 1 || cards[0].Front != "Haus" {
		t.Fatalf("cards = %+v", cards)
	}
	if stats.Excluded != 1 {
		t.Fatalf("stats.Excluded = %d, want 1 (matching is case-insensitive)", stats.Excluded)
	}
}

func TestBuildExcludeIsOptOut(t *testing.T) {
	entries := `{"word":"Ja","pos":"noun","senses":[{"glosses":["a yes"]}]}`
	cards, _ := build(t, entries, ranksOf("Ja"), Options{})
	if len(cards) != 1 {
		t.Fatalf("cards = %v, want the card when no list is given", cards)
	}
}
