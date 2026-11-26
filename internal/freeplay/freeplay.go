// Package freeplay is the word draws outside Learn's letter model
// (Freetype), the weak letter pairs, and the pass / hold / fail rule that
// moves Reinforce's word level (Judge, Standing; docs/HOW-THE-MODES-WORK.md,
// section 0.2).
package freeplay

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

	Levels       []int
	LevelStart   int
	PassAccuracy float64
	FailAccuracy float64
	FailSpeed    float64 // a round slower than this share of its goal fails
	PassesUp     int
	FailsDown    int
	History      int

	WeakRounds     int
	WeakMinSamples int
	WeakMissMs     float64
	WeakPairs      int

	// Freetype's rhythm (rhythm.go) and its word list's move.
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
	ConsPass          float64 // consistency, %, a Freetype round passes at
	ConsFail          float64 // consistency, %, a Freetype round fails under

	FreetypeLists []int
	LongMin       int
	TextMarks     string
	SentenceTries int
}

type Verdict string

const (
	Pass Verdict = "pass"
	Hold Verdict = "hold"
	Fail Verdict = "fail"
)

// Judge: a round passes at its goal speed or faster with PassAccuracy or
// more, fails below FailAccuracy or below FailSpeed of its goal, and holds in
// between. A round with no goal is judged by its accuracy alone.
func Judge(r Round, p Params) Verdict {
	fast := r.Goal <= 0 || r.WPM >= r.Goal
	slow := r.Goal > 0 && r.WPM < p.FailSpeed*r.Goal
	switch {
	case r.Accuracy >= p.PassAccuracy && fast:
		return Pass
	case r.Accuracy < p.FailAccuracy || slow:
		return Fail
	}
	return Hold
}

type Round struct {
	Level       int
	Accuracy    float64
	WPM         float64
	Goal        float64  // the speed goal the round was typed at, wpm
	Consistency *float64 // Freetype's; none in a round under ConsistencyMinSeconds
}

type Place struct {
	Level  int `json:"level"`
	Passes int `json:"passes"`
	Fails  int `json:"fails"`
	Moved  int `json:"moved"`
}

// Standing is where the last run of rounds at one level leaves it, each
// round judged by judge (Judge for Reinforce, JudgeRhythm for Freetype).
func Standing(rounds []Round, judge func(Round, Params) Verdict, p Params) Place {
	top := max(len(p.Levels), 1)
	if len(rounds) == 0 {
		return Place{Level: clampLevel(p.LevelStart, top)}
	}
	level := clampLevel(rounds[len(rounds)-1].Level, top)
	first := len(rounds) - 1
	for first > 0 && clampLevel(rounds[first-1].Level, top) == level {
		first--
	}
	var at Place
	for _, r := range rounds[first:] {
		at.Moved = 0
		switch judge(r, p) {
		case Pass:
			at.Passes, at.Fails = at.Passes+1, 0
			if at.Passes >= p.PassesUp {
				at.Passes = 0
				if level < top {
					at.Moved = 1
				}
			}
		case Fail:
			at.Passes, at.Fails = 0, at.Fails+1
			if at.Fails >= p.FailsDown {
				at.Fails = 0
				if level > 1 {
					at.Moved = -1
				}
			}
		default:
			at.Fails = 0
		}
	}
	at.Level = level + at.Moved
	return at
}

func clampLevel(level, top int) int { return max(1, min(top, level)) }

type Picker struct {
	words []string
	rng   *rand.Rand
}

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
