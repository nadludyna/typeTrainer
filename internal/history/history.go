// Package history reads conclusions from finished rounds for the stats.
package history

import (
	"sort"

	"typingtrainer/internal/storage"
)

type Confusion struct {
	Want  string `json:"want"`
	Typed string `json:"typed"`
	N     int    `json:"n"`
}

func Confusions(sessions []storage.Session, least, limit int) []Confusion {
	type pair struct{ want, typed string }
	count := map[pair]int{}
	for _, s := range sessions {
		for _, k := range s.Keys {
			if k.Typed != "" && k.Typed != k.Char {
				count[pair{k.Char, k.Typed}]++
			}
		}
	}
	out := []Confusion{}
	for p, n := range count {
		if n >= least {
			out = append(out, Confusion{Want: p.want, Typed: p.typed, N: n})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		if out[i].Want != out[j].Want {
			return out[i].Want < out[j].Want
		}
		return out[i].Typed < out[j].Typed
	})
	return out[:min(len(out), limit)]
}
