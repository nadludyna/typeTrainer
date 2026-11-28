// Package engine is the adaptive learning model.
package engine

import (
	"math"
	"strings"
)

type Params struct {
	Alpha         float64
	ErrAlpha      float64
	MaxIntervalMs float64

	TargetWPM    float64
	SlowMs       float64
	MaxErrorRate float64
	ErrorWeight  float64
	SpeedWeight  float64

	PriorDifficulty float64
	PriorStrength   float64

	Temperature float64

	MasteryMinSamples int
	MasteryLevel      float64

	UnmasteredBonus float64

	StartLetters   int
	MinWords       int
	RepeatPenalty  float64
	MinWordLetters int
	MaxRepeats     int
	ExerciseWords  int

	PortionWords  int
	AheadPortions int
	RecentWords   int
	RetryWords    int
	RoundPortions int

	RankStart   int
	RankStep    float64
	RankPenalty float64

	PairBoost float64 // a word's weight grows by this for each weak pair in it (Reinforce)
	// A word with at least FingerShare of its letters on the weak finger weighs 1 + FingerBoost.
	FingerShare  float64
	FingerBoost  float64
	OwnedPenalty float64 // a word the typist owns weighs this much of its weight (Reinforce)
	OwnedBoost   float64 // in a sprint an owned word weighs 1 + this instead; 0 outside a sprint

	// Accuracy rules of Learn; off while AccWindow is 0.
	AccWindow int     // presses of a letter the accuracy looks at
	AccPrior  float64 // imaginary misses added to them
	AccMaster float64 // accuracy of a mastered letter
	AccSlip   float64 // an older letter under this takes the focus
}

func (p Params) accuracyRules() bool { return p.AccWindow > 0 }

func (p Params) TargetMs() float64 { return 12000 / p.TargetWPM }

type Sample struct {
	Char   string  `json:"char"`
	Errors int     `json:"errors"`
	Ms     float64 `json:"ms"`
	Typed  string  `json:"typed,omitempty"`
}

type KeyStat struct {
	Count      int     `json:"count"`
	Mistakes   int     `json:"mistakes"`
	ErrEMA     float64 `json:"errEma"`
	TimeEMA    float64 `json:"timeEma"`
	TimedCount int     `json:"timedCount"`
	Held       bool    `json:"held,omitempty"`
	Hits       string  `json:"hits,omitempty"`    // last AccWindow presses, 1 = correct
	Slipped    bool    `json:"slipped,omitempty"` // under AccSlip, back at AccMaster clears it
}

func emaStep(n int, alpha float64) float64 {
	return math.Max(alpha, 1/float64(n))
}

func Update(s KeyStat, smp Sample, p Params) KeyStat {
	s.Count++
	miss := 0.0
	if smp.Errors > 0 {
		miss = 1
		s.Mistakes++
	}
	s.ErrEMA += emaStep(s.Count, p.errAlpha()) * (miss - s.ErrEMA)

	if smp.Errors == 0 && smp.Ms > 0 && smp.Ms <= p.MaxIntervalMs {
		s.TimedCount++
		s.TimeEMA += emaStep(s.TimedCount, p.Alpha) * (smp.Ms - s.TimeEMA)
	}
	if p.accuracyRules() {
		if s.Hits == "" && s.Count > 1 {
			before := s
			before.Count--
			s.Hits = seedHits(before, p)
		}
		hit := "1"
		if smp.Errors > 0 {
			hit = "0"
		}
		s.Hits += hit
		if len(s.Hits) > p.AccWindow {
			s.Hits = s.Hits[len(s.Hits)-p.AccWindow:]
		}
	}
	return s
}

// Accuracy: correct presses over the last AccWindow presses plus AccPrior
// imaginary misses (Laplace's rule of succession); 0 for an unseen letter.
func Accuracy(s KeyStat, p Params) float64 {
	hits := s.Hits
	if hits == "" && s.Count > 0 {
		hits = seedHits(s, p)
	}
	n := float64(len(hits))
	if n+p.AccPrior == 0 {
		return 0
	}
	return float64(strings.Count(hits, "1")) / (n + p.AccPrior)
}

