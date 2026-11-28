package storage

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"typingtrainer/internal/engine"
)

// The tests every Store passes: run on memory here and on PostgreSQL in
// postgres_test.go.
var storeTests = map[string]func(*testing.T, Store){
	"NewTypist":      testNewTypist,
	"Progress":       testProgress,
	"RoundsOnce":     testRoundsOnce,
	"RoundsByMode":   testRoundsByMode,
	"Reset":          testReset,
	"ResetLearn":     testResetLearn,
	"Import":         testImport,
	"TypistsApart":   testTypistsApart,
	"RoundKeepsKeys": testRoundKeepsKeys,
}

func runStoreTests(t *testing.T, fresh func(*testing.T) Store) {
	for name, test := range storeTests {
		t.Run(name, func(t *testing.T) { test(t, fresh(t)) })
	}
}

func TestMemoryStore(t *testing.T) {
	runStoreTests(t, func(*testing.T) Store { return NewMemoryStore() })
}

const u = "g1"

var at = time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)

func round(mode string, wpm float64) Round {
	return Round{Mode: mode, Length: "words-30", WPM: wpm, Accuracy: 97, Chars: 150, FinishedAt: at}
}

func wpms(rounds []Round) string {
	out := ""
	for _, r := range rounds {
		out += fmt.Sprintf("%v ", r.WPM)
	}
	return out
}

func testNewTypist(t *testing.T, s Store) {
	p, err := s.Progress(u)
	if err != nil || p.Unlocked != 0 || p.Lap != 0 || p.Letters == nil || len(p.Letters) != 0 {
		t.Fatalf("a new typist: %+v %v", p, err)
	}
	if rs, err := s.Rounds(u, "", 0); err != nil || len(rs) != 0 {
		t.Fatalf("a new typist has rounds: %v %v", rs, err)
	}
}

