// Package freeplay is the word draws outside Learn's letter model
// (Freetype), the weak letter pairs, and the pass / hold / fail rule of a
// level (Judge, Standing).
package freeplay

import (
	"math"
	"math/rand"
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
	PassesUp     int
	FailsDown    int
	History      int

	WeakRounds     int
	WeakMinSamples int
	WeakMissMs     float64
	WeakPairs      int

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

func Judge(acc float64, p Params) Verdict {
	switch {
	case acc >= p.PassAccuracy:
		return Pass
	case acc < p.FailAccuracy:
		return Fail
	}
	return Hold
}

type Round struct {
	Level    int
	Accuracy float64
}

type Place struct {
	Level  int `json:"level"`
	Passes int `json:"passes"`
	Fails  int `json:"fails"`
	Moved  int `json:"moved"`
}

func Standing(rounds []Round, p Params) Place {
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
		switch Judge(r.Accuracy, p) {
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

func (pk *Picker) Freetype(recent []string, n, size int, long bool, p Params) []string {
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
	return pk.drawFrom(recent, n, size, pool, 1, p)
}

func (pk *Picker) Size() int { return len(pk.words) }

const tries = 32

func (pk *Picker) drawFrom(recent []string, n, size int, pool []int, share float64, p Params) []string {
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
		word := pk.words[pk.drawWord(window, last, size, from, p)]
		out = append(out, word)
		window = append(window, word)
		if len(window) > keep {
			window = window[len(window)-keep:]
		}
		last = word
	}
	return out
}

func (pk *Picker) drawWord(window []string, last string, size int, from []int, p Params) int {
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
	weight := func(w string) float64 {
		if w == last && size > 1 {
			return 0
		}
		return math.Pow(p.RepeatPenalty, float64(uses[w]))
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
