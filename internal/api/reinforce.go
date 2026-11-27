// Reinforce: once every letter is open, Learn draws from a word level the
// learner picks, weighted to their weak letters and pairs, and the speed
// goal grows in steps.

package api

import (
	"encoding/json"
	"net/http"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/levels"
	"typingtrainer/internal/profile"
	"typingtrainer/internal/storage"
)

func (s *Server) validWordLevel(l *Language, level int) bool {
	return level >= 1 && level <= len(l.Levels)
}

// withWeakPairs is gen weighted to the letter pairs that slow this learner's
// last rounds on a keyboard down.
func (s *Server) withWeakPairs(user string, l *Language, gen *engine.Generator) (*engine.Generator, error) {
	rounds, err := s.store.RecentSessions(user, l.Info.ID, "", max(s.cfg.Levels.WeakRounds, s.cfg.Profile.Rounds))
	if err != nil {
		return nil, err
	}
	rounds = storage.OnKeyboard(rounds)
	var pairs []string
	for _, p := range s.pairsOf(rounds[:min(len(rounds), s.cfg.Levels.WeakRounds)]) {
		pairs = append(pairs, p.Pair)
	}
	return gen.WithPairs(pairs).WithFinger(s.profileOf(rounds, l).Letters), nil
}

// profileOf is what the last Profile.Rounds of these rounds (newest first)
// say about the typist's fingers and kinds of mistakes.
func (s *Server) profileOf(rounds []storage.Session, l *Language) profile.Profile {
	rounds = rounds[:min(len(rounds), s.cfg.Profile.Rounds)]
	keys := make([][]profile.Key, len(rounds))
	for i, r := range rounds {
		keys[i] = make([]profile.Key, len(r.Keys))
		for j, k := range r.Keys {
			keys[i][j] = profile.Key{Char: k.Char, Errors: k.Errors, Ms: k.Ms, Typed: k.Typed}
		}
	}
	return profile.Read(keys, l.Places, l.Fingers, s.cfg.Profile)
}

// wordLevels is the levels' names: how many of the most common words each
// one takes (a few of them, too short or with a foreign letter, are left out).
func (s *Server) wordLevels(l *Language) []int {
	return s.cfg.Levels.Words[:len(l.Levels)]
}

// POST {step}: picks Reinforce's speed goal step by hand, up or down. Mastery starts over at the new goal, as after a step
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
	if err := s.setStep(user, id, req.Step); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"step": req.Step, "targetWpm": who.base.ForLap(req.Step, s.cfg.Laps).TargetWPM})
}

// setStep moves Reinforce's speed goal to a step. Mastery starts over at the
// new goal, as after a step reached by typing. The caller holds writeMu.
func (s *Server) setStep(user, id string, step int) error {
	lap, err := s.store.Lap(user, id)
	if err != nil || step == lap {
		return err
	}
	unlocked, err := s.store.Unlocked(user, id)
	if err != nil {
		return err
	}
	stats, err := s.store.KeyStats(user, id)
	if err != nil {
		return err
	}
	return s.store.SaveLearned(user, id, storage.Learned{Stats: engine.StartLap(stats), Unlocked: unlocked, Lap: step})
}

// standing is where Reinforce's rounds (oldest first) leave its word level:
// each judged at the speed goal of its own step (levels.Judge), the last run
// at one level counted (levels.Standing). Moved is what the last round did,
// so the browser follows it only right after a round at its current level,
// and a level picked by hand starts its own run.
type standing struct {
	levels.Place
	PassesUp  int `json:"passesUp"`
	FailsDown int `json:"failsDown"`
}

func (s *Server) standing(rounds []storage.Session, base engine.Params, count int) standing {
	var rs []levels.Round
	for _, r := range rounds {
		if r.Level > 0 {
			rs = append(rs, levels.Round{Level: int(r.Level), Accuracy: r.Accuracy, WPM: r.WPM, Goal: base.ForLap(r.Lap, s.cfg.Laps).TargetWPM})
		}
	}
	p := s.cfg.Levels
	p.Words = p.Words[:min(count, len(p.Words))]
	return standing{Place: levels.Standing(rs, levels.Judge, p), PassesUp: p.PassesUp, FailsDown: p.FailsDown}
}
