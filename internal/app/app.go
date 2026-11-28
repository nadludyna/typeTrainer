// Package app puts the program together: the words, the store, the three
// modes and the page. The server and the GitHub Pages build both start here.
package app

import (
	"io/fs"
	"net/http"
	"time"

	"typingtrainer/internal/config"
	"typingtrainer/internal/modes/freetype"
	"typingtrainer/internal/modes/learn"
	"typingtrainer/internal/modes/reinforce"
	"typingtrainer/internal/static"
	"typingtrainer/internal/storage"
	"typingtrainer/internal/trainer"
	"typingtrainer/web"
)

// New is the whole program as one handler. guests: a guest cookie per
// browser (the server); without, one typist (GitHub Pages).
func New(store storage.Store, guests bool) (http.Handler, config.Config, error) {
	cfg := config.Default()
	cfg.Guests = guests
	words, err := trainer.LoadWords(cfg, time.Now().UnixNano())
	if err != nil {
		return nil, cfg, err
	}
	t := trainer.New(cfg, store, words)
	mux := http.NewServeMux()
	learn.Register(mux, t)
	reinforce.Register(mux, t)
	freetype.Register(mux, t)
	registerData(mux, t)
	mux.HandleFunc("GET /api/layout", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(words.Layout)
	})
	mux.Handle("/", page(web.Files()))
	return t.WithGuest(mux), cfg, nil
}

// page serves the built page, or says how to build it.
func page(files fs.FS) http.Handler {
	if _, err := fs.Stat(files, "index.html"); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("The page is not built: run npm run build in web/, then start the server again.\n"))
		})
	}
	return static.New(files)
}
