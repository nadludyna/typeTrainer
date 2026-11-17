package engine

import (
	"strings"
	"testing"
)

func accParams() Params {
	p := testParams
	p.AccWindow, p.AccPrior, p.AccMaster, p.AccSlip = 100, 3, 0.90, 0.85
	return p
}

func presses(s KeyStat, p Params, ok, miss int) KeyStat {
	for i := 0; i < miss; i++ {
		s = Update(s, Sample{Char: "a", Errors: 1}, p)
	}
	for i := 0; i < ok; i++ {
		s = Update(s, Sample{Char: "a"}, p)
	}
	return s
}

func TestAccuracy_NoDataIsZero(t *testing.T) {
	if a := Accuracy(KeyStat{}, accParams()); a != 0 {
		t.Fatalf("accuracy of an unseen letter = %v, want 0", a)
	}
	p := accParams()
	p.AccPrior = 0
	if a := Accuracy(KeyStat{}, p); a != 0 {
		t.Fatalf("no prior, no data: %v, want 0", a)
	}
}

func TestAccuracy_PressesToMastery(t *testing.T) {
	p := accParams()
	for _, c := range []struct{ errors, presses int }{{0, 27}, {1, 37}, {2, 47}, {5, 77}} {
		s := presses(KeyStat{}, p, c.presses-c.errors, c.errors)
		if !Mastered(s, p) {
			t.Errorf("%d errors in %d presses: %.4f, want mastered", c.errors, c.presses, Accuracy(s, p))
		}
		s = presses(KeyStat{}, p, c.presses-c.errors-1, c.errors)
		if Mastered(s, p) {
			t.Errorf("%d errors in %d presses: mastered one press early", c.errors, c.presses-1)
		}
	}
}

func TestAccuracy_WindowKeepsTheLastPresses(t *testing.T) {
	p := accParams()
	s := presses(KeyStat{}, p, 150, 50)
	if len(s.Hits) != 100 || strings.Contains(s.Hits, "0") {
		t.Fatalf("window = %d presses with misses %v, want the last 100 clean", len(s.Hits), strings.Contains(s.Hits, "0"))
	}
	if a := Accuracy(s, p); !near(a, 100.0/103) {
		t.Fatalf("best accuracy = %v, want 100/103", a)
	}
}

func TestAccuracy_RulesOffKeepTheOldModel(t *testing.T) {
	s := presses(KeyStat{}, testParams, 30, 0)
	if s.Hits != "" {
		t.Fatalf("hits recorded with the rules off: %q", s.Hits)
	}
}

func TestApply_SlipNeedsBothThresholds(t *testing.T) {
	p := accParams()
	clean := func(n int) []Sample {
		out := make([]Sample, n)
		for i := range out {
			out[i] = Sample{Char: "a"}
		}
		return out
	}
	st := map[string]KeyStat{"a": presses(KeyStat{}, p, 88, 12)} // 88/103 ≈ 85.4 %
	st = Apply(st, nil, p)
	st = Apply(st, clean(0), p)
	if st["a"].Slipped {
		t.Fatal("slipped over 85 %")
	}
	st = map[string]KeyStat{"a": presses(KeyStat{}, p, 85, 14)}
	st = Apply(st, clean(1), p) // 86/103 ≈ 83.5 %
	if !st["a"].Slipped {
		t.Fatalf("not slipped at %.3f", Accuracy(st["a"], p))
	}
	st = Apply(st, clean(4), p) // each clean press pushes a miss out: 90/103 ≈ 87.4 %
	if a := Accuracy(st["a"], p); a < p.AccSlip || a >= p.AccMaster || !st["a"].Slipped {
		t.Fatalf("at %.3f slipped = %v, want slipped between the thresholds", a, st["a"].Slipped)
	}
	st = Apply(st, clean(3), p) // 93/103 ≈ 90.3 %
	if st["a"].Slipped {
		t.Fatalf("still slipped at %.3f", Accuracy(st["a"], p))
	}
}

func accGen(t *testing.T) (*Generator, Params) {
	p := accParams()
	g := NewGenerator(testWords, nil, p)
	if g.Start() >= len(g.Order()) {
		t.Skip("test list opens every letter at once")
	}
	return g, p
}

func mastered(p Params) KeyStat { return presses(KeyStat{}, p, 100, 0) }

func TestNextUnlocked_AccuracyRules(t *testing.T) {
	g, p := accGen(t)
	n := g.Start() + 1
	open := g.Order()[:n]
	st := map[string]KeyStat{}
	if got := g.NextUnlocked(st, n, p); got != n {
		t.Fatalf("no data: opened to %d", got)
	}
	for _, c := range open {
		st[c] = mastered(p)
	}
	if got := g.NextUnlocked(st, n, p); got != n+1 {
		t.Fatalf("all mastered: %d, want %d", got, n+1)
	}
	old := open[0]
	st[old] = presses(KeyStat{}, p, 88, 12) // 88/103 ≈ 85.4 %: under 90, over 85
	if got := g.NextUnlocked(st, n, p); got != n+1 {
		t.Fatalf("old letter between the thresholds held the next letter back")
	}
	st[old] = presses(KeyStat{}, p, 80, 20)
	if got := g.NextUnlocked(st, n, p); got != n {
		t.Fatalf("old letter under 85 %% did not hold the next letter back")
	}
	st[old] = mastered(p)
	st[open[n-1]] = presses(KeyStat{}, p, 20, 0)
	if got := g.NextUnlocked(st, n, p); got != n {
		t.Fatalf("newest letter under 90 %% opened the next one")
	}
}

func TestFocusLetter(t *testing.T) {
	g, p := accGen(t)
	start := g.Start()
	first := g.Order()[:start]
	st := map[string]KeyStat{}
	for _, c := range first {
		st[c] = mastered(p)
	}
	st[first[2]] = presses(KeyStat{}, p, 10, 0)
	if f := g.FocusLetter(st, start, p); f != first[2] {
		t.Fatalf("first lesson focus %q, want the lowest %q", f, first[2])
	}
	n := start + 1
	newest := g.Order()[n-1]
	st[first[2]] = mastered(p)
	if f := g.FocusLetter(st, n, p); f != newest {
		t.Fatalf("focus %q, want the newest %q", f, newest)
	}
	st[first[0]] = presses(KeyStat{}, p, 80, 20)
	st[first[1]] = presses(KeyStat{}, p, 70, 30)
	if f := g.FocusLetter(st, n, p); f != first[1] {
		t.Fatalf("focus %q, want the lowest slipped %q", f, first[1])
	}
}

func TestAccuracy_LetterTypedBeforeTheRules(t *testing.T) {
	p := accParams()
	old := KeyStat{Count: 50, ErrEMA: 0.1}
	if a := Accuracy(old, p); !near(a, 45.0/53) {
		t.Fatalf("seeded accuracy %v, want 45/53", a)
	}
	s := Update(old, Sample{Char: "a"}, p)
	if len(s.Hits) != 51 || strings.Count(s.Hits, "0") != 5 {
		t.Fatalf("hits %q: want the 50 earlier presses and the new one", s.Hits)
	}
}
