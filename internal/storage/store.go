// Package storage keeps each typist's progress and rounds behind Store: in
// memory (tests, GitHub Pages) or in PostgreSQL. Both pass store_test.go.
package storage

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"typingtrainer/internal/engine"
)

// Progress is what Learn and Reinforce have learned about a typist.
type Progress struct {
	Letters  map[string]engine.KeyStat `json:"letters"`
	Unlocked int                       `json:"unlocked"` // Learn's open letters
	Lap      int                       `json:"lap"`      // Reinforce's speed goal step
}

// Round is one finished round of any mode, with every press it was typed with.
type Round struct {
	ID          string          `json:"id,omitempty"` // the browser's id: a round sent twice is kept once
	Mode        string          `json:"mode"`         // ModeLearn, ModeReinforce or FreetypeMode(n)
	Kind        string          `json:"kind,omitempty"`
	Level       int             `json:"level,omitempty"` // Reinforce's word level, 1..
	Lap         int             `json:"lap,omitempty"`   // the speed goal step it was typed at
	Length      string          `json:"length"`          // "words-30" or "time-60"
	WPM         float64         `json:"wpm"`
	Accuracy    float64         `json:"accuracy"`
	Chars       int             `json:"chars"`
	Consistency *float64        `json:"consistency,omitempty"`
	Unlocked    int             `json:"unlocked,omitempty"` // Learn's open letters after it
	FinishedAt  time.Time       `json:"finishedAt"`
	Keys        []engine.Sample `json:"keys,omitempty"`
}

const (
	ModeLearn     = "learn"
	ModeReinforce = "reinforce"
	// FreetypePrefix starts the mode of every Freetype round: "freetype-1000".
	FreetypePrefix = "freetype-"
)

func FreetypeMode(list int) string { return FreetypePrefix + strconv.Itoa(list) }

// FreetypeList is the word list of a Freetype round's mode.
func FreetypeList(mode string) (int, bool) {
	num, ok := strings.CutPrefix(mode, FreetypePrefix)
	n, err := strconv.Atoi(num)
	if !ok || err != nil || n <= 0 || strconv.Itoa(n) != num {
		return 0, false
	}
	return n, true
}

// The kinds of round besides the usual one. A round of a kind is kept and
// shown but moves no word level or list.
const (
	KindWeak   = "weak"   // built from the typist's weak spots
	KindClean  = "clean"  // ends at the third mistake
	KindSprint = "sprint" // 15 seconds of the words the typist owns
)

func ValidKind(k string) bool {
	return k == "" || k == KindWeak || k == KindClean || k == KindSprint
}

// Usual is the rounds of no special kind: the ones a level is judged by.
func Usual(rounds []Round) []Round {
	out := make([]Round, 0, len(rounds))
	for _, r := range rounds {
		if r.Kind == "" {
			out = append(out, r)
		}
	}
	return out
}

// Summary is the round without its presses, for the stats.
func (r Round) Summary() Round {
	r.Keys = nil
	return r
}

// BestKey is what a round's best is kept under: its length, or KindSprint
// for a sprint, so a sprint of owned words is no usual round's best.
func BestKey(r Round) string {
	if r.Kind == KindSprint {
		return KindSprint
	}
	return r.Length
}

// Bests is the fastest round of each BestKey, without its presses.
func Bests(rounds []Round) map[string]Round {
	out := map[string]Round{}
	for _, r := range rounds {
		k := BestKey(r)
		if b, ok := out[k]; !ok || r.WPM > b.WPM {
			out[k] = r.Summary()
		}
	}
	return out
}

// Lengths is the round lengths these rounds have, sorted.
func Lengths(rounds []Round) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range rounds {
		if !seen[r.Length] {
			seen[r.Length] = true
			out = append(out, r.Length)
		}
	}
	sort.Strings(out)
	return out
}

// LocalUser is the one typist of a store without guest cookies (GitHub
// Pages); a guest's id is GuestPrefix + the hash of its cookie.
const (
	LocalUser   = "local"
	GuestPrefix = "g"
)

type Store interface {
	// Progress is the typist's progress; a new typist has none yet.
	Progress(user string) (Progress, error)
	// Save writes the progress and the round, either of them nil to leave
	// it, all or nothing. A round whose ID is already kept is not saved again.
	Save(user string, p *Progress, r *Round) error
	HasRound(user, id string) (bool, error)
	// Rounds is the typist's rounds whose mode starts with prefix ("": every
	// mode), newest first, at most limit (0: all of them).
	Rounds(user, prefix string, limit int) ([]Round, error)
	// Reset deletes everything of the typist; ResetLearn deletes Learn's and
	// Reinforce's progress and rounds and keeps Freetype's.
	Reset(user string) error
	ResetLearn(user string) error
	// Import replaces everything of the typist, all or nothing. rounds are
	// oldest first.
	Import(user string, p Progress, rounds []Round) error
}
