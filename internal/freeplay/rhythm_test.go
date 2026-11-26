package freeplay

import (
	"math/rand"
	"strings"
	"testing"
)

// typed gives every letter of text a time around ms (±20 %), and the second
// letter of each pair in slow a stumble of slowMs every other time.
func typed(text string, ms, slowMs float64, slow []string, seed int64) []Key {
	rng := rand.New(rand.NewSource(seed))
	runes := []rune(text)
	seen := map[string]int{}
	out := make([]Key, len(runes))
	for i, r := range runes {
		out[i] = Key{Char: string(r), Ms: ms * (0.8 + 0.4*rng.Float64())}
		if i == 0 {
			continue
		}
		pair := string(runes[i-1:i+1])
		for _, s := range slow {
			if pair == s {
				if seen[pair]%2 == 0 {
					out[i].Ms = slowMs
				}
				seen[pair]++
			}
		}
	}
	return out
}

const pangrams = "the quick brown fox jumps over the lazy dog pack my box with five dozen liquor jugs "

func TestReadRhythm_NoData(t *testing.T) {
	for _, rounds := range [][][]Key{nil, {}, {typed("a b c", 150, 0, nil, 1)}, {typed("the fox", 150, 0, nil, 1)}} {
		got := ReadRhythm(rounds, testParams)
		if got.Median != 0 || got.Threshold != 0 || len(got.Pairs) != 0 || got.Pairs == nil {
			t.Fatalf("too few presses to read: %+v", got)
		}
	}
}

func TestReadRhythm_EvenTypistStumblesNowhere(t *testing.T) {
	keys := typed(strings.Repeat(pangrams, 10), 150, 0, nil, 1)
	for i := range keys {
		keys[i].Ms = 150
	}
	got := ReadRhythm([][]Key{keys}, testParams)
	if got.Median != 150 || got.Spread != 0 || got.Threshold != 225 || got.Rate != 0 || len(got.Pairs) != 0 {
		t.Fatalf("perfectly even: %+v; want median 150, threshold 1.5 × 150, no stumbles", got)
	}
	got = ReadRhythm([][]Key{typed(strings.Repeat(pangrams, 10), 150, 0, nil, 2)}, testParams)
	if got.Rate != 0 || len(got.Pairs) != 0 {
		t.Fatalf("even within ±20 %%: %+v", got)
	}
}

func TestReadRhythm_OneLongPause(t *testing.T) {
	keys := typed(strings.Repeat(pangrams, 10), 150, 0, nil, 3)
	before := ReadRhythm([][]Key{keys}, testParams)
	keys[5].Ms = 1500 // "qu": slow once
	keys[8].Ms = 3000 // "ck": a rest, not read
	got := ReadRhythm([][]Key{keys}, testParams)
	if got.Presses != before.Presses-1 {
		t.Fatalf("a gap over 2 s is a rest: %d presses, want %d", got.Presses, before.Presses-1)
	}
	if got.Median < before.Median*0.97 || got.Median > before.Median*1.03 || got.Rate == 0 || len(got.Pairs) != 0 {
		t.Fatalf("one long pause: %+v; the median stays, one stumble, no pair", got)
	}
}

func TestReadRhythm_FindsTheStumblePairs(t *testing.T) {
	text := strings.Repeat(pangrams, 12)
	rounds := [][]Key{typed(text, 150, 600, []string{"qu", "zy"}, 4), typed(text, 150, 600, []string{"qu", "zy"}, 5)}
	got := ReadRhythm(rounds, testParams)
	if len(got.Pairs) != 2 {
		t.Fatalf("pairs %+v, want qu and zy", got.Pairs)
	}
	for _, s := range got.Pairs {
		if (s.Pair != "qu" && s.Pair != "zy") || s.Rate < testParams.StumbleRatio*got.Rate {
			t.Fatalf("pair %+v at the typist's rate %.3f", s, got.Rate)
		}
	}
	if got.Threshold <= got.Median || got.Threshold >= 600 {
		t.Fatalf("threshold %.0f between the median %.0f and a 600 ms stumble", got.Threshold, got.Median)
	}
}

