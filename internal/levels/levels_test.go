package levels

import "testing"

var testParams = Params{
	Words: []int{200, 500, 1000, 2000, 3000, 5000, 7500, 10000, 15000, 25000}, LevelStart: 1,
	PassAccuracy: 96, FailAccuracy: 92, FailSpeed: 0.9, ConsPass: 80, ConsFail: 70, PassesUp: 3, FailsDown: 2, History: 50,
	WeakRounds: 20, WeakMinSamples: 15, WeakMissMs: 1000, WeakPairs: 3,
}

func TestJudge(t *testing.T) {
	for acc, want := range map[float64]Verdict{100: Pass, 96: Pass, 95.9: Hold, 92: Hold, 91.9: Fail, 0: Fail} {
		if got := Judge(Round{Accuracy: acc}, testParams); got != want {
			t.Errorf("no goal, accuracy %v: %s, want %s", acc, got, want)
		}
	}
	for _, c := range []struct {
		wpm, acc float64
		want     Verdict
	}{
		{60, 97, Pass},   // at the goal, accurate
		{75, 99, Pass},   // above it
		{59.9, 97, Hold}, // just under the goal: the zone
		{54, 97, Hold},   // 90 % of the goal is still the zone
		{53.9, 99, Fail}, // under 90 %: too slow, however accurate
		{80, 91, Fail},   // fast but careless
		{60, 94, Hold},   // at the goal, accuracy in the zone
	} {
		if got := Judge(Round{WPM: c.wpm, Accuracy: c.acc, Goal: 60}, testParams); got != c.want {
			t.Errorf("goal 60, %v wpm at %v%%: %s, want %s", c.wpm, c.acc, got, c.want)
		}
	}
	if got := Judge(Round{WPM: 10, Accuracy: 97, Goal: 60}, Params{PassAccuracy: 96, FailAccuracy: 92}); got != Hold {
		t.Errorf("FailSpeed 0: speed never fails a round, got %s", got)
	}
}

func rounds(level int, accs ...float64) []Round {
	out := make([]Round, len(accs))
	for i, a := range accs {
		out[i] = Round{Level: level, Accuracy: a}
	}
	return out
}

func TestStanding(t *testing.T) {
	p := testParams
	for _, c := range []struct {
		name   string
		rounds []Round
		want   Place
	}{
		{"no history: LevelStart", nil, Place{Level: 1}},
		{"one pass", rounds(4, 98), Place{Level: 4, Passes: 1}},
		{"two passes and a hold", rounds(4, 98, 94, 97), Place{Level: 4, Passes: 2}},
		{"three passes: up", rounds(4, 98, 99, 97), Place{Level: 5, Moved: 1}},
		{"a fail starts the passes again", rounds(4, 98, 99, 90, 97), Place{Level: 4, Passes: 1}},
		{"one fail", rounds(4, 90), Place{Level: 4, Fails: 1}},
		{"two fails in a row: down", rounds(4, 90, 85), Place{Level: 3, Moved: -1}},
		{"a hold breaks the row", rounds(4, 90, 94, 85), Place{Level: 4, Fails: 1}},
		{"the top level: three passes stay", rounds(10, 98, 98, 98), Place{Level: 10}},
		{"the top level: a fourth pass counts from 0", rounds(10, 98, 98, 98, 98), Place{Level: 10, Passes: 1}},
		{"level 1: two fails stay", rounds(1, 80, 80), Place{Level: 1}},
		{"after a move the same level again (picked): counts from 0", rounds(4, 98, 98, 98, 99), Place{Level: 4, Passes: 1}},
		{"another level before the run does not count",
			append(rounds(3, 98, 98), rounds(4, 99)...), Place{Level: 4, Passes: 1}},
		{"a level out of range counts as the nearest", rounds(14, 98), Place{Level: 10, Passes: 1}},
		{"too slow twice: down, however accurate",
			[]Round{{Level: 5, Accuracy: 99, WPM: 40, Goal: 60}, {Level: 5, Accuracy: 99, WPM: 45, Goal: 60}}, Place{Level: 4, Moved: -1}},
		{"in the zone it holds",
			[]Round{{Level: 5, Accuracy: 97, WPM: 56, Goal: 60}, {Level: 5, Accuracy: 97, WPM: 58, Goal: 60}, {Level: 5, Accuracy: 97, WPM: 57, Goal: 60}}, Place{Level: 5}},
		{"each round at its own goal",
			[]Round{{Level: 5, Accuracy: 97, WPM: 52, Goal: 50}, {Level: 5, Accuracy: 97, WPM: 62, Goal: 60}, {Level: 5, Accuracy: 97, WPM: 70, Goal: 70}}, Place{Level: 6, Moved: 1}},
	} {
		if got := Standing(c.rounds, Judge, p); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestJudgeRhythm(t *testing.T) {
	c := func(v float64) *float64 { return &v }
	for _, x := range []struct {
		r    Round
		want Verdict
	}{
		{Round{Accuracy: 98, Consistency: c(85)}, Pass},
		{Round{Accuracy: 96, Consistency: c(80)}, Pass},
		{Round{Accuracy: 95, Consistency: c(90)}, Hold},
		{Round{Accuracy: 98, Consistency: c(75)}, Hold},
		{Round{Accuracy: 98, Consistency: c(69)}, Fail},
		{Round{Accuracy: 90, Consistency: c(95)}, Fail},
		{Round{Accuracy: 100}, Hold},
		{Round{Accuracy: 100, WPM: 10, Goal: 100, Consistency: c(85)}, Pass},
	} {
		if got := JudgeRhythm(x.r, testParams); got != x.want {
			t.Errorf("%+v: %s, want %s", x.r, got, x.want)
		}
	}
	c80 := c(85)
	run := []Round{{Level: 2, Accuracy: 98, Consistency: c80}, {Level: 2, Accuracy: 98, Consistency: c80}, {Level: 2, Accuracy: 98, Consistency: c80}}
	if got := Standing(run, JudgeRhythm, testParams); got.Level != 3 || got.Moved != 1 {
		t.Fatalf("three even rounds move the list up: %+v", got)
	}
}

// A warm-up round can pass, but its fail only holds: it breaks no run of
// passes and starts no run of fails.
func TestStanding_AColdRoundDoesNotFail(t *testing.T) {
	p := testParams
	bad, good := Round{Level: 2, Accuracy: 80}, Round{Level: 2, Accuracy: 99}
	cold := bad
	cold.Cold = true
	if got := Standing([]Round{bad, bad}, Judge, p); got.Moved != -1 {
		t.Fatalf("two fails move down: %+v", got)
	}
	if got := Standing([]Round{cold, bad}, Judge, p); got.Moved != 0 || got.Fails != 1 || got.Level != 2 {
		t.Fatalf("a cold fail is not counted: %+v", got)
	}
	if got := Standing([]Round{good, good, cold}, Judge, p); got.Passes != 2 {
		t.Fatalf("a cold fail keeps the passes: %+v", got)
	}
	warm := good
	warm.Cold = true
	if got := Standing([]Round{warm, good, good}, Judge, p); got.Moved != 1 {
		t.Fatalf("a cold round can pass: %+v", got)
	}
}
