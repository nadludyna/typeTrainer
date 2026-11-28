// What a typist's rounds say about them: the weak pairs, the weak finger,
// the rhythm, the owned words, tiredness and where a level stands; and why
// each word of a portion came.

package trainer

import (
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"typingtrainer/internal/freetype"
	"typingtrainer/internal/levels"
	"typingtrainer/internal/profile"
	"typingtrainer/internal/storage"
)

// Pairs is the letter pairs that slow these rounds down the most.
func (t *Trainer) Pairs(rounds []storage.Round) []levels.Pair {
	return levels.WeakPairs(Keys(rounds), t.Cfg.Levels)
}

// Profile is what the last Profile.Rounds of these rounds (newest first) say
// about the fingers and the kinds of mistakes.
func (t *Trainer) Profile(rounds []storage.Round) profile.Profile {
	rounds = rounds[:min(len(rounds), t.Cfg.Profile.Rounds)]
	return profile.Read(Keys(rounds), t.Words.Places, t.Words.Fingers, t.Cfg.Profile)
}

// Rhythm is the rhythm of the last RhythmRounds of these Freetype rounds.
func (t *Trainer) Rhythm(rounds []storage.Round) freetype.Rhythm {
	rounds = rounds[:min(len(rounds), t.Cfg.Freetype.RhythmRounds)]
	return freetype.ReadRhythm(Keys(rounds), t.Cfg.Freetype)
}

// Tired says whether these rounds (newest first) end in a sitting where
// accuracy fell round after round.
func (t *Trainer) Tired(rounds []storage.Round) bool {
	stamps := make([]profile.Stamp, len(rounds))
	for i, r := range rounds {
		stamps[len(rounds)-1-i] = profile.Stamp{At: r.FinishedAt, Accuracy: r.Accuracy}
	}
	return profile.Tired(stamps, t.Cfg.Time)
}

// Standing is where a mode's rounds leave its level (levels.Standing), with
// the rule's counts for the page.
type Standing struct {
	levels.Place
	PassesUp  int `json:"passesUp"`
	FailsDown int `json:"failsDown"`
}

// StandingOf judges these rounds (newest first), each at the level it was
// typed at (level: 1.., 0 to skip it), over count levels. Only usual rounds
// are judged, and a round typed cold, after a long break, counts for nothing.
func (t *Trainer) StandingOf(rounds []storage.Round, count int, level func(storage.Round) (int, levels.Round),
	judge func(levels.Round, levels.Params) levels.Verdict) Standing {
	oldest := slices.Clone(rounds)
	slices.Reverse(oldest)
	ends := make([]time.Time, len(oldest))
	for i, r := range oldest {
		ends[i] = r.FinishedAt
	}
	cold := profile.Cold(ends, t.Cfg.Time)
	var rs []levels.Round
	for i, r := range oldest {
		if r.Kind != "" {
			continue
		}
		if n, lr := level(r); n > 0 {
			lr.Level, lr.Cold = n, cold[i]
			rs = append(rs, lr)
		}
	}
	p := t.Cfg.Levels
	return Standing{Place: levels.Standing(rs, judge, count, p), PassesUp: p.PassesUp, FailsDown: p.FailsDown}
}

// ---------- the owned words ----------

type vocabCache struct {
	mu      sync.Mutex
	entries map[string]vocabAt
}

type vocabAt struct {
	at   time.Time
	v    profile.Vocabulary
	seen profile.Seen // when each letter was last typed in Learn and Reinforce
}

// vocabKept is how long a typist's owned words are kept between the
// portions of one round.
const vocabKept = time.Hour

