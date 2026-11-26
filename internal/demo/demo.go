// Package demo makes a made-up history of every mode, in the format of
// /api/v1/import, to look at the screens with data in them while working on
// them (`go run ./tools/demo > demo.json`, then Settings → Import).
package demo

import (
	"math"
	"math/rand"
	"strings"
	"time"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/storage"
)

// File is what /api/v1/import reads.
type File struct {
	Unlocked int                       `json:"unlocked"`
	Keys     map[string]engine.KeyStat `json:"keys"`
	Sessions []storage.Session         `json:"sessions"`
	Freetype map[string]Rounds         `json:"freetype"`
}

type Rounds struct {
	Sessions []storage.Session `json:"sessions"`
}

const (
	learnRounds     = 37
	reinforceRounds = 21
	freetypeRounds  = 12
	firstLetters    = 6
	lastLetters     = 13
	reinforceLevel  = 2 // the 500 most common words
	freetypeList    = "1000"
	days            = 60
	roundWords      = 30
)

// Build: order is the language's letter order (the first ones open first),
// words its word list, most common first. The same seed gives the same file.
func Build(order, words []string, p engine.Params, now time.Time, seed int64) File {
	rng := rand.New(rand.NewSource(seed))
	f := File{Unlocked: lastLetters, Keys: map[string]engine.KeyStat{}, Freetype: map[string]Rounds{}}
	at := func(i, n int) time.Time { // the rounds spread over the last `days` days, the last one an hour ago
		return now.Add(-time.Hour - time.Duration(float64(n-1-i)/float64(max(n-1, 1))*float64(days)*24)*time.Hour)
	}
	all := learnRounds + reinforceRounds
	for i := 0; i < learnRounds; i++ {
		open := min(lastLetters, firstLetters+i*(lastLetters-firstLetters+1)/learnRounds)
		acc := 0.86 + 0.10*float64(i)/learnRounds + (rng.Float64()-0.5)*0.04
		wpm := 18 + 14*float64(i)/learnRounds + rng.Float64()*3
		f.Sessions = append(f.Sessions, round(rng, pick(words, order[:open], 200), acc, wpm, at(i, all), open, 0))
	}
	for i := 0; i < reinforceRounds; i++ {
		acc := 0.95 + rng.Float64()*0.03
		wpm := 38 + 16*float64(i)/reinforceRounds + (rng.Float64()-0.5)*8
		f.Sessions = append(f.Sessions, round(rng, words[:500], acc, wpm, at(learnRounds+i, all), lastLetters, reinforceLevel))
	}
	var ft []storage.Session
	for i := 0; i < freetypeRounds; i++ {
		s := round(rng, words[:1000], 0.95+rng.Float64()*0.03, 52+rng.Float64()*10, at(i, freetypeRounds), 0, 0)
		cons := 66 + 14*float64(i)/freetypeRounds + (rng.Float64()-0.5)*8
		s.Consistency, s.Unlocked, s.Keys = &cons, 0, nil
		ft = append(ft, s)
	}
	f.Freetype[freetypeList] = Rounds{Sessions: ft}
	for _, s := range f.Sessions {
		f.Keys = engine.Apply(f.Keys, s.Keys, p)
	}
	return f
}

// pick is the words made only of these letters (or the first ones, if too few).
func pick(words, letters []string, n int) []string {
	set := strings.Join(letters, "")
	var out []string
	for _, w := range words {
		if strings.Trim(w, set) == "" {
			out = append(out, w)
			if len(out) == n {
				break
			}
		}
	}
	if len(out) < 10 {
		return words[:n]
	}
	return out
}

// round is one round of roundWords words at about this accuracy and speed,
// with its raw presses: a wrong press names the key typed instead.
func round(rng *rand.Rand, pool []string, acc, wpm float64, at time.Time, open int, level float64) storage.Session {
	ms := 12000 / wpm
	var keys []engine.Sample
	right, chars := 0, 0
	for w := 0; w < roundWords; w++ {
		word := pool[rng.Intn(len(pool))]
		if w > 0 {
			keys = append(keys, engine.Sample{Char: " ", Ms: math.Round(ms)})
			chars++
			right++
		}
		for _, r := range word {
			k := engine.Sample{Char: string(r), Ms: math.Round(ms * (0.6 + rng.Float64()*0.8))}
			if rng.Float64() > acc {
				k.Errors, k.Typed = 1, string(word[rng.Intn(len(word))])
			} else {
				right++
			}
			keys = append(keys, k)
			chars++
		}
	}
	accuracy := math.Round(1000*float64(right)/float64(chars)) / 10
	return storage.Session{
		WPM: math.Round(wpm*10) / 10, Accuracy: accuracy, Chars: chars, FinishedAt: at.UTC(),
		Length: "words-30", Keys: keys, Unlocked: open, Level: level, Seconds: float64(chars) * ms / 1000,
	}
}
