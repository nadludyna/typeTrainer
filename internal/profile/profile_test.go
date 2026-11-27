package profile

import (
	"slices"
	"strings"
	"testing"
)

var testParams = Params{Rounds: 20, MinPresses: 50, WeakRatio: 1.5, WeakMargin: 0.02, Prior: 3, MinMistakes: 10, NeighbourDist: 1.1}

// The letter block of QWERTY, each row shifted as on a keyboard.
var (
	order  = []string{"lp", "lr", "lm", "li", "ri", "rm", "rr", "rp"}
	places = func() map[string]Place {
		out := map[string]Place{}
		fingers := []string{"lp", "lr", "lm", "li", "li", "ri", "ri", "rm", "rr", "rp"}
		for row, keys := range []string{"qwertyuiop", "asdfghjkl", "zxcvbnm"} {
			for col, c := range keys {
				out[string(c)] = Place{Finger: fingers[col], Row: row, X: float64(col) + []float64{0, 0.25, 0.75}[row]}
			}
		}
		return out
	}()
)

const text = "the quick brown fox jumps over the lazy dog pack my box with five dozen liquor jugs "

// typed is text typed n times, clean at 150 ms; wrong gives the key pressed
// instead of a character, every time it comes.
func typed(n int, wrong map[rune]string) []Key {
	var out []Key
	for _, c := range strings.Repeat(text, n) {
		k := Key{Char: string(c), Ms: 150}
		if w, ok := wrong[c]; ok {
			k.Errors, k.Typed = 1, w
		}
		out = append(out, k)
	}
	return out
}

func TestRead_NoData(t *testing.T) {
	for _, rounds := range [][][]Key{nil, {}, {{{Char: " "}, {Char: "."}}}} {
		got := Read(rounds, places, order, testParams)
		if got.Presses != 0 || got.Weak != "" || len(got.Fingers) != 0 || len(got.Kinds) != 0 || got.Letters == nil || got.Kinds == nil {
			t.Fatalf("nothing to read: %+v", got)
		}
	}
}

func TestRead_CleanTypistHasNoWeakFinger(t *testing.T) {
	got := Read([][]Key{typed(10, nil)}, places, order, testParams)
	if got.Rate != 0 || got.Weak != "" || len(got.Kinds) != 0 || len(got.Fingers) != 8 {
		t.Fatalf("clean: %+v", got)
	}
	for _, f := range got.Fingers {
		if f.Weak || f.Ms != 150 || f.Presses == 0 {
			t.Fatalf("finger %+v", f)
		}
	}
}

func TestRead_EvenMistakesNameNoFinger(t *testing.T) {
	// every tenth press wrong, whatever the finger
	keys := typed(10, nil)
	for i := range keys {
		if i%10 == 0 {
			keys[i].Errors = 1
		}
	}
	if got := Read([][]Key{keys}, places, order, testParams); got.Weak != "" {
		t.Fatalf("mistakes spread evenly: weak finger %q, %+v", got.Weak, got.Fingers)
	}
}

func TestRead_TheWeakFinger(t *testing.T) {
	// the left pinky misses every a, q and z; the rest is clean
	got := Read([][]Key{typed(10, map[rune]string{'a': "s", 'q': "w", 'z': "x"})}, places, order, testParams)
	if got.Weak != "lp" || !slices.Equal(got.Letters, []string{"a", "q", "z"}) {
		t.Fatalf("weak %q, letters %v, fingers %+v", got.Weak, got.Letters, got.Fingers)
	}
	for _, f := range got.Fingers {
		if f.Weak != (f.Finger == "lp") {
			t.Fatalf("finger %+v", f)
		}
	}
}

func TestRead_AFingerWithFewPressesIsNotJudged(t *testing.T) {
	keys := []Key{{Char: "q", Errors: 1, Typed: "w"}, {Char: "q", Errors: 1, Typed: "w"}}
	for range 200 {
		keys = append(keys, Key{Char: "j", Ms: 150})
	}
	if got := Read([][]Key{keys}, places, order, testParams); got.Weak != "" {
		t.Fatalf("two presses of a finger say nothing: %+v", got.Fingers)
	}
}

func TestRead_KindsOfMistakes(t *testing.T) {
	var keys []Key
	add := func(n int, word string, at int, typed string) {
		for range n {
			for i, c := range word {
				k := Key{Char: string(c), Ms: 150}
				if i == at {
					k.Errors, k.Typed = 1, typed
				}
				keys = append(keys, k)
			}
		}
	}
	add(6, "the ", 0, "h") // swap: the next letter first
	add(3, "fox ", 0, "g") // slip: g is next to f
	add(2, "fox ", 0, "p") // place: p is far from f
	add(1, "fox ", 0, "1") // a key the layout has not: not counted
	add(1, "fox ", 0, "")  // an old round without the wrong key: not counted
	got := Read([][]Key{keys}, places, order, testParams)
	if got.Mistakes != 11 || len(got.Kinds) != 3 {
		t.Fatalf("mistakes %d, kinds %+v", got.Mistakes, got.Kinds)
	}
	want := []Kind{{Swap, 6, 6.0 / 11}, {Slip, 3, 3.0 / 11}, {Elsewhere, 2, 2.0 / 11}}
	if !slices.Equal(got.Kinds, want) {
		t.Fatalf("kinds %+v, want %+v", got.Kinds, want)
	}
	few := Read([][]Key{keys[:20]}, places, order, testParams)
	if len(few.Kinds) != 0 {
		t.Fatalf("under %d mistakes no kind is named: %+v", testParams.MinMistakes, few.Kinds)
	}
}

func TestRead_NeighboursAreARowApartAtMost(t *testing.T) {
	kind := func(want, typed string) string {
		var keys []Key
		for range 10 {
			keys = append(keys, Key{Char: want, Errors: 1, Typed: typed}, Key{Char: " "})
		}
		return Read([][]Key{keys}, places, order, testParams).Kinds[0].Kind
	}
	for _, c := range []struct{ want, typed, kind string }{
		{"f", "g", Slip}, {"f", "r", Slip}, {"f", "v", Slip}, {"q", "a", Slip},
		{"q", "z", Elsewhere}, {"f", "j", Elsewhere}, {"a", "l", Elsewhere},
	} {
		if got := kind(c.want, c.typed); got != c.kind {
			t.Errorf("%s typed as %s: %s, want %s", c.want, c.typed, got, c.kind)
		}
	}
}
