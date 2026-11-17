// Package engine is the adaptive learning model.
package engine

import "math"

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

	HomeBoost float64

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
}

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
	return s
}

func (p Params) errAlpha() float64 {
	if p.ErrAlpha > 0 {
		return p.ErrAlpha
	}
	return p.Alpha
}

func Apply(stats map[string]KeyStat, samples []Sample, p Params) map[string]KeyStat {
	out := make(map[string]KeyStat, len(stats))
	for k, v := range stats {
		out[k] = v
	}
	touched := map[string]bool{}
	for _, smp := range samples {
		out[smp.Char] = Update(out[smp.Char], smp, p)
		touched[smp.Char] = true
	}
	for c := range touched {
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
	return s.Count >= p.MasteryMinSamples && s.TimedCount > 0 && Level(s, p) >= p.MasteryLevel
}

func Mastered(s KeyStat, p Params) bool {
	return s.Held || MeetsGoal(s, p)
}

func SoftmaxWeight(difficulty float64, p Params) float64 {
	return math.Exp(difficulty / p.Temperature)
}
