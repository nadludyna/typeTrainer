// The letter model Learn and Reinforce share: a round's presses update each
// letter; Learn opens the next letter, and the speed goal steps up once
// every letter is mastered at it.

package trainer

import (
	"math"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/storage"
)

// Learned is the progress after a round's presses.
type Learned struct {
	Progress  storage.Progress
	NewLetter string  // the letter the round opened, "" for none
	NewGoal   float64 // the speed goal of the step the round reached, 0 for none
	Lap       int     // the step the round was typed at
}

// Params is the model's numbers at a speed goal step.
func (t *Trainer) Params(lap int) engine.Params {
	return t.Cfg.Engine.ForLap(lap, t.Cfg.Laps)
}

func (t *Trainer) LastLap() int { return t.Cfg.Engine.LastLap(t.Cfg.Laps) }

// Learn applies a round's presses to the typist's letters. A Reinforce round
// trains every letter as a Learn round would, but only a Learn round opens
// the next letter; the speed goal steps on every letter, open in Learn or
// not. The caller holds Write.
func (t *Trainer) Learn(user string, samples []engine.Sample, reinforce bool) (Learned, error) {
	p, err := t.Store.Progress(user)
	if err != nil {
		return Learned{}, err
	}
	gen := t.Words.Gen
	all := len(gen.Order())
	p.Unlocked = gen.Clamp(p.Unlocked)
	p.Lap = min(max(p.Lap, 0), t.LastLap())
	out := Learned{Progress: p, Lap: p.Lap}
	if len(samples) == 0 {
		return out, nil
	}
	params := t.Params(p.Lap)
	before := p.Unlocked
	p.Letters = engine.Apply(p.Letters, samples, params.ForStage(before, all))
	if !reinforce {
		p.Unlocked = gen.NextUnlocked(p.Letters, before, params.ForStage(before, all))
	}
	if p.Unlocked > before {
		out.NewLetter = gen.Order()[p.Unlocked-1]
	}
	open := p.Unlocked
	if reinforce {
		open = all
	}
	if p.Lap < t.LastLap() && engine.LapDone(p.Letters, gen.Order(), open, params.ForStage(open, all)) {
		p.Letters, p.Lap = engine.StartLap(p.Letters), p.Lap+1
		out.NewGoal = t.Params(p.Lap).TargetWPM
	}
	out.Progress = p
	return out, nil
}

// Letter is one letter as the page shows it.
type Letter struct {
	Char     string  `json:"char"`
	Count    int     `json:"count"`
	Level    float64 `json:"level"` // 0..100: Learn's accuracy, or with every letter open the mix of accuracy and speed
	AvgMs    float64 `json:"avgMs"`
	Mastered bool    `json:"mastered"`
	Open     bool    `json:"open"`
}

// Letters is every letter in the order they open.
func (t *Trainer) Letters(p storage.Progress, open int) []Letter {
	gen := t.Words.Gen
	params := t.Params(p.Lap).ForStage(open, len(gen.Order()))
	out := make([]Letter, 0, len(gen.Order()))
	for i, c := range gen.Order() {
		st := p.Letters[c]
		out = append(out, Letter{Char: c, Count: st.Count, AvgMs: math.Round(st.TimeEMA),
			Level: shownLevel(engine.Level(st, params), params), Mastered: engine.Mastered(st, params), Open: i < open})
	}
	return out
}

// shownLevel rounds the level to a tenth; an accuracy rounds down, so a
// letter short of mastered never shows the mastered number.
func shownLevel(level float64, p engine.Params) float64 {
	if p.AccWindow > 0 {
		return math.Floor(level*10) / 10
	}
	return math.Round(level*10) / 10
}
