// Helpers of the rounds outside Learn's letters: how many portions a round
// starts with, the weak pairs of a set of rounds, and a mode's history for
// its stats.

package api

import (
	"slices"

	"typingtrainer/internal/levels"
	"typingtrainer/internal/storage"
)

func (s *Server) validStartWords(n int) bool { return n >= 0 && n <= s.cfg.Freetype.MaxStartWords }

func (s *Server) startPortions(start bool, words int) int {
	if !start || words <= 0 {
		return 1
	}
	p := s.cfg.Freetype
	return (words + p.PortionWords - 1) / p.PortionWords
}

// pairsOf is the letter pairs that slow these rounds down the most.
func (s *Server) pairsOf(rounds []storage.Session) []levels.Pair {
	keys := make([][]levels.Key, len(rounds))
	for i, r := range rounds {
		keys[i] = make([]levels.Key, len(r.Keys))
		for j, k := range r.Keys {
			keys[i][j] = levels.Key{Char: k.Char, Errors: k.Errors, Ms: k.Ms}
		}
	}
	return levels.WeakPairs(keys, s.cfg.Levels)
}

func inRange(v, lo, hi float64) bool { return v >= lo && v <= hi }

func (s *Server) history(user, lang, mode, length string) (map[string]any, error) {
	count, err := s.store.FreetypeSessionCount(user, lang, mode, length)
	if err != nil {
		return nil, err
	}
	all, err := s.store.RecentFreetypeSessions(user, lang, mode, length, count)
	if err != nil {
		return nil, err
	}
	sessions := slices.Clone(all[:min(len(all), 50)])
	lengths, err := s.store.FreetypeSessionLengths(user, lang, mode)
	if err != nil {
		return nil, err
	}
	bests, err := s.store.BestFreetypeRounds(user, lang, mode)
	if err != nil {
		return nil, err
	}
	return map[string]any{"sessions": summaries(sessions), "sessionCount": count, "lengths": lengths, "bests": bests}, nil
}
