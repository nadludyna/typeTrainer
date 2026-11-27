// The words a typist owns: typed clean and no slower than their usual word,
// in OwnHits different rounds. A round with a mistake in the word takes one
// hit away, so a word can be lost again.

package profile

import "slices"

type WordParams struct {
	Rounds  int     // the last rounds read
	OwnHits int     // clean, fast rounds a word needs to be owned
	Slack   float64 // "no slower than usual": at most this many times the typist's usual time per letter
}

type Vocabulary struct {
	Usual float64         // the typist's usual time per letter inside a word, ms (the median over words)
	Owned map[string]bool // never nil
}

// one word as typed in a round
type typedWord struct {
	word  string
	clean bool
	ms    float64 // mean time of its letters after the first, 0 when none is timed
}

// ReadWords reads the rounds, oldest first. A word is the keys between two
// spaces; its first letter's time holds the reading of the word and is left
// out. A word typed twice in a round counts once, by its worst try.
func ReadWords(rounds [][]Key, p WordParams) Vocabulary {
	out := Vocabulary{Owned: map[string]bool{}}
	perRound := make([][]typedWord, len(rounds))
	var all []float64
	for r, keys := range rounds {
		start := 0
		for i := 0; i <= len(keys); i++ {
			if i < len(keys) && keys[i].Char != " " {
				continue
			}
			if w, ok := readWord(keys[start:i]); ok {
				perRound[r] = append(perRound[r], w)
				if w.clean && w.ms > 0 {
					all = append(all, w.ms)
				}
			}
			start = i + 1
		}
	}
	if len(all) == 0 {
		return out
	}
	slices.Sort(all)
	out.Usual = all[len(all)/2]
	if len(all)%2 == 0 {
		out.Usual = (all[len(all)/2-1] + all[len(all)/2]) / 2
	}
	hits := map[string]int{}
	for _, words := range perRound {
		good, bad := map[string]bool{}, map[string]bool{}
		for _, w := range words {
			switch {
			case !w.clean:
				bad[w.word] = true
			case w.ms > 0 && w.ms <= p.Slack*out.Usual:
				good[w.word] = true
			}
		}
		for w := range bad {
			hits[w] = max(0, hits[w]-1)
		}
		for w := range good {
			if !bad[w] {
				hits[w]++
			}
		}
	}
	for w, n := range hits {
		if n >= p.OwnHits {
			out.Owned[w] = true
		}
	}
	return out
}

func readWord(keys []Key) (typedWord, bool) {
	if len(keys) < 2 {
		return typedWord{}, false
	}
	w := typedWord{clean: true}
	sum, n := 0.0, 0
	for i, k := range keys {
		w.word += k.Char
		if k.Errors > 0 {
			w.clean = false
		}
		if i > 0 && k.Ms > 0 {
			sum += k.Ms
			n++
		}
	}
	if n > 0 {
		w.ms = sum / float64(n)
	}
	return w, true
}

// OwnedOf counts the owned words among words.
func (v Vocabulary) OwnedOf(words []string) int {
	n := 0
	for _, w := range words {
		if v.Owned[w] {
			n++
		}
	}
	return n
}
