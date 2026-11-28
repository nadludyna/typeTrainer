// Command demo prints a made-up history of every mode as an import file, to
// look at the screens with data in them: go run ./tools/demo > demo.json,
// then Import in the settings. The same seed gives the same file.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"typingtrainer/internal/app"
	"typingtrainer/internal/config"
	"typingtrainer/internal/engine"
	"typingtrainer/internal/storage"
	"typingtrainer/internal/trainer"
)

func main() {
	f, err := build(time.Now(), 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "demo:", err)
		os.Exit(1)
	}
	json.NewEncoder(os.Stdout).Encode(f)
}

const (
	learnRounds     = 37
	reinforceRounds = 21
	freetypeRounds  = 12
	firstLetters    = 6
	lastLetters     = 13
	reinforceLevel  = 2 // the 500 most common words
	freetypeList    = 1000
	days            = 60
	roundWords      = 30
)

func build(now time.Time, seed int64) (app.File, error) {
	cfg := config.Default()
	w, err := trainer.LoadWords(cfg, seed)
	if err != nil {
		return app.File{}, err
	}
	order, words := w.Gen.Order(), w.Picker.Top(25000)
	rng := rand.New(rand.NewSource(seed))
	f := app.File{Progress: storage.Progress{Unlocked: lastLetters, Letters: map[string]engine.KeyStat{}}}
	at := func(i, n int) time.Time { // spread over the last `days` days, the last an hour ago
		return now.Add(-time.Hour - time.Duration(float64(n-1-i)/float64(max(n-1, 1))*float64(days)*24)*time.Hour).UTC()
	}
	all := learnRounds + reinforceRounds
	for i := range learnRounds {
		open := min(lastLetters, firstLetters+i*(lastLetters-firstLetters+1)/learnRounds)
		acc := 0.86 + 0.10*float64(i)/learnRounds + (rng.Float64()-0.5)*0.04
		wpm := 18 + 14*float64(i)/learnRounds + rng.Float64()*3
		r := round(rng, pick(words, order[:open], 200), acc, wpm, at(i, all), storage.ModeLearn)
		r.Unlocked = open
		f.Rounds = append(f.Rounds, r)
	}
	for i := range reinforceRounds {
		acc := 0.95 + rng.Float64()*0.03
		wpm := 38 + 16*float64(i)/reinforceRounds + (rng.Float64()-0.5)*8
		r := round(rng, words[:500], acc, wpm, at(learnRounds+i, all), storage.ModeReinforce)
		r.Unlocked, r.Level = lastLetters, reinforceLevel
		f.Rounds = append(f.Rounds, r)
	}
	for _, r := range f.Rounds {
		f.Progress.Letters = engine.Apply(f.Progress.Letters, r.Keys, cfg.Engine)
	}
	for i := range freetypeRounds {
		r := round(rng, words[:freetypeList], 0.95+rng.Float64()*0.03, 52+rng.Float64()*10, at(i, freetypeRounds), storage.FreetypeMode(freetypeList))
		cons := 66 + 14*float64(i)/freetypeRounds + (rng.Float64()-0.5)*8
		r.Consistency = &cons
		f.Rounds = append(f.Rounds, r)
	}
	return f, nil
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
func round(rng *rand.Rand, pool []string, acc, wpm float64, at time.Time, mode string) storage.Round {
	ms := 12000 / wpm
	var keys []engine.Sample
	right, chars := 0, 0
	for w := range roundWords {
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
	return storage.Round{Mode: mode, Length: "words-30", WPM: math.Round(wpm*10) / 10,
		Accuracy: math.Round(1000*float64(right)/float64(chars)) / 10, Chars: chars, FinishedAt: at, Keys: keys}
}
