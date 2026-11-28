package freetype

import (
	"math"
	"math/rand"
	"slices"
	"testing"
)

var testParams = Params{
	PortionWords: 6, MaxStartWords: 60, RecentWords: 20, RepeatPenalty: 0.3,
	RhythmRounds: 20, RhythmMaxMs: 2000, RhythmMinPresses: 50, StumbleSpreads: 3, StumbleLeast: 1.5, StumblePrior: 3,
	StumbleMinPresses: 8, StumbleMinCount: 2, StumbleRatio: 2, StumblePairs: 3, StumbleBoost: 2,
}

func plain() Params { return testParams }

func TestDraw_Uniform(t *testing.T) {
	words := []string{"a", "bb", "ccc", "dddd", "eeeee", "the", "zebra", "quixotic"}
	pk := NewPicker(words, rand.New(rand.NewSource(1)))
	p := plain()
	p.RecentWords = 0
	count := map[string]int{}
	const n = 80000
	for _, w := range pk.Freetype(nil, n, len(words), nil, p) {
		count[w]++
	}
	mean := float64(n) / float64(len(words))
	for _, w := range words {
		if d := math.Abs(float64(count[w])-mean) / mean; d > 0.05 {
			t.Errorf("%q: %d draws, mean %.0f (off by %.0f%%)", w, count[w], mean, d*100)
		}
	}
}

func TestDraw_WordsOnly(t *testing.T) {
	words := []string{"one", "two", "three", "four", "five", "six", "seven", "eight"}
	pk := NewPicker(words, rand.New(rand.NewSource(5)))
	for size := 1; size <= 10; size++ {
		for _, w := range pk.Freetype(nil, 200, size, nil, testParams) {
			if !slices.Contains(words, w) {
				t.Fatalf("size %d: %q is not a word of the list", size, w)
			}
		}
	}
}

func TestDraw_NeverTheSameWordTwiceInARow(t *testing.T) {
	pk := NewPicker([]string{"ab", "ba"}, rand.New(rand.NewSource(2)))
	got := pk.Freetype([]string{"ab"}, 50, 2, nil, testParams)
	prev := "ab"
	for _, w := range got {
		if w == prev {
			t.Fatalf("%q twice in a row in %v", w, got)
		}
		prev = w
	}
}

func TestDraw_RecentWordsAreRarer(t *testing.T) {
	words := []string{"one", "two", "three"}
	pk := NewPicker(words, rand.New(rand.NewSource(3)))
	recent := []string{"one", "one", "one", "two"}
	count := map[string]int{}
	for i := 0; i < 3000; i++ {
		count[pk.Freetype(recent, 1, 3, nil, plain())[0]]++
	}
	if count["two"] != 0 {
		t.Fatalf("the last word came again: %v", count)
	}
	if count["one"]*5 > count["three"] {
		t.Fatalf("recent word not rarer: %v", count)
	}
}

func TestDraw_EdgeCases(t *testing.T) {
	if got := NewPicker(nil, rand.New(rand.NewSource(1))).Freetype(nil, 6, 5, nil, testParams); got != nil {
		t.Fatalf("no words: %v", got)
	}
	pk := NewPicker([]string{"aaa"}, rand.New(rand.NewSource(1)))
	if got := pk.Freetype(nil, 0, 5, nil, testParams); got != nil {
		t.Fatalf("n = 0: %v", got)
	}
	if got := pk.Freetype([]string{"aaa"}, 6, 5, nil, testParams); len(got) != 6 {
		t.Fatalf("one word: %v", got)
	}
}

func TestDraw_DoesNotChangeRecent(t *testing.T) {
	recent := make([]string, 2, 10)
	recent[0], recent[1] = "one", "two"
	spare := recent[:3]
	spare[2] = "keep"
	NewPicker([]string{"one", "two", "three"}, rand.New(rand.NewSource(4))).Freetype(recent, 6, 10, nil, testParams)
	if spare[2] != "keep" || recent[0] != "one" || recent[1] != "two" {
		t.Fatalf("recent changed: %v", spare)
	}
}

func TestFreetype_OnlyTheChosenWords(t *testing.T) {
	words := []string{"the", "of", "and", "zebra", "quixotic"}
	pk := NewPicker(words, rand.New(rand.NewSource(7)))
	for _, w := range pk.Freetype(nil, 2000, 3, nil, testParams) {
		if w != "the" && w != "of" && w != "and" {
			t.Fatalf("size 3: %q is not one of the 3 most common words", w)
		}
	}
	seen := map[string]bool{}
	for _, w := range pk.Freetype(nil, 2000, 1000, nil, testParams) {
		seen[w] = true
	}
	if len(seen) != len(words) {
		t.Errorf("size above the list: saw %d words, want all %d", len(seen), len(words))
	}
	if got := pk.Freetype([]string{"of"}, 5, 0, nil, testParams); len(got) != 5 || got[0] != "the" {
		t.Errorf("size 0 means the first word only: %v", got)
	}
	if got := pk.Size(); got != len(words) {
		t.Errorf("Size %d, want %d", got, len(words))
	}
}

func TestDraw_OddsAreTheWeights(t *testing.T) {
	pk := NewPicker([]string{"a", "b", "c", "d"}, rand.New(rand.NewSource(8)))
	p := plain()
	recent := []string{"a", "a", "b"}
	count := map[string]int{}
	const n = 60000
	for i := 0; i < n; i++ {
		count[pk.Freetype(recent, 1, 4, nil, p)[0]]++
	}
	if count["b"] != 0 {
		t.Fatalf("the last word came again: %v", count)
	}
	total := 0.09 + 1 + 1
	for w, weight := range map[string]float64{"a": 0.09, "c": 1, "d": 1} {
		want := weight / total * n
		if math.Abs(float64(count[w])-want) > 4*math.Sqrt(want) {
			t.Errorf("%q drawn %d times, want about %.0f (%v)", w, count[w], want, count)
		}
	}
}

func TestDraw_FullWeighing(t *testing.T) {
	pk := NewPicker([]string{"a", "b"}, rand.New(rand.NewSource(9)))
	p := plain()
	recent := []string{"a", "a", "a", "a", "a", "a", "a", "a", "a", "b"}
	for i := 0; i < 200; i++ {
		if got := pk.Freetype(recent, 1, 2, nil, p)[0]; got != "a" {
			t.Fatalf("only %q is allowed, got %q", "a", got)
		}
	}
	p.RepeatPenalty = 0
	for i := 0; i < 50; i++ {
		if got := pk.Freetype([]string{"a", "b"}, 1, 2, nil, p)[0]; got != "a" && got != "b" {
			t.Fatalf("all weights 0: %q is not a word of the list", got)
		}
	}
}