// Vocabulary is the words the typist owns, read from the last Owned.Rounds
// rounds of Learn and Reinforce and as many of Freetype, and when each letter
// was last typed. It is read again when a round starts (fresh) and kept for
// the round's other portions.
func (t *Trainer) Vocabulary(user string, fresh bool) (profile.Vocabulary, profile.Seen, error) {
	c := &t.vocab
	c.mu.Lock()
	kept, ok := c.entries[user]
	c.mu.Unlock()
	if ok && !fresh && time.Since(kept.at) < vocabKept {
		return kept.v, kept.seen, nil
	}
	n := t.Cfg.Owned.Rounds
	learning, err := t.Rounds(user, Learning, n)
	if err != nil {
		return profile.Vocabulary{}, profile.Seen{}, err
	}
	seen := profile.Seen{Last: map[string]time.Time{}}
	for _, r := range learning {
		if seen.Oldest.IsZero() || r.FinishedAt.Before(seen.Oldest) {
			seen.Oldest = r.FinishedAt
		}
		for _, k := range r.Keys {
			if r.FinishedAt.After(seen.Last[k.Char]) {
				seen.Last[k.Char] = r.FinishedAt
			}
		}
	}
	free, err := t.Rounds(user, storage.FreetypePrefix, n)
	if err != nil {
		return profile.Vocabulary{}, profile.Seen{}, err
	}
	rounds := append(learning, free...)
	slices.SortStableFunc(rounds, func(a, b storage.Round) int { return a.FinishedAt.Compare(b.FinishedAt) })
	v := profile.ReadWords(Keys(rounds), t.Cfg.Owned)
	c.mu.Lock()
	for k, old := range c.entries {
		if time.Since(old.at) >= vocabKept {
			delete(c.entries, k)
		}
	}
	c.entries[user] = vocabAt{at: time.Now(), v: v, seen: seen}
	c.mu.Unlock()
	return v, seen, nil
}

// OwnedPerList is how many of the n most common words the typist owns, for each n.
func (t *Trainer) OwnedPerList(v profile.Vocabulary, sizes []int) []int {
	out := make([]int, len(sizes))
	for i, n := range sizes {
		out[i] = v.OwnedOf(t.Words.Picker.Top(n))
	}
	return out
}

// ---------- why a word came ----------

// The kinds of reasons, each followed by ":" and what it is about.
const (
	whyBack    = "back"    // a letter not typed for long
	whyPair    = "pair"    // a slow pair of letters
	whyStumble = "stumble" // a pair the rhythm stumbles on
	whyFinger  = "finger"  // the weak finger
	whyFocus   = "focus"   // the letter in focus
	whyOwned   = "owned"   // a word the typist owns, in a sprint
)

// Spots is what the words of a round are weighed toward.
type Spots struct {
	Back     []string        // letters coming back
	Pairs    []string        // slow pairs
	Stumbles []string        // stumble pairs
	Finger   string          // the weak finger, "" with none
	Letters  string          // its letters
	Share    float64         // the share of a word's letters that makes it the finger's word
	Owned    map[string]bool // a sprint's owned words; nil outside a sprint
}

// Why is the first reason that holds for the word: in a sprint the word
// being owned, else the rarest kind first; "" for a word that came by itself.
func (s Spots) Why(word, focus string) string {
	if s.Owned[word] {
		return whyOwned + ":"
	}
	for _, c := range s.Back {
		if strings.Contains(word, c) {
			return whyBack + ":" + c
		}
	}
	for _, p := range s.Pairs {
		if strings.Contains(word, p) {
			return whyPair + ":" + p
		}
	}
	for _, p := range s.Stumbles {
		if strings.Contains(word, p) {
			return whyStumble + ":" + p
		}
	}
	if s.Finger != "" && word != "" {
		on := 0
		for _, c := range word {
			if strings.ContainsRune(s.Letters, c) {
				on++
			}
		}
		if float64(on) >= s.Share*float64(utf8.RuneCountInString(word)) {
			return whyFinger + ":" + s.Finger
		}
	}
	if focus != "" && strings.Contains(word, focus) {
		return whyFocus + ":" + focus
	}
	return ""
}

func (s Spots) Whys(words []string, focus string) []string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = s.Why(w, focus)
	}
	return out
}
