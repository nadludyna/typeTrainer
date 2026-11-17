// Placement: one short test before the first Learn round sets the speed goal
// and the letters a learner already knows.

package engine

import "math"

type PlacementParams struct {
	GoalAbove     float64 // the goal is the test speed plus this, in wpm
	MinGoal       float64
	MaxGoal       float64
	KnownWPM      float64 // below this speed no letter counts as known
	KnownAccuracy float64 // below this accuracy, in %, no letter counts as known
	KnownPresses  int     // a known letter was pressed at least this often
	KnownSlack    float64 // a known letter is at most this many times slower than the test's average press
}

// Place reads a test: its speed, its accuracy and its presses. A known letter
// had no mistake, enough timed presses, and an average press not much slower
// than the whole test. Known letters keep the order they first appear in.
func Place(samples []Sample, wpm, accuracy float64, p PlacementParams) (goal float64, known []string) {
	goal = math.Min(math.Max(math.Round(wpm)+p.GoalAbove, p.MinGoal), p.MaxGoal)
	if wpm < p.KnownWPM || accuracy < p.KnownAccuracy {
		return goal, nil
	}
	limit := p.KnownSlack * 12000 / wpm
	type seen struct {
		ms     float64
		timed  int
		missed bool
	}
	by := map[string]*seen{}
	var order []string
	for _, s := range samples {
		if s.Char == " " || s.Char == "" {
			continue
		}
		c := by[s.Char]
		if c == nil {
			c = &seen{}
			by[s.Char] = c
			order = append(order, s.Char)
		}
		if s.Errors > 0 {
			c.missed = true
		} else if s.Ms > 0 {
			c.ms += s.Ms
			c.timed++
		}
	}
	for _, ch := range order {
		c := by[ch]
		if !c.missed && c.timed >= p.KnownPresses && c.ms/float64(c.timed) <= limit {
			known = append(known, ch)
		}
	}
	return goal, known
}
