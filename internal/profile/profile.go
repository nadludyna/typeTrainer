// Package profile reads what a typist's rounds say about their hands: how
// each finger does against the typist's own average, and why presses go
// wrong (a slip to a neighbouring key, a key somewhere else, two letters
// swapped). It reads the raw keys of finished rounds and a keyboard layout;
// it never touches the letter model.
package profile

import (
	"cmp"
	"math"
	"slices"
)

type Params struct {
	Rounds        int     // the last rounds read
	MinPresses    int     // a finger with fewer presses is not judged
	WeakRatio     float64 // a weak finger misses this many times the typist's own rate
	WeakMargin    float64 // and at least this much more, as a share of presses
	Prior         float64 // presses imagined at the typist's own rate in each finger
	MinMistakes   int     // with fewer mistakes with a known key no kind is named
	NeighbourDist float64 // keys this close (in key widths, a row apart at most) are neighbours
}

// Key is one press: the wanted character, how many wrong presses it took,
// the time to the first press, the first wrong key.
type Key struct {
	Char   string
	Errors int
	Ms     float64
	Typed  string
}

// Place is where a key stands on the keyboard and which finger presses it.
type Place struct {
	Finger string
	Row    int
	X      float64 // in key widths from the left edge of the letter block
}

type Finger struct {
	Finger   string  `json:"finger"`
	Presses  int     `json:"presses"`
	Mistakes int     `json:"mistakes"`
	Rate     float64 `json:"rate"` // its share of mistakes, pulled toward the typist's own
	Ms       float64 `json:"ms"`   // mean time of its clean presses, 0 with none
	Weak     bool    `json:"weak"`
}

// The kinds of mistakes.
const (
	Slip      = "slip"  // a neighbouring key: the finger went a little off
	Elsewhere = "place" // a key somewhere else: where the letter is, is not known yet
	Swap      = "swap"  // the next letter typed first: the hands are out of step
)

type Kind struct {
	Kind  string  `json:"kind"`
	Count int     `json:"count"`
	Share float64 `json:"share"`
}

type Profile struct {
	Presses  int      `json:"presses"`
	Rate     float64  `json:"rate"`     // the typist's own share of mistakes
	Fingers  []Finger `json:"fingers"`  // in the order given
	Weak     string   `json:"weak"`     // the weakest weak finger, "" with none
	Letters  []string `json:"letters"`  // its letters
	Mistakes int      `json:"mistakes"` // mistakes whose wrong key is known
	Kinds    []Kind   `json:"kinds"`    // the most common first; empty under MinMistakes
}

// Read reads the rounds (any order) against the layout's places; order is
// the fingers as they are listed. Presses of keys the layout does not have
// (space, marks) are not read.
func Read(rounds [][]Key, places map[string]Place, order []string, p Params) Profile {
	type sum struct {
		n, bad, timed int
		ms            float64
	}
	fingers := map[string]*sum{}
	kinds := map[string]int{}
	out := Profile{Fingers: []Finger{}, Letters: []string{}, Kinds: []Kind{}}
	bad := 0
	for _, keys := range rounds {
		for i, k := range keys {
			at, ok := places[k.Char]
			if !ok {
				continue
			}
			f := fingers[at.Finger]
			if f == nil {
				f = &sum{}
				fingers[at.Finger] = f
			}
			f.n++
			out.Presses++
			if k.Errors == 0 {
				if k.Ms > 0 {
					f.timed++
					f.ms += k.Ms
				}
				continue
			}
			f.bad++
			bad++
			typed, known := places[k.Typed]
			if !known || k.Typed == k.Char {
				continue
			}
			out.Mistakes++
			switch {
			case i+1 < len(keys) && keys[i+1].Char == k.Typed:
				kinds[Swap]++
			case math.Abs(float64(typed.Row-at.Row)) <= 1 && math.Abs(typed.X-at.X) <= p.NeighbourDist:
				kinds[Slip]++
			default:
				kinds[Elsewhere]++
			}
		}
	}
	if out.Presses == 0 {
		return out
	}
	out.Rate = float64(bad) / float64(out.Presses)
	worst := 0.0
	for _, name := range order {
		f := fingers[name]
		if f == nil {
			continue
		}
		x := Finger{Finger: name, Presses: f.n, Mistakes: f.bad}
		x.Rate = (float64(f.bad) + p.Prior*out.Rate) / (float64(f.n) + p.Prior)
		if f.timed > 0 {
			x.Ms = f.ms / float64(f.timed)
		}
		x.Weak = f.n >= p.MinPresses && out.Rate > 0 && x.Rate >= p.WeakRatio*out.Rate && x.Rate-out.Rate >= p.WeakMargin
		if x.Weak && x.Rate > worst {
			worst, out.Weak = x.Rate, name
		}
		out.Fingers = append(out.Fingers, x)
	}
	if out.Weak != "" {
		for c, at := range places {
			if at.Finger == out.Weak {
				out.Letters = append(out.Letters, c)
			}
		}
		slices.Sort(out.Letters)
	}
	if out.Mistakes >= p.MinMistakes && out.Mistakes > 0 {
		for kind, n := range kinds {
			out.Kinds = append(out.Kinds, Kind{Kind: kind, Count: n, Share: float64(n) / float64(out.Mistakes)})
		}
		slices.SortFunc(out.Kinds, func(a, b Kind) int { return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Kind, b.Kind)) })
	}
	return out
}
