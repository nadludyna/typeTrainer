// Package reinforce is the Reinforce mode: it trains speed. Real words of a
// word level (the 200 to the 25 000 most common), weighted to the slow
// letters and pairs; the speed goal grows in steps, and the word level moves
// by itself with the rounds.
package reinforce

import (
	"net/http"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/levels"
	"typingtrainer/internal/profile"
	"typingtrainer/internal/storage"
	"typingtrainer/internal/trainer"
)

type mode struct{ t *trainer.Trainer }

func Register(mux *http.ServeMux, t *trainer.Trainer) {
	m := mode{t}
	mux.HandleFunc("POST /api/reinforce/portion", t.Route(m.portion))
	mux.HandleFunc("POST /api/reinforce/results", t.Route(m.results))
	mux.HandleFunc("GET /api/reinforce/stats", t.Route(m.stats))
	mux.HandleFunc("POST /api/reinforce/step", t.Route(m.step))
}

func (m mode) validLevel(level int) bool { return level >= 1 && level <= len(m.t.Words.Levels) }

func (m mode) portion(w http.ResponseWriter, r *http.Request, user string) {
	var req trainer.PortionRequest
	if !trainer.Read(w, r, m.t.Cfg.MaxBodyBytes, &req) {
		return
	}
	if msg := m.t.CheckPortion(req); msg != "" {
		trainer.Bad(w, msg)
		return
	}
	if !m.validLevel(req.Level) {
		trainer.Bad(w, "level: a word level, 1 to the number of levels")
		return
	}
	out, err := m.t.LetterPortions(user, req, true)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	trainer.JSON(w, out)
}

// Result is what a finished round changed.
type Result struct {
	NewGoal     float64  `json:"newGoal"` // the speed goal of the step the round reached, 0 for none
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
	if !m.validLevel(res.Level) || res.List != 0 {
		trainer.Bad(w, "level: a word level, 1 to the number of levels; no list")
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
	done, err := m.t.Learn(user, learnt, true)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	round := m.t.Round(res, storage.ModeReinforce)
	round.Level, round.Lap, round.Unlocked = res.Level, done.Lap, done.Progress.Unlocked
	if err := m.t.Store.Save(user, &done.Progress, &round); err != nil {
		trainer.Fail(w, err)
		return
	}
	trainer.JSON(w, Result{NewGoal: done.NewGoal, Consistency: round.Consistency})
}

// Stats is Reinforce's stats and what its bar over the text shows.
type Stats struct {
	Letters    []trainer.Letter `json:"letters"` // every letter, at the current goal
	Goal       float64          `json:"goal"`    // the speed goal, wpm
	Step       int              `json:"step"`
	LastStep   int              `json:"lastStep"`
	WordLevels []int            `json:"wordLevels"` // each level's words: the n most common
	Standing   trainer.Standing `json:"standing"`   // where the rounds leave the word level
	Pairs      []levels.Pair    `json:"pairs"`      // the pairs that slow the last rounds down
	Owned      []int            `json:"owned"`      // the words owned in each level
	WeakSpots  int              `json:"weakSpots"`  // what a weak-spots round would be built from
	History    trainer.History  `json:"history"`
	Profile    profile.Profile  `json:"profile"`
	Tired      bool             `json:"tired"`
}

func (m mode) stats(w http.ResponseWriter, r *http.Request, user string) {
	length := r.URL.Query().Get("length")
	p, err := m.t.Store.Progress(user)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	rounds, err := m.t.Store.Rounds(user, storage.ModeReinforce, 0)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	learning, err := m.t.Rounds(user, trainer.Learning, max(m.t.Cfg.Levels.WeakRounds, m.t.Cfg.Profile.Rounds))
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	v, _, err := m.t.Vocabulary(user, true)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	lap := min(max(p.Lap, 0), m.t.LastLap())
	sizes := m.t.Cfg.WordLevels[:len(m.t.Words.Levels)]
	trainer.JSON(w, Stats{
		Letters: m.t.Letters(p, len(m.t.Words.Gen.Order())), Goal: m.t.Params(lap).TargetWPM, Step: lap, LastStep: m.t.LastLap(),
		WordLevels: sizes, Standing: m.standing(rounds), Pairs: m.t.Pairs(rounds[:min(len(rounds), m.t.Cfg.Levels.WeakRounds)]),
		Owned: m.t.OwnedPerList(v, sizes), WeakSpots: m.t.WeakSpots(learning),
		History: trainer.HistoryOf(rounds, length), Profile: m.t.Profile(learning), Tired: m.t.Tired(rounds),
	})
}

// standing judges each round at the speed goal of its own step: it passes at
// the goal with high accuracy, fails far under it (levels.Judge).
func (m mode) standing(rounds []storage.Round) trainer.Standing {
	return m.t.StandingOf(rounds, len(m.t.Words.Levels), func(r storage.Round) (int, levels.Round) {
		return r.Level, levels.Round{Accuracy: r.Accuracy, WPM: r.WPM, Goal: m.t.Params(r.Lap).TargetWPM}
	}, levels.Judge)
}

// Step is the speed goal step picked and its goal, wpm.
type Step struct {
	Step int     `json:"step"`
	Goal float64 `json:"goal"`
}

// step picks the speed goal step by hand, up or down. Mastery starts over at
// the new goal, as after a step reached by typing.
func (m mode) step(w http.ResponseWriter, r *http.Request, user string) {
	var req struct {
		Step int `json:"step"`
	}
	if !trainer.Read(w, r, m.t.Cfg.MaxSmallBodyBytes, &req) {
		return
	}
	if req.Step < 0 || req.Step > m.t.LastLap() {
		trainer.Bad(w, "step: 0 to the last step")
		return
	}
	m.t.Write.Lock()
	defer m.t.Write.Unlock()
	p, err := m.t.Store.Progress(user)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	if req.Step != p.Lap {
		p.Letters, p.Lap = engine.StartLap(p.Letters), req.Step
		if err := m.t.Store.Save(user, &p, nil); err != nil {
			trainer.Fail(w, err)
			return
		}
	}
	trainer.JSON(w, Step{Step: req.Step, Goal: m.t.Params(req.Step).TargetWPM})
}
