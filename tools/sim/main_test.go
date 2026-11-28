package main

// Learn's adaptive word choice must beat random open words.

import (
	"math/rand"
	"strconv"
	"testing"

	"typingtrainer/internal/config"
	"typingtrainer/internal/engine"
	"typingtrainer/internal/sim"
	"typingtrainer/internal/trainer"
)

const minGain = 3

const typists = 3

const (
	kRandom = 0
	kStream = 3
)

func TestAdaptiveBeatsRandom(t *testing.T) {
	cfg := config.Default()
	words, home, err := trainer.Corpus(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []struct{ ID string }{{"en"}} {
		t.Run(l.ID, func(t *testing.T) {
			t.Parallel()

			for seed := int64(1); seed <= typists; seed++ {
				stream := runOne(kStream, seed, words, home, cfg)
				random := runOne(kRandom, seed, words, home, cfg)
				t.Logf("%s, typist %d: stream %d exercises, random %s", l.ID, seed, stream.Exercises, exercisesText(random, cfg))
				if !stream.Reached {
					t.Errorf("typist %d: the stream did not reach the goal in %d exercises", seed, cfg.Sim.MaxExercises)
					continue
				}
				if got := random.Exercises; got < minGain*stream.Exercises {
					t.Errorf("typist %d: random words needed %d exercises, the stream %d: less than %d times more",
						seed, got, stream.Exercises, minGain)
				}
			}
		})
	}
}

func runOne(k int, seed int64, words, home []string, cfg config.Config) sim.Result {
	ep := engineParams(k, cfg.Engine.Temperature, cfg)
	gen := engine.NewGenerator(words, rand.New(rand.NewSource(seed)), ep)
	gen.SetHome(home, ep)
	gen.SetWordParams(cfg.Words)
	typist := sim.NewTypist(gen.Order(), cfg.Sim, rand.New(rand.NewSource(seed))).
		Clone(rand.New(rand.NewSource(seed + 1000)))
	s := strategy(k, cfg.Engine.Temperature, gen, cfg, rand.New(rand.NewSource(seed+2000)))
	return sim.Run(gen, s, typist, ep, cfg.Sim)
}

func exercisesText(r sim.Result, cfg config.Config) string {
	if !r.Reached {
		return "did not reach the goal in " + strconv.Itoa(cfg.Sim.MaxExercises) + " exercises"
	}
	return strconv.Itoa(r.Exercises) + " exercises"
}
