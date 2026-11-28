// Package trainer is what the three modes share: the words and the keyboard,
// the store, the letter model of Learn and Reinforce, the weak spots read
// from a typist's rounds, and the HTTP helpers. Each mode is its own package
// under internal/modes and only uses this one.
package trainer

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"typingtrainer/data"
	"typingtrainer/internal/config"
	"typingtrainer/internal/engine"
	"typingtrainer/internal/freetype"
	"typingtrainer/internal/layout"
	"typingtrainer/internal/profile"
	"typingtrainer/internal/storage"
)

// Words is the word list and the keyboard, read once at start.
type Words struct {
	Gen     *engine.Generator   // Learn's words, the most common first
	Levels  []*engine.Generator // Reinforce's word levels
	Picker  *freetype.Picker    // Freetype's words
	Layout  []byte              // the keyboard layout, as the page gets it
	Places  map[string]profile.Place
	Fingers []string // left little to right little
}

type Trainer struct {
	Cfg   config.Config
	Store storage.Store
	Words *Words

	// Write is held around every read-modify-write of a typist's data, so
	// two requests never interleave. Enough for one server process.
	Write sync.Mutex
	gen   sync.Mutex // the random sources are not safe in parallel

	portions portionCache
	vocab    vocabCache
}

func New(cfg config.Config, store storage.Store, words *Words) *Trainer {
	return &Trainer{Cfg: cfg, Store: store, Words: words,
		portions: portionCache{entries: map[string]*cachedPortion{}, ttl: cfg.PortionCacheTTL, cap: cfg.PortionCacheRounds},
		vocab:    vocabCache{entries: map[string]vocabAt{}}}
}

// Draw runs f with the random sources to itself: they are not safe in
// parallel.
func (t *Trainer) Draw(f func()) {
	t.gen.Lock()
	defer t.gen.Unlock()
	f()
}

// Corpus is the word list's words made only of the alphabet's letters, the
// most common first, and the keys under the resting fingers.
func Corpus(cfg config.Config) (words, home []string, err error) {
	lay, err := layout.Parse(data.Layout)
	if err != nil {
		return nil, nil, err
	}
	words = wordsOf(string(data.Words), cfg.Alphabet)
	if len(words) == 0 {
		return nil, nil, fmt.Errorf("the word list is empty")
	}
	return words, lay.RestingKeys(), nil
}

// LoadWords builds the words and the keyboard from the data built in.
func LoadWords(cfg config.Config, seed int64) (*Words, error) {
	words, home, err := Corpus(cfg)
	if err != nil {
		return nil, err
	}
	lay, err := layout.Parse(data.Layout)
	if err != nil {
		return nil, err
	}
	gen := engine.NewGenerator(words, rand.New(rand.NewSource(seed)), cfg.Engine)
	gen.SetHome(home, cfg.Engine)
	gen.SetWordParams(cfg.Words)
	w := &Words{Gen: gen, Layout: data.Layout, Fingers: lay.FingerOrder(), Places: map[string]profile.Place{},
		Picker: freetype.NewPicker(words, rand.New(rand.NewSource(seed+1)))}
	for c, at := range lay.Places() {
		w.Places[c] = profile.Place{Finger: at.Finger, Row: at.Row, X: at.X}
	}
	for _, n := range cfg.WordLevels {
		w.Levels = append(w.Levels, gen.WithWords(words[:min(n, len(words))], cfg.Engine))
	}
	return w, nil
}

// wordsOf is the list's words made only of the alphabet's letters, each once.
func wordsOf(list, alphabet string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, w := range strings.Fields(strings.ToLower(list)) {
		if seen[w] || w == "" || strings.Trim(w, alphabet) != "" {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	return out
}
