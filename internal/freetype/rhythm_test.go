package freetype

import (
	"math/rand"
	"slices"
	"strings"
	"testing"

	"typingtrainer/data"
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
		pair := string(runes[i-1 : i+1])
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
			for _, w := range pk.Freetype(nil, 6, len(words), pairs, p) {
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

// The whole loop on a simulated typist who stumbles on "ou" every other
// time: 30 rounds of 30 words from the 1 000 most common, the stumble pairs
// read again before every round from the last 20. With the loop the typist
// meets their stumble pair more often than with words drawn evenly.
func TestRhythmLoop_SimulatedTypistMeetsTheirStumblePairMoreOften(t *testing.T) {
	words := strings.Fields(string(data.Words))[:1000]
	run := func(adaptive bool) (share float64, found []string) {
		pk := NewPicker(words, rand.New(rand.NewSource(9)))
		rng := rand.New(rand.NewSource(10))
		var rounds [][]Key
		var pairs []string
		met, all := 0, 0
		for r := range 30 {
			text := strings.Join(pk.Freetype(nil, 30, len(words), pairs, testParams), " ") + " "
			runes := []rune(text)
			keys := make([]Key, len(runes))
			for i, c := range runes {
				keys[i] = Key{Char: string(c), Ms: 150 * (0.8 + 0.4*rng.Float64())}
				if i > 0 && string(runes[i-1:i+1]) == "ou" {
					if r >= 10 {
						met++
					}
					if rng.Float64() < 0.5 {
						keys[i].Ms = 600
					}
				}
				if r >= 10 && i > 0 && runes[i] != ' ' && runes[i-1] != ' ' {
					all++
				}
			}
			rounds = append(rounds, keys)
			rh := ReadRhythm(rounds[max(0, len(rounds)-testParams.RhythmRounds):], testParams)
			pairs = nil
			for _, s := range rh.Pairs {
				if adaptive {
					pairs = append(pairs, s.Pair)
				}
				found = append(found, s.Pair)
			}
		}
		return float64(met) / float64(all), found
	}
	plain, _ := run(false)
	loop, found := run(true)
	t.Logf(`presses on "ou" in rounds 11-30: %.1f %% with words drawn evenly, %.1f %% with the loop (%.1f×)`, 100*plain, 100*loop, loop/plain)
	if !slices.Contains(found, "ou") {
		t.Fatalf("the loop never found ou: %v", found)
	}
	if loop < 1.5*plain {
		t.Fatalf("ou: %.3f with the loop, %.3f without; want at least 1.5×", loop, plain)
	}
}

func TestFreetype_OwnedWordsComeLessOften(t *testing.T) {
	words := []string{"quiz", "lazy", "fox", "dog", "the", "box", "pack", "jugs", "five", "over"}
	share := func(owned map[string]bool) float64 {
		pk := NewPicker(words, rand.New(rand.NewSource(8))).WithOwned(owned)
		p := testParams
		p.RepeatPenalty, p.OwnedPenalty = 1, 0.3
		with, all := 0, 0
		for range 400 {
			for _, w := range pk.Freetype(nil, 6, len(words), nil, p) {
				all++
				if w == "quiz" || w == "lazy" {
					with++
				}
			}
		}
		return float64(with) / float64(all)
	}
	// 2 of 10 words: 20 % plain; at 0.3 of their weight, 0.6 / 8.6 ≈ 7 %.
	plain, owned := share(nil), share(map[string]bool{"quiz": true, "lazy": true})
	if plain < 0.17 || plain > 0.23 || owned < 0.05 || owned > 0.10 {
		t.Fatalf("owned words: %.2f plain, %.2f owned; want about 0.20 and 0.07", plain, owned)
	}
	// A sprint: owned 51 times a word not owned, 102 / 110 ≈ 93 %; with
	// nothing owned it is the plain draw.
	sprint := func(owned map[string]bool) float64 {
		pk := NewPicker(words, rand.New(rand.NewSource(8))).WithOwned(owned)
		p := testParams
		p.RepeatPenalty, p.OwnedPenalty, p.OwnedBoost = 1, 0.3, 50
		with, all := 0, 0
		for range 400 {
			for _, w := range pk.Freetype(nil, 6, len(words), nil, p) {
				all++
				if w == "quiz" || w == "lazy" {
					with++
				}
			}
		}
		return float64(with) / float64(all)
	}
	if s := sprint(map[string]bool{"quiz": true, "lazy": true}); s < 0.85 {
		t.Fatalf("a sprint: %.2f owned words, want about 0.93", s)
	}
	if s := sprint(nil); s < 0.17 || s > 0.23 {
		t.Fatalf("a sprint with nothing owned: %.2f, want the plain 0.20", s)
	}
	pk := NewPicker(words, rand.New(rand.NewSource(1)))
	if pk.WithOwned(nil) != pk || len(pk.Top(3)) != 3 || len(pk.Top(99)) != 10 || len(pk.Top(-1)) != 0 {
		t.Fatal("nothing owned is the same picker; Top is the first n words")
	}
}
