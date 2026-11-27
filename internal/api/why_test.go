package api

import (
	"slices"
	"testing"
)

func TestWhy(t *testing.T) {
	w := weakSpots{back: []string{"z"}, pairs: []string{"ou"}, stumbles: []string{"qu"}, finger: "lp", letters: "aqz", share: 0.3}
	for word, want := range map[string]string{
		"lazy":   "back:z", // the rarest reason wins over the finger
		"house":  "pair:ou",
		"quit":   "stumble:qu",
		"salad":  "finger:lp", // 2 of 5 letters
		"father": "focus:f",   // 1 of 6 letters is not the finger's word
		"then":   "",
		"":       "",
	} {
		if got := w.why(word, "f"); got != want {
			t.Errorf("%q: %q, want %q", word, got, want)
		}
	}
	if got := (weakSpots{}).whys([]string{"lazy", "house"}, ""); !slices.Equal(got, []string{"", ""}) {
		t.Errorf("no weak spots, no focus: %v", got)
	}
	// In a sprint an owned word came for being owned, before any weak spot.
	w.owned = map[string]bool{"house": true}
	if got := w.whys([]string{"house", "quit"}, ""); !slices.Equal(got, []string{"owned:", "stumble:qu"}) {
		t.Errorf("a sprint: %v", got)
	}
}
