// Package freetype is Freetype's words: draws from the N most common words
// or long words, real sentences, and the rhythm that weighs the draw
// (docs/HOW-THE-MODES-WORK.md, section 0.3). How its word list moves is
// package levels.
package freetype

import (
	"math"
	"math/rand"
	"strings"
	"unicode/utf8"
)

type Params struct {
	PortionWords  int
	MaxStartWords int
	RecentWords   int
	RepeatPenalty float64
	OwnedPenalty  float64 // a word the typist owns weighs this much of its weight

	Lists         []int // the word lists: the N most common words
	LongMin       int
	TextMarks     string
	SentenceTries int

	// The rhythm (rhythm.go).
	RhythmRounds      int     // the last rounds it is read from
	RhythmMaxMs       float64 // a longer gap between two letters is a rest, not a stumble
	RhythmMinPresses  int     // with fewer presses inside words no rhythm is read
	StumbleSpreads    float64 // a stumble is slower than the median plus this many spreads
	StumbleLeast      float64 // and slower than this many medians
	StumblePrior      float64 // presses imagined at the typist's own stumble rate in each pair
	StumbleMinPresses int     // a pair typed fewer times is noise
	StumbleMinCount   int     // stumbles a pair needs to count
	StumbleRatio      float64 // a stumble pair stumbles this many times the typist's own rate
	StumblePairs      int     // at most this many stumble pairs
	StumbleBoost      float64 // a word with a stumble pair weighs 1 + this
}

type Picker struct {
	words []string
	rng   *rand.Rand
	owned map[string]bool
}

// WithOwned is this picker with the words the typist owns: they come less
// often (OwnedPenalty). It shares the words and the random source.
func (pk *Picker) WithOwned(owned map[string]bool) *Picker {
	if len(owned) == 0 {
		return pk
	}
	out := *pk
	out.owned = owned
	return &out
}

// Top is the n most common words.
func (pk *Picker) Top(n int) []string { return pk.words[:min(max(n, 0), len(pk.words))] }

func NewPicker(words []string, rng *rand.Rand) *Picker {
	return &Picker{words: words, rng: rng}
}

// Freetype draws n words from the size most common (long: of LongMin
// letters or more), a word with one of the stumble pairs 1 + StumbleBoost
// times as likely as one without.
func (pk *Picker) Freetype(recent []string, n, size int, long bool, pairs []string, p Params) []string {
	size = min(max(size, 1), len(pk.words))
	var pool []int
	if long {
		for i, w := range pk.words[:size] {
			if utf8.RuneCountInString(w) >= p.LongMin {
				pool = append(pool, i)
			}
		}
	}
	if len(pool) == 0 {
		pool = nil
	}
	return pk.drawFrom(recent, n, size, pool, 1, pairs, p)
}

func (pk *Picker) Size() int { return len(pk.words) }

const tries = 32

func (pk *Picker) drawFrom(recent []string, n, size int, pool []int, share float64, pairs []string, p Params) []string {
	if len(pk.words) == 0 || n <= 0 {
		return nil
	}
	keep := max(p.RecentWords, 0)
	window := make([]string, 0, keep+n)
	window = append(window, recent[max(0, len(recent)-keep):]...)
	last := ""
	if len(recent) > 0 {
		last = recent[len(recent)-1]
	}
	out := make([]string, 0, n)
	for len(out) < n {
		from := []int(nil)
		if pool != nil && pk.rng.Float64() < share {
			from = pool
		}
		word := pk.words[pk.drawWord(window, last, size, from, pairs, p)]
		out = append(out, word)
		window = append(window, word)
		if len(window) > keep {
			window = window[len(window)-keep:]
		}
		last = word
	}
	return out
}

func (pk *Picker) drawWord(window []string, last string, size int, from []int, pairs []string, p Params) int {
	count := size
	at := func(k int) int { return k }
	if from != nil {
		count = len(from)
		at = func(k int) int { return from[k] }
	}
	uses := make(map[string]int, len(window))
	for _, w := range window {
		uses[w]++
	}
	// Weights stay at most 1 for the draw by rejection: a word without a
	// stumble pair is scaled down instead of one with it up.
	plain := 1.0
	if len(pairs) > 0 {
		plain = 1 / (1 + p.StumbleBoost)
	}
	weight := func(w string) float64 {
		if w == last && size > 1 {
			return 0
		}
		x := math.Pow(p.RepeatPenalty, float64(uses[w]))
		if !hasPair(w, pairs) {
			x *= plain
		}
		if pk.owned[w] {
			x *= p.OwnedPenalty
		}
		return x
	}
	for range tries {
		i := at(pk.rng.Intn(count))
		if pk.rng.Float64() < weight(pk.words[i]) {
			return i
		}
	}
	total := 0.0
	for k := range count {
		total += weight(pk.words[at(k)])
	}
	if total <= 0 {
		return at(pk.rng.Intn(count))
	}
	point, found := pk.rng.Float64()*total, at(count-1)
	for k := range count {
		i := at(k)
		x := weight(pk.words[i])
		if x > 0 {
			found = i
		}
		if point < x {
			return i
		}
		point -= x
	}
	return found
}

func hasPair(word string, pairs []string) bool {
	for _, pair := range pairs {
		if strings.Contains(word, pair) {
			return true
		}
	}
	return false
}
