// Freetype routes.

package api

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"time"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/freetype"
	"typingtrainer/internal/levels"
	"typingtrainer/internal/profile"
	"typingtrainer/internal/storage"
)

// ---------- Freetype's loop: the rhythm and the word list ----------

// typedRounds is the last Freetype rounds, newest first.
func (s *Server) typedRounds(user, lang string) ([]storage.Session, error) {
	rounds, err := s.store.LastFreetypeRounds(user, lang, max(s.cfg.Freetype.RhythmRounds, s.cfg.Levels.History))
	return rounds, err
}

// rhythmOf is the rhythm of the last WeakRounds of these rounds.
func (s *Server) rhythmOf(rounds []storage.Session) freetype.Rhythm {
	rounds = rounds[:min(len(rounds), s.cfg.Freetype.RhythmRounds)]
	keys := make([][]freetype.Key, len(rounds))
	for i, r := range rounds {
		keys[i] = make([]freetype.Key, len(r.Keys))
		for j, k := range r.Keys {
			keys[i][j] = freetype.Key{Char: k.Char, Errors: k.Errors, Ms: k.Ms}
		}
	}
	return freetype.ReadRhythm(keys, s.cfg.Freetype)
}

// listStanding is where the rounds of one variant (newest first) leave its
// word list: each judged by its consistency (levels.JudgeRhythm), the last
// run at one list counted, as Reinforce's word level is.
func (s *Server) listStanding(rounds []storage.Session, lists []int) standing {
	p := s.cfg.Levels
	p.Words = lists
	var rs []levels.Round
	var ends []time.Time
	for i := len(rounds) - 1; i >= 0; i-- {
		ends = append(ends, rounds[i].FinishedAt)
	}
	cold := profile.Cold(ends, s.cfg.Time)
	for i := len(rounds) - 1; i >= 0; i-- {
		n, ok := storage.ParseFreetypeMode(rounds[i].Mode)
		at := slices.Index(lists, n)
		if !ok || at < 0 || rounds[i].Kind != "" {
			continue
		}
		rs = append(rs, levels.Round{Level: at + 1, Accuracy: rounds[i].Accuracy, WPM: rounds[i].WPM, Consistency: rounds[i].Consistency, Cold: cold[len(rounds)-1-i]})
	}
	return standing{Place: levels.Standing(rs, levels.JudgeRhythm, p), PassesUp: p.PassesUp, FailsDown: p.FailsDown}
}

func (s *Server) freetypeLists(l *Language) []int {
	out := []int{}
	for _, n := range s.cfg.Freetype.Lists {
		if n <= l.Freetype.Size() {
			out = append(out, n)
		}
	}
	return out
}

func (s *Server) validList(l *Language, n int) bool {
	for _, m := range s.freetypeLists(l) {
		if m == n {
			return true
		}
	}
	return false
}

// freetypeKeys is the export's keys of Freetype's histories, one per list.
func (s *Server) freetypeKeys(l *Language) []string {
	out := []string{}
	for _, n := range s.freetypeLists(l) {
		out = append(out, strconv.Itoa(n))
	}
	return out
}

