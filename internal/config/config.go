// Package config holds every tunable number.
package config

import (
	"time"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/freetype"
	"typingtrainer/internal/levels"
	"typingtrainer/internal/profile"
	"typingtrainer/internal/sim"
)

type Config struct {
	Addr      string
	Server    Server
	MaxEvents int

	PortionCacheTTL    time.Duration
	PortionCacheRounds int

	MaxBodyBytes      int
	MaxSmallBodyBytes int
	MaxRoundChars     int

	MaxImportBytes  int
	MaxImportRounds int

	ImportFutureTolerance time.Duration

	Alphabet string   // the letters of the language
	Lengths  []string // the round lengths a round may have

	ConsistencyMinSeconds int
	ConfusionMin          int
	ConfusionShown        int
	Guests                bool // a guest cookie per browser; off on GitHub Pages, where one guest is all
	GuestDays             int  // how long a guest cookie lasts
	Engine                engine.Params
	Laps                  engine.LapParams
	Words                 engine.WordParams
	Freetype              freetype.Params
	WordLevels            []int // Reinforce's word levels: each takes the n most common words
	Levels                levels.Params
	WeakBoost             float64 // in a weak-spots round a word with a weak spot, in a sprint an owned word, weighs 1 + this
	Profile               profile.Params
	Owned                 profile.WordParams // the words a typist owns
	Time                  profile.TimeParams // forgetting, warm-up, tiredness
	Checkin               Checkin
	Sim                   sim.Params
}

// Checkin is Learn's first visit: one question sets the first letters.
type Checkin struct {
	SomeLetters int // Learn's start for "a little" and "yes"
}

type Server struct {
	ReadHeader time.Duration
	Read       time.Duration
	Write      time.Duration
	Idle       time.Duration
	Shutdown   time.Duration
}

func Default() Config {
	return Config{
		Addr: "localhost:3000",
		Server: Server{
			ReadHeader: 10 * time.Second,
			Read:       2 * time.Minute,
			Write:      2 * time.Minute,
			Idle:       2 * time.Minute,
			Shutdown:   20 * time.Second,
		},
		MaxEvents: 5000,

		PortionCacheTTL:    10 * time.Minute,
		PortionCacheRounds: 256,

		MaxBodyBytes:      1 << 20,
		MaxSmallBodyBytes: 1024,
		MaxRoundChars:     1 << 20,

		MaxImportBytes:  64 << 20,
		MaxImportRounds: 50000,

		ImportFutureTolerance: 5 * time.Minute,

		Alphabet: "abcdefghijklmnopqrstuvwxyz",
		Lengths:  []string{"words-10", "words-15", "words-20", "words-30", "words-50", "words-100", "time-15", "time-30", "time-60"},

		ConsistencyMinSeconds: 5,
		ConfusionMin:          3,
		Laps:                  engine.LapParams{StepWPM: 10, MaxWPM: 200},
		Checkin:               Checkin{SomeLetters: 10},
		ConfusionShown:        6,
		GuestDays:             365,
		Engine: engine.Params{
			Alpha:         0.1,
			ErrAlpha:      0.05,
			MaxIntervalMs: 2000,

			TargetWPM:    30,
			SlowMs:       1000,
			MaxErrorRate: 0.25,
			ErrorWeight:  0.6,
			SpeedWeight:  0.4,

			PriorDifficulty: 0.7,
			PriorStrength:   5,

			Temperature: 0.1,

			MasteryMinSamples: 40,
			MasteryLevel:      80,
			UnmasteredBonus:   0.2,

			StartLetters:   6,
			MinWords:       40,
			RepeatPenalty:  0.3,
			MinWordLetters: 2,
			MaxRepeats:     2,
			ExerciseWords:  20,

			PortionWords:  6,
			AheadPortions: 2,
			RecentWords:   24,
			RetryWords:    2,
			RoundPortions: 5,

			RankStart:   200,
			RankStep:    200,
			RankPenalty: 0.5,

			PairBoost:   2,
			FingerShare: 0.3, FingerBoost: 2, OwnedPenalty: 0.3,

			AccWindow: 100,
			AccPrior:  3,
			AccMaster: 0.90,
			AccSlip:   0.85,
		},
		Words: engine.WordParams{
			FloorWords:  10,
			RampLetters: 10000,
		},
		Freetype: freetype.Params{
			PortionWords:  6,
			MaxStartWords: 100,
			RecentWords:   20,
			RepeatPenalty: 0.3,
			OwnedPenalty:  0.3,

			Lists: []int{200, 1000, 5000, 10000, 25000},

			RhythmRounds:      20,
			RhythmMaxMs:       2000,
			RhythmMinPresses:  50,
			StumbleSpreads:    3,
			StumbleLeast:      1.5,
			StumblePrior:      3,
			StumbleMinPresses: 8,
			StumbleMinCount:   2,
			StumbleRatio:      2,
			StumblePairs:      3,
			StumbleBoost:      2,
		},
		Profile: profile.Params{
			Rounds: 20, MinPresses: 50, WeakRatio: 1.5, WeakMargin: 0.02, Prior: 3,
			MinMistakes: 10, NeighbourDist: 1.1,
		},
		Owned: profile.WordParams{Rounds: 60, OwnHits: 3, Slack: 1.1},
		Time: profile.TimeParams{
			Forget: 5 * 24 * time.Hour, Returning: 2,
			Sitting: 30 * time.Minute, TiredRuns: 3, TiredDrop: 3,
			ColdAfter: 6 * time.Hour,
		},
		WeakBoost:  50,
		WordLevels: []int{200, 500, 1000, 2000, 3000, 5000, 7500, 10000, 15000, 25000},
		Levels: levels.Params{
			LevelStart:   1,
			PassAccuracy: 96,
			FailAccuracy: 92,
			FailSpeed:    0.9,
			ConsPass:     80,
			ConsFail:     70,
			PassesUp:     3,
			FailsDown:    2,
			History:      50,

			WeakRounds:     20,
			WeakMinSamples: 15,
			WeakMissMs:     1000,
			WeakPairs:      3,
		},
		Sim: sim.Params{
			ErrMin: 0.03, ErrMax: 0.30,
			TimeMin: 350, TimeMax: 900,
			ErrFloor:  0.01,
			TimeFloor: 200,
			LearnRate: 0.02,
			TimeNoise: 0.25,

			GoalErrorRate: 0.05,

			ExpertErrMax:   0.05,
			ExpertTimeMin:  150,
			ExpertTimeMax:  350,
			OpenExercises:  100,
			FirstExercises: 30,

			Runs:         20,
			MaxExercises: 3000,

			BestOf:        20,
			MissPenaltyMs: 5000,
			ScoreWindow:   50,
		},
	}
}
