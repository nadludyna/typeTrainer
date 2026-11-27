package profile

import (
	"strings"
	"testing"
)

var wordParams = WordParams{Rounds: 60, OwnHits: 3, Slack: 1.1}

// round types the words at ms per letter; slow words take 3 times as long,
// bad words get a mistake on their second letter.
func round(words string, ms float64, slow, bad string) []Key {
	var out []Key
	for _, w := range strings.Fields(words) {
		for i, c := range w {
			k := Key{Char: string(c), Ms: ms}
			if strings.Contains(" "+slow+" ", " "+w+" ") {
				k.Ms = 3 * ms
			}
			if i == 1 && strings.Contains(" "+bad+" ", " "+w+" ") {
				k.Errors = 1
			}
			out = append(out, k)
		}
		out = append(out, Key{Char: " ", Ms: ms})
	}
	return out
}

func owned(v Vocabulary) string {
	var out []string
	for _, w := range []string{"the", "quick", "brown", "fox", "jumps"} {
		if v.Owned[w] {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

func TestReadWords_NoData(t *testing.T) {
	for _, rounds := range [][][]Key{nil, {}, {{{Char: " "}, {Char: "a"}, {Char: " "}}}} {
		if v := ReadWords(rounds, wordParams); v.Owned == nil || len(v.Owned) != 0 || v.Usual != 0 {
			t.Fatalf("nothing to read: %+v", v)
		}
	}
}

func TestReadWords_ThreeCleanFastRoundsOwnAWord(t *testing.T) {
	const text = "the quick brown fox jumps"
	two := [][]Key{round(text, 150, "", ""), round(text, 150, "", "")}
	if got := owned(ReadWords(two, wordParams)); got != "" {
		t.Fatalf("two rounds own nothing yet: %q", got)
	}
	three := append(two, round(text, 150, "", ""))
	if v := ReadWords(three, wordParams); owned(v) != text || v.Usual != 150 || v.OwnedOf([]string{"the", "fox", "zebra"}) != 2 {
		t.Fatalf("three clean rounds: %q, usual %v", owned(v), v.Usual)
	}
	// the same word three times in one round is one hit
	if got := owned(ReadWords([][]Key{round("fox fox fox the the the", 150, "", "")}, wordParams)); got != "" {
		t.Fatalf("one round is one hit, however often the word comes: %q", got)
	}
}

func TestReadWords_SlowOrWrongDoesNotCount(t *testing.T) {
	const text = "the quick brown fox jumps"
	rounds := [][]Key{round(text, 150, "quick", "fox"), round(text, 150, "quick", "fox"), round(text, 150, "quick", "fox")}
	if got := owned(ReadWords(rounds, wordParams)); got != "the brown jumps" {
		t.Fatalf("a slow word and a word with a mistake are not owned: %q", got)
	}
}

func TestReadWords_AMistakeTakesAHitAway(t *testing.T) {
	const text = "the quick brown fox jumps"
	clean := round(text, 150, "", "")
	rounds := [][]Key{clean, clean, clean, round(text, 150, "", "fox")}
	if got := owned(ReadWords(rounds, wordParams)); got != "the quick brown jumps" {
		t.Fatalf("a mistake after three hits loses the word: %q", got)
	}
	if got := owned(ReadWords(append(rounds, clean), wordParams)); got != text {
		t.Fatalf("one more clean round wins it back: %q", got)
	}
	// clean and wrong in the same round: the worst try counts
	mixed := append(round("fox", 150, "", ""), round("fox", 150, "", "fox")...)
	if got := owned(ReadWords([][]Key{clean, clean, mixed}, wordParams)); strings.Contains(got, "fox") {
		t.Fatalf("a round with a wrong try is not a hit: %q", got)
	}
}
