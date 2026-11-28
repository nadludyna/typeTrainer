// Freetype's rhythm: a typist's usual time between two letters of a word,
// the time past which a press is a stumble, and the letter pairs they
// stumble on far more often than on the rest. Freetype brings words with
// those pairs more often.

package freetype

import (
	"cmp"
	"math"
	"slices"
	"unicode"
)

// Key is one press as the rhythm reads it.
type Key struct {
	Char   string
	Errors int
	Ms     float64
}

type Rhythm struct {
	Median    float64   `json:"median"`    // the usual time between two letters of a word, ms
	Spread    float64   `json:"spread"`    // how far times stray from it, ms (scaled MAD)
	Threshold float64   `json:"threshold"` // a press slower than this is a stumble, ms
	Rate      float64   `json:"rate"`      // the share of presses that are stumbles
	Presses   int       `json:"presses"`   // the presses read
	Pairs     []Stumble `json:"pairs"`     // the stumble pairs, the worst first
}

type Stumble struct {
	Pair     string  `json:"pair"`
	Rate     float64 `json:"rate"` // its share of stumbles, pulled toward the typist's own
	Count    int     `json:"count"`
	Stumbles int     `json:"stumbles"`
}

// madScale turns the median absolute deviation into a standard deviation
// for times spread normally, so StumbleSpreads reads as "standard deviations".
const madScale = 1.4826

// ReadRhythm reads the presses inside words: a letter after a letter, right
// on the first try, timed, and no slower than RhythmMaxMs (a longer gap is a
// rest). With fewer than RhythmMinPresses of them it reads nothing.
func ReadRhythm(rounds [][]Key, p Params) Rhythm {
	type press struct {
		pair string
		ms   float64
	}
	var presses []press
	for _, keys := range rounds {
		for i := 1; i < len(keys); i++ {
			a, b, k := []rune(keys[i-1].Char), []rune(keys[i].Char), keys[i]
			if len(a) != 1 || len(b) != 1 || !isLower(a[0]) || !isLower(b[0]) || k.Errors > 0 || k.Ms <= 0 || k.Ms > p.RhythmMaxMs {
				continue
			}
			presses = append(presses, press{string(a) + string(b), k.Ms})
		}
	}
	out := Rhythm{Presses: len(presses), Pairs: []Stumble{}}
	if len(presses) == 0 || len(presses) < p.RhythmMinPresses {
		return out
	}
	ms := make([]float64, len(presses))
	for i, x := range presses {
		ms[i] = x.ms
	}
	out.Median = median(ms)
	for i := range ms {
		ms[i] = math.Abs(ms[i] - out.Median)
	}
	out.Spread = madScale * median(ms)
	out.Threshold = max(out.Median+p.StumbleSpreads*out.Spread, p.StumbleLeast*out.Median)

	type count struct{ n, stumbles int }
	pairs := map[string]*count{}
	all := 0
	for _, x := range presses {
		c := pairs[x.pair]
		if c == nil {
			c = &count{}
			pairs[x.pair] = c
		}
		c.n++
		if x.ms > out.Threshold {
			c.stumbles++
			all++
		}
	}
	out.Rate = float64(all) / float64(len(presses))
	for pair, c := range pairs {
		if c.n < p.StumbleMinPresses || c.stumbles < p.StumbleMinCount {
			continue
		}
		rate := (float64(c.stumbles) + p.StumblePrior*out.Rate) / (float64(c.n) + p.StumblePrior)
		if rate >= p.StumbleRatio*out.Rate {
			out.Pairs = append(out.Pairs, Stumble{Pair: pair, Rate: rate, Count: c.n, Stumbles: c.stumbles})
		}
	}
	slices.SortFunc(out.Pairs, func(x, y Stumble) int {
		return cmp.Or(cmp.Compare(y.Rate, x.Rate), cmp.Compare(x.Pair, y.Pair))
	})
	out.Pairs = out.Pairs[:min(len(out.Pairs), max(p.StumblePairs, 0))]
	return out
}

// median of v; v is reordered.
func median(v []float64) float64 {
	slices.Sort(v)
	n := len(v)
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

func isLower(r rune) bool { return unicode.IsLetter(r) && unicode.IsLower(r) }
