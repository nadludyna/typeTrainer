// Package learn is the Learn mode: it trains accuracy. Letters open one at
// a time from the home row; the words lean on the letter in focus and the
// weak spots.
package learn

import (
	"net/http"

	"typingtrainer/internal/history"
	"typingtrainer/internal/profile"
	"typingtrainer/internal/storage"
	"typingtrainer/internal/trainer"
)

type mode struct{ t *trainer.Trainer }

func Register(mux *http.ServeMux, t *trainer.Trainer) {
	m := mode{t}
	mux.HandleFunc("POST /api/learn/portion", t.Route(m.portion))
	mux.HandleFunc("POST /api/learn/results", t.Route(m.results))
	mux.HandleFunc("GET /api/learn/stats", t.Route(m.stats))
	mux.HandleFunc("POST /api/learn/checkin", t.Route(m.checkin))
	mux.HandleFunc("POST /api/learn/reset", t.Route(m.reset))
}

func (m mode) portion(w http.ResponseWriter, r *http.Request, user string) {
	var req trainer.PortionRequest
	if !trainer.Read(w, r, m.t.Cfg.MaxBodyBytes, &req) {
		return
	}
	if msg := m.t.CheckPortion(req); msg != "" {
		trainer.Bad(w, msg)
		return
	}
	if req.Kind != "" || req.Level != 0 {
		trainer.Bad(w, "kind, level: Learn has only the usual round")
		return
	}
	out, err := m.t.LetterPortions(user, req, false)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	trainer.JSON(w, out)
}

// Result is what a finished round changed.
type Result struct {
	Unlocked    int      `json:"unlocked"`
	NewLetter   string   `json:"newLetter"` // the letter the round opened, "" for none
	Consistency *float64 `json:"consistency"`
}

func (m mode) results(w http.ResponseWriter, r *http.Request, user string) {
	var res trainer.Result
	if !trainer.Read(w, r, m.t.Cfg.MaxBodyBytes, &res) {
		return
	}
	if msg := m.t.Check(&res); msg != "" {
		trainer.Bad(w, msg)
		return
	}
	if res.Kind != "" || res.Level != 0 || res.List != 0 {
		trainer.Bad(w, "kind, level, list: Learn has only the usual round")
		return
	}
	m.t.Write.Lock()
	defer m.t.Write.Unlock()
	again, err := m.t.Store.HasRound(user, res.ID)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	learnt := res.Keys
	if again {
		learnt = nil
	}
	done, err := m.t.Learn(user, learnt, false)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	round := m.t.Round(res, storage.ModeLearn)
	round.Unlocked, round.Lap = done.Progress.Unlocked, done.Lap
	if err := m.t.Store.Save(user, &done.Progress, &round); err != nil {
		trainer.Fail(w, err)
		return
	}
	trainer.JSON(w, Result{Unlocked: done.Progress.Unlocked, NewLetter: done.NewLetter, Consistency: round.Consistency})
}

// Stats is Learn's stats and what the typing screen shows over the text.
type Stats struct {
	Letters    []trainer.Letter    `json:"letters"` // in the order they open
	Unlocked   int                 `json:"unlocked"`
	Mastery    float64             `json:"mastery"` // the accuracy a letter is mastered at, in %
	History    trainer.History     `json:"history"`
	Profile    profile.Profile     `json:"profile"`
	Confusions []history.Confusion `json:"confusions"` // the keys typed instead of others
	Tired      bool                `json:"tired"`
}

func (m mode) stats(w http.ResponseWriter, r *http.Request, user string) {
	p, err := m.t.Store.Progress(user)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	rounds, err := m.t.Store.Rounds(user, storage.ModeLearn, 0)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	open := m.t.Words.Gen.Clamp(p.Unlocked)
	c := m.t.Cfg
	trainer.JSON(w, Stats{
		Letters: m.t.Letters(p, open), Unlocked: open, Mastery: 100 * c.Engine.AccMaster,
		History: trainer.HistoryOf(rounds, ""), Profile: m.t.Profile(rounds),
		Confusions: history.Confusions(rounds, c.ConfusionMin, c.ConfusionShown), Tired: m.t.Tired(rounds),
	})
}

// checkin is Learn's first visit: "have you learned touch typing before?"
// A little or yes opens SomeLetters at once; only while Learn has no rounds,
// and never fewer letters than now.
func (m mode) checkin(w http.ResponseWriter, r *http.Request, user string) {
	var req struct {
		Answer string `json:"answer"` // no, little or yes
	}
	if !trainer.Read(w, r, m.t.Cfg.MaxSmallBodyBytes, &req) {
		return
	}
	if req.Answer != "no" && req.Answer != "little" && req.Answer != "yes" {
		trainer.Bad(w, "answer: no, little or yes")
		return
	}
	m.t.Write.Lock()
	defer m.t.Write.Unlock()
	p, err := m.t.Store.Progress(user)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	rounds, err := m.t.Store.Rounds(user, storage.ModeLearn, 1)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	gen := m.t.Words.Gen
	open := gen.Clamp(p.Unlocked)
	if len(rounds) == 0 && req.Answer != "no" {
		open = min(len(gen.Order()), max(open, m.t.Cfg.Checkin.SomeLetters))
	}
	if open != p.Unlocked {
		p.Unlocked = open
		if err := m.t.Store.Save(user, &p, nil); err != nil {
			trainer.Fail(w, err)
			return
		}
	}
	trainer.JSON(w, map[string]int{"unlocked": open})
}

// reset starts Learn over: its letters and rounds, and Reinforce's, go;
// Freetype stays.
func (m mode) reset(w http.ResponseWriter, r *http.Request, user string) {
	m.t.Write.Lock()
	defer m.t.Write.Unlock()
	if err := m.t.Store.ResetLearn(user); err != nil {
		trainer.Fail(w, err)
		return
	}
	trainer.JSON(w, map[string]bool{"ok": true})
}
