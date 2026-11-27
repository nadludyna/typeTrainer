// What time does to a typist: a letter not typed for days comes back into
// the words, and accuracy that falls round after round in one sitting says
// the typist is tired.

package profile

import (
	"cmp"
	"slices"
	"time"
)

type TimeParams struct {
	Forget    time.Duration // a letter not typed for this long comes back
	Returning int           // at most this many letters come back at once
	Sitting   time.Duration // rounds closer than this are one sitting
	TiredRuns int           // falls of accuracy in a row that say "tired"
	TiredDrop float64       // and by this many points in all, at least
	ColdAfter time.Duration // the first round after a break this long is a warm-up
}

// Seen is when each letter was last typed in the rounds read, and when the
// oldest of those rounds ended (zero with no rounds).
type Seen struct {
	Last   map[string]time.Time
	Oldest time.Time
}

// Returning is the letters (of those given) not typed for Forget or longer,
// the longest away first. A letter never typed in the rounds read counts
// from the oldest round; with no rounds nothing returns.
func Returning(seen Seen, letters []string, now time.Time, p TimeParams) []string {
	if seen.Oldest.IsZero() || p.Forget <= 0 {
		return nil
	}
	type away struct {
		letter string
		since  time.Time
	}
	var out []away
	for _, c := range letters {
		since, ok := seen.Last[c]
		if !ok {
			since = seen.Oldest
		}
		if now.Sub(since) >= p.Forget {
			out = append(out, away{c, since})
		}
	}
	slices.SortFunc(out, func(a, b away) int { return cmp.Or(a.since.Compare(b.since), cmp.Compare(a.letter, b.letter)) })
	out = out[:min(len(out), max(p.Returning, 0))]
	letters = make([]string, len(out))
	for i, a := range out {
		letters[i] = a.letter
	}
	return letters
}

// Stamp is a round's end and its accuracy, %.
type Stamp struct {
	At       time.Time
	Accuracy float64
}

// Tired reads the rounds, oldest first: the last TiredRuns + 1 rounds are
// one sitting, each less accurate than the one before, by TiredDrop points
// or more from the first to the last.
func Tired(rounds []Stamp, p TimeParams) bool {
	n := p.TiredRuns
	if n <= 0 || len(rounds) < n+1 {
		return false
	}
	last := rounds[len(rounds)-n-1:]
	for i := 1; i < len(last); i++ {
		if last[i].Accuracy >= last[i-1].Accuracy || last[i].At.Sub(last[i-1].At) > p.Sitting {
			return false
		}
	}
	return last[0].Accuracy-last[n].Accuracy >= p.TiredDrop
}

// Cold says for each round, oldest first, whether it is a warm-up: the
// first round ever, or the first after a break of ColdAfter or longer.
func Cold(ends []time.Time, p TimeParams) []bool {
	out := make([]bool, len(ends))
	for i := range ends {
		out[i] = p.ColdAfter > 0 && (i == 0 || ends[i].Sub(ends[i-1]) >= p.ColdAfter)
	}
	return out
}
