// A finished round as the page sends it, its checks, and the rounds a mode
// reads back.

package trainer

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/storage"
)

// Result is the end of a round, sent by every mode.
type Result struct {
	ID        string          `json:"id"`      // the browser's round id, the same in every try
	Samples   []engine.Sample `json:"samples"` // the presses not sent with a portion yet
	Keys      []engine.Sample `json:"keys"`    // every press of the round: kept raw, learned from once
	WPM       float64         `json:"wpm"`
	Accuracy  float64         `json:"accuracy"`
	Chars     int             `json:"chars"`
	PerSecond []int           `json:"perSecond"` // keypresses in each full second, for the consistency
	Length    string          `json:"length"`
	Kind      string          `json:"kind"`
	Level     int             `json:"level"` // Reinforce's word level
	List      int             `json:"list"`  // Freetype's word list
}

// Check is what is wrong with the result, "" when nothing is.
func (t *Trainer) Check(res *Result) string {
	if res.Chars == 0 {
		res.Chars = len(res.Keys)
	}
	c := t.Cfg
	switch {
	case res.Chars < 1 || res.Chars > c.MaxRoundChars:
		return "chars: out of range"
	case res.WPM < 0 || res.WPM > 400 || res.Accuracy < 0 || res.Accuracy > 100:
		return "wpm or accuracy: out of range"
	case len(res.PerSecond) > c.MaxEvents || slices.ContainsFunc(res.PerSecond, func(n int) bool { return n < 0 || n > c.MaxEvents }):
		return "perSecond: out of range"
	case !storage.ValidKind(res.Kind):
		return "kind: unknown kind of round"
	case !slices.Contains(c.Lengths, res.Length):
		return "length: unknown round length"
	case !ValidID(res.ID):
		return "id: at most 64 letters, digits and -"
	}
	if msg := t.CheckSamples(res.Samples); msg != "" {
		return msg
	}
	if msg := t.CheckSamples(res.Keys); msg != "" {
		return "keys: " + msg
	}
	return ""
}

// CheckSamples: each sample is one letter of the alphabet or a space.
func (t *Trainer) CheckSamples(samples []engine.Sample) string {
	if len(samples) > t.Cfg.MaxEvents {
		return "samples: too many"
	}
	for _, s := range samples {
		if utf8.RuneCountInString(s.Char) != 1 || s.Ms < 0 || s.Errors < 0 {
			return "samples: each needs exactly one char, errors >= 0 and ms >= 0"
		}
		if s.Typed != "" && (utf8.RuneCountInString(s.Typed) != 1 || s.Errors == 0) {
			return "samples: typed is one char, and only with errors"
		}
		if s.Char != " " && !strings.Contains(t.Cfg.Alphabet, s.Char) {
			return "samples: " + s.Char + " is not a letter"
		}
	}
	return ""
}

func ValidID(id string) bool {
	return len(id) <= 64 && strings.Trim(id, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-") == ""
}

// Round is the round to keep for the result, typed in mode.
func (t *Trainer) Round(res Result, mode string) storage.Round {
	r := storage.Round{ID: res.ID, Mode: mode, Kind: res.Kind, Length: res.Length, WPM: res.WPM, Accuracy: res.Accuracy,
		Chars: res.Chars, Keys: res.Keys, FinishedAt: time.Now().UTC()}
	if v, ok := engine.Consistency(res.PerSecond, t.Cfg.ConsistencyMinSeconds); ok {
		r.Consistency = &v
	}
	return r
}

// Rounds is the typist's rounds of a mode, newest first ("" every mode, a
// prefix such as storage.FreetypePrefix all of its lists), at most limit (0:
// all). Learning is Learn's and Reinforce's together.
func (t *Trainer) Rounds(user, mode string, limit int) ([]storage.Round, error) {
	if mode != Learning {
		return t.Store.Rounds(user, mode, limit)
	}
	learn, err := t.Store.Rounds(user, storage.ModeLearn, limit)
	if err != nil {
		return nil, err
	}
	reinforce, err := t.Store.Rounds(user, storage.ModeReinforce, limit)
	if err != nil {
		return nil, err
	}
	all := append(learn, reinforce...)
	slices.SortStableFunc(all, func(a, b storage.Round) int { return b.FinishedAt.Compare(a.FinishedAt) })
	if limit > 0 {
		all = all[:min(limit, len(all))]
	}
	return all, nil
}

// Learning stands for Learn's and Reinforce's rounds together in Rounds.
const Learning = "learning"

// History is what each mode's stats show of its rounds of one length.
type History struct {
	Rounds  []storage.Round          `json:"rounds"` // the last 50, newest first, without their presses
	Count   int                      `json:"count"`
	Lengths []string                 `json:"lengths"`
	Bests   map[string]storage.Round `json:"bests"`
}

// HistoryOf is the history of these rounds (newest first) of one length
// ("" every length).
func HistoryOf(rounds []storage.Round, length string) History {
	h := History{Lengths: storage.Lengths(rounds), Bests: storage.Bests(rounds), Rounds: []storage.Round{}}
	for _, r := range rounds {
		if length != "" && r.Length != length {
			continue
		}
		h.Count++
		if len(h.Rounds) < 50 {
			h.Rounds = append(h.Rounds, r.Summary())
		}
	}
	return h
}

// Keys is the presses of these rounds, one slice per round.
func Keys(rounds []storage.Round) [][]engine.Sample {
	out := make([][]engine.Sample, len(rounds))
	for i, r := range rounds {
		out[i] = r.Keys
	}
	return out
}