// seedHits stands in for the presses of a letter typed before the accuracy
// rules: as many of the last presses as it has, missed at its error rate.
func seedHits(s KeyStat, p Params) string {
	n := min(s.Count, p.AccWindow)
	miss := int(math.Round(float64(n) * clamp01(s.ErrEMA)))
	return strings.Repeat("0", miss) + strings.Repeat("1", n-miss)
}

// ForStage turns the accuracy rules off once every letter is open: they are
// Learn's; Reinforce adds speed.
func (p Params) ForStage(unlocked, all int) Params {
	if unlocked >= all {
		p.AccWindow = 0
	}
	return p
}

func (p Params) errAlpha() float64 {
	if p.ErrAlpha > 0 {
		return p.ErrAlpha
	}
	return p.Alpha
}

// Apply learns from a batch of presses. A batch is whole words (the browser
// sends a portion with its space, or the rest of a round), so the first press
// of a batch and every press right after a space begin a word: their time
// includes reading the word, not only the finger, so they count for accuracy
// but not for speed.
func Apply(stats map[string]KeyStat, samples []Sample, p Params) map[string]KeyStat {
	out := make(map[string]KeyStat, len(stats))
	for k, v := range stats {
		out[k] = v
	}
	touched := map[string]bool{}
	for i, smp := range samples {
		if i == 0 || samples[i-1].Char == " " {
			smp.Ms = 0
		}
		out[smp.Char] = Update(out[smp.Char], smp, p)
		touched[smp.Char] = true
	}
	for c := range touched {
		if p.accuracyRules() {
			s := out[c]
			if a := Accuracy(s, p); a < p.AccSlip {
				s.Slipped = true
			} else if a >= p.AccMaster {
				s.Slipped = false
			}
			out[c] = s
			continue
		}
		if s := out[c]; !s.Held && MeetsGoal(s, p) {
			s.Held = true
			out[c] = s
		}
	}
	return out
}

func clamp01(x float64) float64 { return math.Max(0, math.Min(1, x)) }

func RawDifficulty(s KeyStat, p Params) float64 {
	errorPart := clamp01(s.ErrEMA / p.MaxErrorRate)
	speedPart := 1.0
	if s.TimedCount > 0 {
		speedPart = clamp01((s.TimeEMA - p.TargetMs()) / (p.SlowMs - p.TargetMs()))
	}
	return p.ErrorWeight*errorPart + p.SpeedWeight*speedPart
}

func Difficulty(s KeyStat, p Params) float64 {
	if p.accuracyRules() {
		return 1 - Accuracy(s, p)
	}
	n := float64(s.Count)
	k := p.PriorStrength
	if n+k == 0 {
		return p.PriorDifficulty
	}
	return (n*RawDifficulty(s, p) + k*p.PriorDifficulty) / (n + k)
}

// Level is a letter's progress from 0 to 100: the same number as the
// difficulty, turned around (100 = no mistakes at the goal speed or faster).
func Level(s KeyStat, p Params) float64 {
	return 100 * (1 - Difficulty(s, p))
}

// MeetsGoal: enough presses, one of them timed, and the level high enough.
// Apply checks it once per batch of presses (a round); once met, the letter
// stays mastered (KeyStat.Held).
func MeetsGoal(s KeyStat, p Params) bool {
	if p.accuracyRules() {
		return Accuracy(s, p) >= p.AccMaster
	}
	return s.Count >= p.MasteryMinSamples && s.TimedCount > 0 && Level(s, p) >= p.MasteryLevel
}

func Mastered(s KeyStat, p Params) bool {
	if p.accuracyRules() {
		return MeetsGoal(s, p)
	}
	return s.Held || MeetsGoal(s, p)
}

func SoftmaxWeight(difficulty float64, p Params) float64 {
	return math.Exp(difficulty / p.Temperature)
}
