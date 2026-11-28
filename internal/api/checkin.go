// The check-in when a mode is first opened: one question or a short sample
// sets where the mode starts. The
// sample's presses are never sent; only its speed and accuracy are.

package api

import (
	"encoding/json"
	"net/http"

	"typingtrainer/internal/config"
	"typingtrainer/internal/engine"
)

type checkinRequest struct {
	Mode     string  `json:"mode"`
	Answer   string  `json:"answer"`
	WPM      float64 `json:"wpm"`
	Accuracy float64 `json:"accuracy"`
}

// checkinLetters is Learn's open letters after an answer: "little" and
// "yes" open SomeLetters, "no" keeps the start. It never closes a letter.
func checkinLetters(answer string, now, all int, c config.Checkin) int {
	if answer == "little" || answer == "yes" {
		return min(all, max(now, c.SomeLetters))
	}
	return now
}

// checkinStep is the first speed goal step at least GoalMargin above the
// sample's speed, or the last step.
func checkinStep(wpm float64, base engine.Params, laps engine.LapParams, c config.Checkin) int {
	last := base.LastLap(laps)
	for step := 0; step <= last; step++ {
		if base.ForLap(step, laps).TargetWPM >= wpm+c.GoalMargin {
			return step
		}
	}
	return last
}

// pickBy is the value of the first threshold x reaches (thresholds highest
// first), else the fallback.
func pickBy(x float64, thresholds []float64, values []int, fallback int) int {
	for i, t := range thresholds {
		if i < len(values) && x >= t {
			return values[i]
		}
	}
	return fallback
}

func (s *Server) handleCheckin(w http.ResponseWriter, r *http.Request, user string, l *Language) {
	var req checkinRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, int64(s.cfg.MaxSmallBodyBytes))).Decode(&req); err != nil {
		badRequest(w, "invalid JSON")
		return
	}
	if !inRange(req.WPM, 0, 400) || !inRange(req.Accuracy, 0, 100) {
		badRequest(w, "wpm 0..400, accuracy 0..100")
		return
	}
	c, id := s.cfg.Checkin, l.Info.ID
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	switch req.Mode {
	case "learn":
		if req.Answer != "no" && req.Answer != "little" && req.Answer != "yes" {
			badRequest(w, "answer: no, little or yes")
			return
		}
		now, err := s.store.Unlocked(user, id)
		if err != nil {
			serverError(w, err)
			return
		}
		rounds, err := s.store.SessionCount(user, id, "")
		if err != nil {
			serverError(w, err)
			return
		}
		gen := l.Gen
		now = gen.Clamp(now)
		next := now
		if rounds == 0 {
			next = checkinLetters(req.Answer, now, len(gen.Order()), c)
		}
		if next != now {
			if err := s.store.SetUnlocked(user, id, next); err != nil {
				serverError(w, err)
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"unlocked": next})
	case "reinforce":
		who := s.learner(user, l)
		step := checkinStep(req.WPM, who.base, s.cfg.Laps, c)
		if err := s.setStep(user, id, step); err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"step": step, "targetWpm": who.base.ForLap(step, s.cfg.Laps).TargetWPM,
			"wordLevel": min(pickBy(req.Accuracy, c.LevelAccuracy, c.LevelFor, 1), len(l.Levels)),
		})
	case "freetype":
		writeJSON(w, http.StatusOK, map[string]any{"list": pickBy(req.WPM, c.ListSpeed, c.ListFor, s.cfg.Freetype.Lists[0])})
	default:
		badRequest(w, "mode: learn, reinforce or freetype")
	}
}
