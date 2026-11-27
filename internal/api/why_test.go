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
}
