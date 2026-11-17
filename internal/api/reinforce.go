// Reinforce: once every letter is open, Learn draws from a word level the
// learner picks, weighted to their weak letters and pairs, and the speed
// goal grows in steps.

package api

import (
	"encoding/json"
	"net/http"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/storage"
)

func (s *Server) validWordLevel(l *Language, level int) bool {
	return level >= 1 && level <= len(l.Levels)
}

// reinforce is the generator of one word level for this learner, with the
// pairs that slow their last Learn rounds down.
func (s *Server) reinforce(user string, l *Language, gen *engine.Generator, level int) (*engine.Generator, error) {
	rounds, err := s.store.RecentSessions(user, l.Info.ID, "", s.cfg.Freeplay.WeakRounds)
	if err != nil {
		return nil, err
	}
	var pairs []string
	for _, p := range s.pairsOf(rounds) {
		pairs = append(pairs, p.Pair)
	}
	return l.Levels[level-1].WithPairs(pairs), nil
}

// wordLevels is the levels' names: how many of the most common words each
// one takes (a few of them, too short or with a foreign letter, are left out).
func (s *Server) wordLevels(l *Language) []int {
	return s.cfg.Freeplay.Levels[:len(l.Levels)]
}

// POST {step}: picks a speed goal step by hand, up or down, once every
// letter is open. Mastery starts over at the new goal, as after a step
// reached by typing.
func (s *Server) handleLearnStep(w http.ResponseWriter, r *http.Request, user string, l *Language) {
	var req struct {
		Step int `json:"step"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, int64(s.cfg.MaxSmallBodyBytes))).Decode(&req); err != nil {
		badRequest(w, "invalid JSON")
		return
	}
	id := l.Info.ID
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	who := s.learner(user, l)
	if req.Step < 0 || req.Step > who.base.LastLap(s.cfg.Laps) {
		badRequest(w, "step: 0 to the last step")
		return
	}
	unlocked, err := s.store.Unlocked(user, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if who.gen.Clamp(unlocked) < len(who.gen.Order()) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "step: some letters are still locked"})
		return
	}
	lap, err := s.store.Lap(user, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if req.Step != lap {
		stats, err := s.store.KeyStats(user, id)
		if err != nil {
			serverError(w, err)
			return
		}
		if err := s.store.SaveLearned(user, id, storage.Learned{Stats: engine.StartLap(stats), Unlocked: unlocked, Lap: req.Step}); err != nil {
			serverError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"step": req.Step, "targetWpm": who.base.ForLap(req.Step, s.cfg.Laps).TargetWPM})
}