func (s *Server) handleFreetypePortion(w http.ResponseWriter, r *http.Request, user string, l *Language) {
	var req struct {
		Recent []string `json:"recent"`
		Start  bool     `json:"start"`
		Words  int      `json:"words"`
		List   int      `json:"list"`
		Kind   string   `json:"kind"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, int64(s.cfg.MaxBodyBytes))).Decode(&req); err != nil {
		badRequest(w, "invalid JSON")
		return
	}
	if len(req.Recent) > s.cfg.MaxEvents {
		badRequest(w, "recent: too long")
		return
	}
	if !s.validStartWords(req.Words) {
		badRequest(w, "words: out of range")
		return
	}
	if !s.validList(l, req.List) {
		badRequest(w, "list: not offered")
		return
	}
	p := s.cfg.Freetype
	rounds, err := s.typedRounds(user, l.Info.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	var stumbles []string
	for _, x := range s.rhythmOf(rounds).Pairs {
		stumbles = append(stumbles, x.Pair)
	}
	if req.Kind == storage.KindWeak {
		p.StumbleBoost = s.cfg.WeakBoost
	}
	if req.Kind == storage.KindSprint {
		p.OwnedBoost = s.cfg.WeakBoost
	}
	spots := weakSpots{stumbles: stumbles}
	v, _, err := s.vocabulary(user, l.Info.ID, req.Start)
	if err != nil {
		serverError(w, err)
		return
	}
	picker := l.Freetype.WithOwned(v.Owned)
	if req.Kind == storage.KindSprint {
		spots.owned = v.Owned
	}
	count := s.startPortions(req.Start, req.Words)
	type portion struct {
		Words []string `json:"words"`
		Why   []string `json:"why,omitempty"`
	}
	out := make([]portion, count)
	recent := append([]string{}, req.Recent...)
	s.genMu.Lock()
	for i := range out {
		out[i].Words = picker.Freetype(recent, p.PortionWords, req.List, stumbles, p)
		recent = append(recent, out[i].Words...)
		if len(stumbles) > 0 || len(spots.owned) > 0 {
			out[i].Why = spots.whys(out[i].Words, "")
		}
	}
	s.genMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"portions": out})
}

func (s *Server) handleFreetypeResults(w http.ResponseWriter, r *http.Request, user string, l *Language) {
	var req resultsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, int64(s.cfg.MaxBodyBytes))).Decode(&req); err != nil {
		badRequest(w, "invalid JSON")
		return
	}
	req.Samples = nil
	if msg := validate(req, s.cfg.MaxRoundChars, s.cfg.MaxEvents, l.Info.Alphabet); msg != "" {
		badRequest(w, msg)
		return
	}
	if !s.validLength(req.Length) {
		badRequest(w, "length: unknown round length")
		return
	}
	if !s.validList(l, req.List) || req.Level != 0 {
		badRequest(w, "list: not offered (or a level, which Freetype has not)")
		return
	}
	var consistency *float64
	if v, ok := engine.Consistency(req.PerSecond, s.cfg.ConsistencyMinSeconds); ok {
		consistency = &v
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	err := s.store.AddFreetypeSession(user, l.Info.ID, storage.FreetypeMode(req.List), storage.Session{
		WPM: req.WPM, Accuracy: req.Accuracy, Chars: req.Chars, Consistency: consistency, FinishedAt: time.Now(),
		Keys:   req.Keys,
		Length: req.Length,
		Kind:   req.Kind,
		Round:  req.Round,
	})
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"consistency": consistency})
}

func (s *Server) handleFreetypeStats(w http.ResponseWriter, r *http.Request, user string, l *Language) {
	length, ok := s.lengthParam(w, r)
	if !ok {
		return
	}
	lists := s.freetypeLists(l)
	list := 0
	if len(lists) > 0 {
		list = lists[0]
	}
	if v := r.URL.Query().Get("list"); v != "" {
		var err error
		if list, err = strconv.Atoi(v); err != nil || !s.validList(l, list) {
			badRequest(w, "list: not offered")
			return
		}
	}
	out, err := s.history(user, l.Info.ID, storage.FreetypeMode(list), length)
	if err != nil {
		serverError(w, err)
		return
	}
	out["lists"], out["list"] = lists, list
	rounds, err := s.typedRounds(user, l.Info.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	rhythm := s.rhythmOf(rounds)
	out["rhythm"], out["weakSpots"] = rhythm, len(rhythm.Pairs)
	oldest := slices.Clone(rounds)
	slices.Reverse(oldest)
	out["tired"] = s.tired(oldest)
	out["standing"] = s.listStanding(rounds, lists)
	v, _, err := s.vocabulary(user, l.Info.ID, true)
	if err != nil {
		serverError(w, err)
		return
	}
	out["owned"] = ownedPerList(v, l, lists)
	writeJSON(w, http.StatusOK, out)
}