func testProgress(t *testing.T, s Store) {
	want := Progress{Letters: map[string]engine.KeyStat{"a": {Count: 3, ErrEMA: 0.1, Hits: "101", Held: true}}, Unlocked: 7, Lap: 2}
	if err := s.Save(u, &want, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Progress(u)
	if got.Unlocked != 7 || got.Lap != 2 || got.Letters["a"] != want.Letters["a"] {
		t.Fatalf("progress: %+v", got)
	}
	got.Letters["b"] = engine.KeyStat{Count: 1}
	if again, _ := s.Progress(u); len(again.Letters) != 1 {
		t.Fatal("a change to a read progress reached the store")
	}
	r := round(ModeLearn, 30)
	s.Save(u, nil, &r)
	if again, _ := s.Progress(u); again.Unlocked != 7 {
		t.Fatalf("a round alone changed the progress: %+v", again)
	}
}

func testRoundsOnce(t *testing.T, s Store) {
	r := round(ModeLearn, 30)
	r.ID = "r-1"
	s.Save(u, nil, &r)
	r.WPM = 99
	s.Save(u, nil, &r)
	rs, _ := s.Rounds(u, "", 0)
	if len(rs) != 1 || rs[0].WPM != 30 {
		t.Fatalf("a round sent twice: %v", wpms(rs))
	}
	if ok, _ := s.HasRound(u, "r-1"); !ok {
		t.Fatal("HasRound misses a kept round")
	}
	if ok, _ := s.HasRound(u, ""); ok {
		t.Fatal("no id is no round")
	}
	plain := round(ModeLearn, 40)
	s.Save(u, nil, &plain)
	s.Save(u, nil, &plain)
	if rs, _ := s.Rounds(u, "", 0); len(rs) != 3 {
		t.Fatalf("rounds without an id are each kept: %d", len(rs))
	}
}

func testRoundsByMode(t *testing.T, s Store) {
	for i, mode := range []string{ModeLearn, ModeReinforce, FreetypeMode(200), FreetypeMode(1000), ModeLearn} {
		r := round(mode, float64(10*(i+1)))
		s.Save(u, nil, &r)
	}
	for prefix, want := range map[string]string{"": "50 40 30 20 10 ", ModeLearn: "50 10 ", ModeReinforce: "20 ", FreetypePrefix: "40 30 ", FreetypeMode(200): "30 "} {
		if rs, _ := s.Rounds(u, prefix, 0); wpms(rs) != want {
			t.Errorf("%q: %v, want %v", prefix, wpms(rs), want)
		}
	}
	if rs, _ := s.Rounds(u, "", 2); wpms(rs) != "50 40 " {
		t.Errorf("the last two: %v", wpms(rs))
	}
}

func testReset(t *testing.T, s Store) {
	s.Save(u, &Progress{Unlocked: 9}, &Round{Mode: ModeLearn, Length: "words-30", FinishedAt: at})
	if err := s.Reset(u); err != nil {
		t.Fatal(err)
	}
	p, _ := s.Progress(u)
	rs, _ := s.Rounds(u, "", 0)
	if p.Unlocked != 0 || len(rs) != 0 {
		t.Fatalf("after a reset: %+v, %d rounds", p, len(rs))
	}
}

func testResetLearn(t *testing.T, s Store) {
	for _, mode := range []string{ModeLearn, ModeReinforce, FreetypeMode(200)} {
		r := round(mode, 30)
		s.Save(u, &Progress{Unlocked: 9, Lap: 3}, &r)
	}
	s.ResetLearn(u)
	p, _ := s.Progress(u)
	rs, _ := s.Rounds(u, "", 0)
	if p.Unlocked != 0 || p.Lap != 0 || len(rs) != 1 || rs[0].Mode != FreetypeMode(200) {
		t.Fatalf("Learn starts over, Freetype stays: %+v, %v", p, rs)
	}
}

func testImport(t *testing.T, s Store) {
	old := round(ModeLearn, 1)
	s.Save(u, &Progress{Unlocked: 3}, &old)
	in := []Round{round(ModeLearn, 10), round(FreetypeMode(200), 20)}
	in[0].ID, in[1].ID = "a", "a" // the second is the first sent again
	if err := s.Import(u, Progress{Unlocked: 12, Letters: map[string]engine.KeyStat{"e": {Count: 5}}}, in); err != nil {
		t.Fatal(err)
	}
	p, _ := s.Progress(u)
	rs, _ := s.Rounds(u, "", 0)
	if p.Unlocked != 12 || p.Letters["e"].Count != 5 || wpms(rs) != "10 " {
		t.Fatalf("import replaces all: %+v, %v", p, wpms(rs))
	}
}

func testTypistsApart(t *testing.T, s Store) {
	r := round(ModeLearn, 30)
	s.Save(u, &Progress{Unlocked: 8}, &r)
	if p, _ := s.Progress("g2"); p.Unlocked != 0 {
		t.Fatal("another typist sees the progress")
	}
	if rs, _ := s.Rounds("g2", "", 0); len(rs) != 0 {
		t.Fatal("another typist sees the rounds")
	}
	s.Reset("g2")
	if rs, _ := s.Rounds(u, "", 0); len(rs) != 1 {
		t.Fatal("another typist's reset deleted the rounds")
	}
}

func testRoundKeepsKeys(t *testing.T, s Store) {
	cons := 87.5
	r := Round{ID: "k", Mode: ModeReinforce, Kind: KindWeak, Level: 3, Lap: 2, Length: "words-30", WPM: 55.5, Accuracy: 96,
		Chars: 3, Consistency: &cons, Unlocked: 26, FinishedAt: at,
		Keys: []engine.Sample{{Char: "a", Ms: 120}, {Char: "b", Errors: 1, Ms: 300, Typed: "v"}}}
	s.Save(u, nil, &r)
	rs, _ := s.Rounds(u, "", 0)
	got := rs[0]
	if got.ID != "k" || got.Kind != KindWeak || got.Level != 3 || got.Lap != 2 || got.WPM != 55.5 || *got.Consistency != 87.5 ||
		got.Unlocked != 26 || !got.FinishedAt.Equal(at) || !slices.Equal(got.Keys, r.Keys) {
		t.Fatalf("round read back: %+v", got)
	}
}

func TestFreetypeList(t *testing.T) {
	if n, ok := FreetypeList(FreetypeMode(1000)); !ok || n != 1000 {
		t.Errorf("FreetypeList = %d, %v", n, ok)
	}
	for _, bad := range []string{"", ModeLearn, "freetype-", "freetype-0", "freetype-01", "freetype-1000-long"} {
		if _, ok := FreetypeList(bad); ok {
			t.Errorf("%q is no Freetype list", bad)
		}
	}
}

func TestBests(t *testing.T) {
	a, b, c := round(ModeLearn, 30), round(ModeLearn, 50), round(ModeLearn, 90)
	c.Kind = KindSprint
	b.Keys = []engine.Sample{{Char: "a"}}
	bests := Bests([]Round{a, b, c})
	if bests["words-30"].WPM != 50 || bests[KindSprint].WPM != 90 || bests["words-30"].Keys != nil {
		t.Fatalf("bests: %+v", bests)
	}
}
