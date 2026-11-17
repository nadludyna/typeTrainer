package engine

import (
	"slices"
	"testing"
)

var pp = PlacementParams{GoalAbove: 5, MinGoal: 15, MaxGoal: 150, KnownWPM: 25, KnownAccuracy: 90, KnownPresses: 2, KnownMisses: 1, KnownSlack: 1.25}

func presses(text string, ms float64) []Sample {
	var out []Sample
	for _, r := range text {
		out = append(out, Sample{Char: string(r), Ms: ms})
	}
	return out
}

func TestPlace_Goal(t *testing.T) {
	for _, c := range []struct{ wpm, want float64 }{{0, 15}, {8.4, 15}, {40.4, 45}, {115, 120}, {300, 150}} {
		if g, _ := Place(nil, c.wpm, 100, pp); g != c.want {
			t.Errorf("%v wpm: goal %v, want %v", c.wpm, g, c.want)
		}
	}
}

func TestPlace_Known(t *testing.T) {
	s := presses("aab ccd", 200)                                                                             // 60 wpm: 200 ms a press, the limit is 250
	s = append(s, Sample{Char: "e", Errors: 1}, Sample{Char: "e", Ms: 100}, Sample{Char: "e", Ms: 100})      // one slip is fine
	s = append(append(s, Sample{Char: "h", Errors: 1}, Sample{Char: "h", Errors: 1}), presses("hh", 100)...) // two are not
	s = append(s, presses("ff", 400)...)                                                                     // too slow
	s = append(s, Sample{Char: "g", Ms: 100}, Sample{Char: "g"})                                             // one timed press only
	_, known := Place(s, 60, 95, pp)
	if want := []string{"a", "c", "e"}; !slices.Equal(known, want) {
		t.Errorf("known %v, want %v", known, want)
	}
	if _, k := Place(s, 20, 95, pp); k != nil {
		t.Errorf("too slow a test: known %v", k)
	}
	if _, k := Place(s, 60, 80, pp); k != nil {
		t.Errorf("too many mistakes: known %v", k)
	}
	if _, k := Place(nil, 60, 100, pp); k != nil {
		t.Errorf("no presses: known %v", k)
	}
}
