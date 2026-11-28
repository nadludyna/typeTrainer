// A typist's whole data: export to a file, import from one, reset.

package app

import (
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"typingtrainer/internal/storage"
	"typingtrainer/internal/trainer"
)

// File is everything kept of a typist, as Export writes it.
type File struct {
	Progress storage.Progress `json:"progress"`
	Rounds   []storage.Round  `json:"rounds"` // oldest first
}

// Imported is how many rounds an import kept.
type Imported struct {
	Rounds int `json:"rounds"`
}

func registerData(mux *http.ServeMux, t *trainer.Trainer) {
	mux.HandleFunc("GET /api/export", t.Route(func(w http.ResponseWriter, r *http.Request, user string) {
		p, err := t.Store.Progress(user)
		if err != nil {
			trainer.Fail(w, err)
			return
		}
		rounds, err := t.Store.Rounds(user, "", 0)
		if err != nil {
			trainer.Fail(w, err)
			return
		}
		slices.Reverse(rounds)
		trainer.JSON(w, File{Progress: p, Rounds: rounds})
	}))
	mux.HandleFunc("POST /api/import", t.Route(func(w http.ResponseWriter, r *http.Request, user string) {
		var f File
		if !trainer.Read(w, r, t.Cfg.MaxImportBytes, &f) {
			return
		}
		if msg := check(t, f); msg != "" {
			trainer.Bad(w, msg)
			return
		}
		t.Write.Lock()
		defer t.Write.Unlock()
		if err := t.Store.Import(user, f.Progress, f.Rounds); err != nil {
			trainer.Fail(w, err)
			return
		}
		trainer.JSON(w, Imported{Rounds: len(f.Rounds)})
	}))
	mux.HandleFunc("POST /api/reset", t.Route(func(w http.ResponseWriter, r *http.Request, user string) {
		t.Write.Lock()
		defer t.Write.Unlock()
		if err := t.Store.Reset(user); err != nil {
			trainer.Fail(w, err)
			return
		}
		trainer.Done(w)
	}))
}

// check is what is wrong with an imported file, "" when nothing is: nothing
// impossible or foreign reaches the statistics.
func check(t *trainer.Trainer, f File) string {
	letters, laps, levels := len(t.Words.Gen.Order()), t.LastLap(), len(t.Words.Levels)
	p := f.Progress
	if p.Unlocked < 0 || p.Unlocked > letters || p.Lap < 0 || p.Lap > laps {
		return "progress: unlocked or lap out of range"
	}
	for c, k := range p.Letters {
		if utf8.RuneCountInString(c) != 1 || (c != " " && !strings.Contains(t.Cfg.Alphabet, c)) {
			return "progress: " + c + " is not a letter"
		}
		if k.Count < 0 || k.Mistakes < 0 || k.Mistakes > k.Count || k.TimedCount < 0 || k.TimedCount > k.Count ||
			k.ErrEMA < 0 || k.ErrEMA > 1 || k.TimeEMA < 0 {
			return "progress: impossible numbers for " + c
		}
	}
	if len(f.Rounds) > t.Cfg.MaxImportRounds {
		return "rounds: too many"
	}
	future := time.Now().Add(t.Cfg.ImportFutureTolerance)
	for _, r := range f.Rounds {
		_, freetype := storage.FreetypeList(r.Mode)
		switch {
		case r.Mode != storage.ModeLearn && r.Mode != storage.ModeReinforce && !freetype:
			return "rounds: unknown mode " + r.Mode
		case (r.Mode == storage.ModeReinforce) != (r.Level >= 1 && r.Level <= levels):
			return "rounds: a Reinforce round has a word level, no other one"
		case r.WPM < 0 || r.WPM > 400 || r.Accuracy < 0 || r.Accuracy > 100 || r.Chars < 0:
			return "rounds: wpm, accuracy or chars out of range"
		case r.Consistency != nil && (*r.Consistency < 0 || *r.Consistency > 100):
			return "rounds: consistency out of range"
		case r.Unlocked < 0 || r.Unlocked > letters || r.Lap < 0 || r.Lap > laps:
			return "rounds: unlocked or lap out of range"
		case !storage.ValidKind(r.Kind) || !trainer.ValidID(r.ID):
			return "rounds: unknown kind or a bad id"
		case !slices.Contains(t.Cfg.Lengths, r.Length):
			return "rounds: unknown length " + r.Length
		case r.FinishedAt.After(future):
			return "rounds: finished in the future"
		}
		if msg := t.CheckSamples(r.Keys); msg != "" {
			return "rounds: keys: " + msg
		}
	}
	return ""
}