func TestReadRhythm_ARarePairIsNoise(t *testing.T) {
	text := strings.Repeat(pangrams, 10) + "quiz quiz quiz "
	got := ReadRhythm([][]Key{typed(text, 150, 600, []string{"iz"}, 6)}, testParams)
	for _, s := range got.Pairs {
		if s.Pair == "iz" {
			t.Fatalf("iz typed 3 times is noise: %+v", got.Pairs)
		}
	}
}

func TestReadRhythm_WordStartsAndMistakesAreNotRead(t *testing.T) {
	keys := typed(strings.Repeat(pangrams, 10), 150, 0, nil, 7)
	for i, k := range keys {
		if i > 0 && keys[i-1].Char == " " {
			keys[i].Ms = 900 // reading the next word
		}
		if k.Char == "x" {
			keys[i].Errors = 1
			keys[i].Ms = 900
		}
	}
	if got := ReadRhythm([][]Key{keys}, testParams); got.Rate != 0 {
		t.Fatalf("a word's first letter and a wrong press are not stumbles: %+v", got)
	}
}

func TestJudgeRhythm(t *testing.T) {
	c := func(v float64) *float64 { return &v }
	for _, x := range []struct {
		r    Round
		want Verdict
	}{
		{Round{Accuracy: 98, Consistency: c(85)}, Pass},
		{Round{Accuracy: 96, Consistency: c(80)}, Pass},
		{Round{Accuracy: 95, Consistency: c(90)}, Hold},
		{Round{Accuracy: 98, Consistency: c(75)}, Hold},
		{Round{Accuracy: 98, Consistency: c(69)}, Fail},
		{Round{Accuracy: 90, Consistency: c(95)}, Fail},
		{Round{Accuracy: 100}, Hold},
		{Round{Accuracy: 100, WPM: 10, Goal: 100, Consistency: c(85)}, Pass},
	} {
		if got := JudgeRhythm(x.r, testParams); got != x.want {
			t.Errorf("%+v: %s, want %s", x.r, got, x.want)
		}
	}
	c80 := c(85)
	run := []Round{{Level: 2, Accuracy: 98, Consistency: c80}, {Level: 2, Accuracy: 98, Consistency: c80}, {Level: 2, Accuracy: 98, Consistency: c80}}
	if got := Standing(run, JudgeRhythm, testParams); got.Level != 3 || got.Moved != 1 {
		t.Fatalf("three even rounds move the list up: %+v", got)
	}
}

// A word with a stumble pair comes 1 + StumbleBoost times as often as it
// would; with no pairs the draw is as before.
func TestFreetype_StumblePairsComeMoreOften(t *testing.T) {
	words := []string{"quiz", "lazy", "fox", "dog", "the", "box", "pack", "jugs", "five", "over"}
	share := func(pairs []string) float64 {
		pk := NewPicker(words, rand.New(rand.NewSource(8)))
		p := testParams
		p.RepeatPenalty = 1
		with, all := 0, 0
		for range 400 {
			for _, w := range pk.Freetype(nil, 6, len(words), false, pairs, p) {
				all++
				if w == "quiz" || w == "lazy" {
					with++
				}
			}
		}
		return float64(with) / float64(all)
	}
	plain, boosted := share(nil), share([]string{"qu", "zy"})
	// 2 of 10 words: 20 % plain; weighted 3 : 1, 2·3 / (2·3 + 8) ≈ 43 %.
	if plain < 0.17 || plain > 0.23 || boosted < 0.39 || boosted > 0.47 {
		t.Fatalf("words with a stumble pair: %.2f plain, %.2f with the pairs; want about 0.20 and 0.43", plain, boosted)
	}
}
