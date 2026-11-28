// Package freetype is the Freetype mode: it trains rhythm. Words drawn from
// the n most common; the letter pairs where the typist's rhythm stumbles come
// more often, and the word list moves by itself with the consistency. It
// shares no data with Learn and Reinforce.
package freetype

import (
	"net/http"
	"slices"
	"strconv"

	"typingtrainer/internal/freetype"
	"typingtrainer/internal/levels"
	"typingtrainer/internal/storage"
	"typingtrainer/internal/trainer"
)

type mode struct{ t *trainer.Trainer }

func Register(mux *http.ServeMux, t *trainer.Trainer) {
	m := mode{t}
	mux.HandleFunc("POST /api/freetype/portion", t.Route(m.portion))
	mux.HandleFunc("POST /api/freetype/results", t.Route(m.results))
	mux.HandleFunc("GET /api/freetype/stats", t.Route(m.stats))
}

// lists is the word lists offered: those the word list is long enough for.
func (m mode) lists() []int {
	out := []int{}
	for _, n := range m.t.Cfg.Freetype.Lists {
		if n <= m.t.Words.Picker.Size() {
			out = append(out, n)
		}
	}
	return out
}

func (m mode) validList(n int) bool { return slices.Contains(m.lists(), n) }

// rounds is the last Freetype rounds of every list, newest first.
func (m mode) rounds(user string) ([]storage.Round, error) {
	return m.t.Store.Rounds(user, storage.FreetypePrefix, max(m.t.Cfg.Freetype.RhythmRounds, m.t.Cfg.Levels.History))
}

type portionRequest struct {
	Recent []string `json:"recent"`
	Start  bool     `json:"start"`
	Words  int      `json:"words"` // with start: enough portions for this many words
	List   int      `json:"list"`
	Kind   string   `json:"kind"`
}

type portion struct {
	Words []string `json:"words"`
	Why   []string `json:"why,omitempty"`
}

func (m mode) portion(w http.ResponseWriter, r *http.Request, user string) {
	var req portionRequest
	if !trainer.Read(w, r, m.t.Cfg.MaxBodyBytes, &req) {
		return
	}
	p := m.t.Cfg.Freetype
	switch {
	case len(req.Recent) > m.t.Cfg.MaxEvents:
		trainer.Bad(w, "recent: too long")
		return
	case req.Words < 0 || req.Words > p.MaxStartWords:
		trainer.Bad(w, "words: out of range")
		return
	case !m.validList(req.List):
		trainer.Bad(w, "list: not offered")
		return
	case !storage.ValidKind(req.Kind):
		trainer.Bad(w, "kind: unknown kind of round")
		return
	}
	rounds, err := m.rounds(user)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	v, _, err := m.t.Vocabulary(user, req.Start)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	spots := trainer.Spots{}
	for _, x := range m.t.Rhythm(rounds).Pairs {
		spots.Stumbles = append(spots.Stumbles, x.Pair)
	}
	switch req.Kind {
	case storage.KindWeak:
		p.StumbleBoost = m.t.Cfg.WeakBoost
	case storage.KindSprint:
		p.OwnedBoost, spots.Owned = m.t.Cfg.WeakBoost, v.Owned
	}
	count := 1
	if req.Start && req.Words > 0 {
		count = (req.Words + p.PortionWords - 1) / p.PortionWords
	}
	picker := m.t.Words.Picker.WithOwned(v.Owned)
	out := make([]portion, count)
	recent := slices.Clone(req.Recent)
	m.t.Draw(func() {
		for i := range out {
			out[i].Words = picker.Freetype(recent, p.PortionWords, req.List, spots.Stumbles, p)
			if len(spots.Stumbles) > 0 || len(spots.Owned) > 0 {
				out[i].Why = spots.Whys(out[i].Words, "")
			}
			recent = append(recent, out[i].Words...)
		}
	})
	trainer.JSON(w, map[string]any{"portions": out})
}

func (m mode) results(w http.ResponseWriter, r *http.Request, user string) {
	var res trainer.Result
	if !trainer.Read(w, r, m.t.Cfg.MaxBodyBytes, &res) {
		return
	}
	res.Samples = nil // Freetype never learns: its presses are only kept
	if msg := m.t.Check(&res); msg != "" {
		trainer.Bad(w, msg)
		return
	}
	if !m.validList(res.List) || res.Level != 0 {
		trainer.Bad(w, "list: not offered (or a level, which Freetype has not)")
		return
	}
	round := m.t.Round(res, storage.FreetypeMode(res.List))
	m.t.Write.Lock()
	defer m.t.Write.Unlock()
	if err := m.t.Store.Save(user, nil, &round); err != nil {
		trainer.Fail(w, err)
		return
	}
	trainer.JSON(w, map[string]any{"consistency": round.Consistency})
}

// Stats is Freetype's stats of one word list and what its bar shows.
type Stats struct {
	Lists     []int            `json:"lists"` // the word lists offered
	List      int              `json:"list"`
	History   trainer.History  `json:"history"` // of this list
	Rhythm    freetype.Rhythm  `json:"rhythm"`
	WeakSpots int              `json:"weakSpots"` // what a weak-spots round would be built from
	Standing  trainer.Standing `json:"standing"`  // where the rounds leave the word list
	Owned     []int            `json:"owned"`     // the words owned in each list
	Tired     bool             `json:"tired"`
}

func (m mode) stats(w http.ResponseWriter, r *http.Request, user string) {
	lists := m.lists()
	list := lists[0]
	if v := r.URL.Query().Get("list"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || !m.validList(n) {
			trainer.Bad(w, "list: not offered")
			return
		}
		list = n
	}
	ofList, err := m.t.Store.Rounds(user, storage.FreetypeMode(list), 0)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	rounds, err := m.rounds(user)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	v, _, err := m.t.Vocabulary(user, true)
	if err != nil {
		trainer.Fail(w, err)
		return
	}
	rhythm := m.t.Rhythm(rounds)
	trainer.JSON(w, Stats{
		Lists: lists, List: list, History: trainer.HistoryOf(ofList, r.URL.Query().Get("length")),
		Rhythm: rhythm, WeakSpots: len(rhythm.Pairs), Standing: m.standing(rounds, lists),
		Owned: m.t.OwnedPerList(v, lists), Tired: m.t.Tired(rounds),
	})
}

// standing judges each round by its consistency (levels.JudgeRhythm), at
// the list it was typed at.
func (m mode) standing(rounds []storage.Round, lists []int) trainer.Standing {
	return m.t.StandingOf(rounds, len(lists), func(r storage.Round) (int, levels.Round) {
		n, _ := storage.FreetypeList(r.Mode)
		return slices.Index(lists, n) + 1, levels.Round{Accuracy: r.Accuracy, WPM: r.WPM, Consistency: r.Consistency}
	}, levels.JudgeRhythm)
}
