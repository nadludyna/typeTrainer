// The hidden tuning panel: a user's own values for Learn's main numbers.

package api

import (
	"encoding/json"
	"net/http"

	"typingtrainer/internal/engine"
)

type tunable struct {
	Name     string  `json:"name"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Step     float64 `json:"step"`
	get      func(engine.Params) float64
	set      func(*engine.Params, float64)
	Value    float64 `json:"value"`
	Default  float64 `json:"default"`
	Override bool    `json:"override"`
}

// The panel shows memories in presses and the focus as a push; the engine
// keeps them as smoothing factors and a temperature.
func memory(alpha float64) float64    { return 2/alpha - 1 }
func alphaOf(presses float64) float64 { return 2 / (presses + 1) }

func tunables() []tunable {
	return []tunable{
		{Name: "TargetWPM", Min: 10, Max: 150, Step: 5,
			get: func(p engine.Params) float64 { return p.TargetWPM }, set: func(p *engine.Params, v float64) { p.TargetWPM = v }},
		{Name: "MasteryLevel", Min: 50, Max: 98, Step: 1,
			get: func(p engine.Params) float64 { return p.MasteryLevel }, set: func(p *engine.Params, v float64) { p.MasteryLevel = v }},
		{Name: "MasteryMinSamples", Min: 10, Max: 150, Step: 5,
			get: func(p engine.Params) float64 { return float64(p.MasteryMinSamples) }, set: func(p *engine.Params, v float64) { p.MasteryMinSamples = int(v) }},
		{Name: "ErrMemory", Min: 10, Max: 120, Step: 1,
			get: func(p engine.Params) float64 { return memory(p.ErrAlpha) }, set: func(p *engine.Params, v float64) { p.ErrAlpha = alphaOf(v) }},
		{Name: "TimeMemory", Min: 5, Max: 60, Step: 1,
			get: func(p engine.Params) float64 { return memory(p.Alpha) }, set: func(p *engine.Params, v float64) { p.Alpha = alphaOf(v) }},
		{Name: "SpeedWeight", Min: 0, Max: 1, Step: 0.05,
			get: func(p engine.Params) float64 { return p.SpeedWeight },
			set: func(p *engine.Params, v float64) { p.SpeedWeight, p.ErrorWeight = v, 1-v }},
		{Name: "Push", Min: 2, Max: 30, Step: 1,
			get: func(p engine.Params) float64 { return 1 / p.Temperature }, set: func(p *engine.Params, v float64) { p.Temperature = 1 / v }},
		{Name: "RankPenalty", Min: 0.1, Max: 1, Step: 0.05,
			get: func(p engine.Params) float64 { return p.RankPenalty }, set: func(p *engine.Params, v float64) { p.RankPenalty = v }},
	}
}

// params is Learn's numbers for this user: the config with the panel's values
// on top. The values live in the server's memory only; the browser keeps a
// copy and sends it again after a restart.
func (s *Server) params(user string) engine.Params {
	p := s.cfg.Engine
	s.tuneMu.Lock()
	own := s.tune[user]
	s.tuneMu.Unlock()
	if len(own) == 0 {
		return p
	}
	for _, t := range tunables() {
		if v, ok := own[t.Name]; ok {
			t.set(&p, v)
		}
	}
	return p
}

func (s *Server) tuneList(user string) []tunable {
	p := s.params(user)
	s.tuneMu.Lock()
	own := s.tune[user]
	s.tuneMu.Unlock()
	out := tunables()
	for i := range out {
		out[i].Value = round3(out[i].get(p))
		out[i].Default = round3(out[i].get(s.cfg.Engine))
		_, out[i].Override = own[out[i].Name]
	}
	return out
}

func round3(v float64) float64 { return float64(int64(v*1000+0.5)) / 1000 }

func (s *Server) handleTuneGet(w http.ResponseWriter, r *http.Request, user string, l *Language) {
	writeJSON(w, http.StatusOK, map[string]any{"params": s.tuneList(user)})
}

// POST {values: {name: value}}: the whole set of the user's own values;
// an empty set goes back to the defaults.
func (s *Server) handleTuneSet(w http.ResponseWriter, r *http.Request, user string, l *Language) {
	var req struct {
		Values map[string]float64 `json:"values"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, int64(s.cfg.MaxSmallBodyBytes))).Decode(&req); err != nil {
		badRequest(w, "invalid JSON")
		return
	}
	known := map[string]tunable{}
	for _, t := range tunables() {
		known[t.Name] = t
	}
	own := map[string]float64{}
	for name, v := range req.Values {
		t, ok := known[name]
		if !ok {
			badRequest(w, "unknown value: "+name)
			return
		}
		if !inRange(v, t.Min, t.Max) {
			badRequest(w, name+": out of range")
			return
		}
		own[name] = v
	}
	s.tuneMu.Lock()
	if len(own) == 0 {
		delete(s.tune, user)
	} else {
		s.tune[user] = own
	}
	s.tuneMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"params": s.tuneList(user)})
}
