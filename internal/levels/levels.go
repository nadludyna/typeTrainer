// Package levels moves a mode's level by its rounds: Reinforce's word level
// judged by speed (Judge), Freetype's word list by consistency (JudgeRhythm),
// both over the last run at one level (Standing); and the weak letter pairs
// Reinforce brings more often (WeakPairs). docs/HOW-THE-MODES-WORK.md,
// sections 0.2 and 0.3.
package levels

type Params struct {
	Words        []int // each level's number of most common words
	LevelStart   int
	PassAccuracy float64
	FailAccuracy float64
	FailSpeed    float64 // a round slower than this share of its goal fails
	ConsPass     float64 // consistency, %, a Freetype round passes at
	ConsFail     float64 // consistency, %, a Freetype round fails under
	PassesUp     int
	FailsDown    int
	History      int

	WeakRounds     int
	WeakMinSamples int
	WeakMissMs     float64
	WeakPairs      int
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
	Cold        bool     // a warm-up round: it can pass, but a fail only holds
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
	top := max(len(p.Words), 1)
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
		verdict := judge(r, p)
		if r.Cold && verdict == Fail {
			verdict = Hold
		}
		switch verdict {
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

// JudgeRhythm judges a Freetype round for its word list, as Judge does a
// Reinforce round for its word level, by consistency instead of speed: it
// passes at ConsPass or more with PassAccuracy or more, fails under ConsFail
// or under FailAccuracy, and holds in between. A round too short to have a
// consistency holds.
func JudgeRhythm(r Round, p Params) Verdict {
	switch {
	case r.Consistency == nil:
		return Hold
	case *r.Consistency >= p.ConsPass && r.Accuracy >= p.PassAccuracy:
		return Pass
	case *r.Consistency < p.ConsFail || r.Accuracy < p.FailAccuracy:
		return Fail
	}
	return Hold
}
