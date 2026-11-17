// Placement test and starting Learn over.

package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/storage"
)

type placementRequest struct {
	Keys     []engine.Sample `json:"keys"`
	WPM      float64         `json:"wpm"`
	Accuracy float64         `json:"accuracy"`
	Skip     bool            `json:"skip"`
}

// POST: the test's presses, speed and accuracy, or {skip: true}. The test
// runs once; "Start Learn over" allows it again.
func (s *Server) handlePlacement(w http.ResponseWriter, r *http.Request, user string, l *Language) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(s.cfg.MaxBodyBytes))
	var req placementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid JSON")
		return
	}
	if msg := validateSamples(req.Keys, s.cfg.MaxEvents, l.Info.Alphabet); msg != "" {
		badRequest(w, "keys: "+msg)
		return
	}
	if req.WPM < 0 || req.WPM > 400 || req.Accuracy < 0 || req.Accuracy > 100 {
		badRequest(w, "wpm or accuracy out of range")
		return
	}
	if !req.Skip && len(req.Keys) == 0 {
		badRequest(w, "keys: the test has no presses")
		return
	}
	id := l.Info.ID
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	pl, err := s.store.Placement(user, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if pl.Done {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "placement: already done"})
		return
	}
	pl = storage.Placement{Done: true}
	if !req.Skip {
		goal, known := engine.Place(req.Keys, req.WPM, req.Accuracy, s.cfg.Placement)
		pl.Goal, pl.Known = goal, strings.Join(known, "")
	}
	if err := s.store.SetPlacement(user, id, pl); err != nil {
		serverError(w, err)
		return
	}
	who, err := s.learner(user, l)
	if err != nil {
		serverError(w, err)
		return
	}
	if !req.Skip {
		stats, err := s.store.KeyStats(user, id)
		if err != nil {
			serverError(w, err)
			return
		}
		if err := s.store.SaveKeyStats(user, id, engine.Apply(stats, req.Keys, who.base)); err != nil {
			serverError(w, err)
			return
		}
	}
	unlocked, err := s.store.Unlocked(user, id)
	if err != nil {
		serverError(w, err)
		return
	}
	unlocked = max(unlocked, who.gen.Start())
	if err := s.store.SetUnlocked(user, id, unlocked); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"goal": who.base.TargetWPM, "known": pl.Known, "unlocked": unlocked})
}

// POST: Learn starts over, test included; the other modes keep their rounds.
func (s *Server) handleLearnReset(w http.ResponseWriter, r *http.Request, user string, l *Language) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.store.ResetLearn(user, l.Info.ID); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) validPlacement(pl storage.Placement, alphabet string) string {
	p := s.cfg.Placement
	if pl.Goal != 0 && (pl.Goal < p.MinGoal || pl.Goal > p.MaxGoal) {
		return "goal out of range"
	}
	seen := map[rune]bool{}
	for _, r := range pl.Known {
		if r == utf8.RuneError || seen[r] || !strings.ContainsRune(alphabet, r) {
			return "known: letters of the alphabet, each once"
		}
		seen[r] = true
	}
	return ""
}
