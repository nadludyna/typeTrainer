// Why a word came: the reason the trainer weighed it up, sent with every
// portion so the round's result can show it.

package api

import (
	"strings"
	"unicode/utf8"
)

// The kinds of reasons, each followed by ":" and what it is about.
const (
	whyBack    = "back"    // a letter not typed for long
	whyPair    = "pair"    // a slow pair of letters
	whyStumble = "stumble" // a pair the rhythm stumbles on
	whyFinger  = "finger"  // the weak finger
	whyFocus   = "focus"   // the letter in focus
	whyOwned   = "owned"   // a word the typist owns, in a sprint
)

// weakSpots is what the words of a round are weighed toward.
type weakSpots struct {
	back     []string        // letters coming back
	pairs    []string        // slow pairs
	stumbles []string        // stumble pairs
	finger   string          // the weak finger, "" with none
	letters  string          // its letters
	share    float64         // the share of a word's letters that makes it the finger's word
	owned    map[string]bool // a sprint's owned words; nil outside a sprint
}

// why is the first reason that holds for the word: in a sprint the word
// being owned, else the rarest kind first; "" for a word that came by itself.
func (w weakSpots) why(word, focus string) string {
	if w.owned[word] {
		return whyOwned + ":"
	}
	for _, c := range w.back {
		if strings.Contains(word, c) {
			return whyBack + ":" + c
		}
	}
	for _, p := range w.pairs {
		if strings.Contains(word, p) {
			return whyPair + ":" + p
		}
	}
	for _, p := range w.stumbles {
		if strings.Contains(word, p) {
			return whyStumble + ":" + p
		}
	}
	if w.finger != "" && word != "" {
		on := 0
		for _, c := range word {
			if strings.ContainsRune(w.letters, c) {
				on++
			}
		}
		if float64(on) >= w.share*float64(utf8.RuneCountInString(word)) {
			return whyFinger + ":" + w.finger
		}
	}
	if focus != "" && strings.Contains(word, focus) {
		return whyFocus + ":" + focus
	}
	return ""
}

func (w weakSpots) whys(words []string, focus string) []string {
	out := make([]string, len(words))
	for i, word := range words {
		out[i] = w.why(word, focus)
	}
	return out
}
