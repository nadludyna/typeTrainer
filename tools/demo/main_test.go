package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"typingtrainer/internal/app"
	"typingtrainer/internal/storage"
)

// The demo file is one the server takes, with every mode in it.
func TestDemoImports(t *testing.T) {
	f, err := build(time.Now(), 1)
	if err != nil {
		t.Fatal(err)
	}
	h, _, err := app.New(storage.NewMemoryStore(), false)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(f)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/import", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("import: %d %s", rec.Code, rec.Body)
	}
	for _, path := range []string{"/api/learn/stats", "/api/reinforce/stats", "/api/freetype/stats?list=1000"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		var st struct {
			History struct{ Count int } `json:"history"`
		}
		json.Unmarshal(rec.Body.Bytes(), &st)
		if rec.Code != http.StatusOK || st.History.Count == 0 {
			t.Errorf("%s: %d, %d rounds", path, rec.Code, st.History.Count)
		}
	}
}
