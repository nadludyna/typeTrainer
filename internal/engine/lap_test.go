package engine

import "testing"

func TestLaps(t *testing.T) {
	p, lp := Params{TargetWPM: 30}, LapParams{StepWPM: 5, MaxWPM: 60}
	for _, c := range []struct {
		lap  int
		want float64
	}{{0, 30}, {1, 35}, {6, 60}, {9, 60}, {-1, 30}} {
		if got := p.ForLap(c.lap, lp).TargetWPM; got != c.want {
			t.Errorf("lap %d: %v wpm, want %v", c.lap, got, c.want)
		}
	}
	if got := p.LastLap(lp); got != 6 {
		t.Errorf("last lap %d, want 6", got)
	}
	if got := p.LastLap(LapParams{}); got != 0 {
		t.Errorf("no laps configured: last lap %d", got)
	}
	if p.ForLap(3, LapParams{}).TargetWPM != 30 {
		t.Error("no laps configured: the goal must not move")
	}

	m := Params{TargetWPM: 30, MasteryMinSamples: 20, MasteryLevel: 80, PriorDifficulty: 0.7, PriorStrength: 5, SlowMs: 1000, MaxErrorRate: 0.25, ErrorWeight: 0.6, SpeedWeight: 0.4}
	done := KeyStat{Count: 30, ErrEMA: 0.01, TimeEMA: 700, TimedCount: 30, Held: true}
	stats := map[string]KeyStat{"a": done, "b": done}
	if LapDone(stats, []string{"a", "b", "c"}, 2, m) {
		t.Error("a letter still locked: the lap is not done")
	}
	if !LapDone(stats, []string{"a", "b"}, 2, m) {
		t.Error("every letter open and mastered: the lap is done")
	}
	if LapDone(nil, nil, 0, m) {
		t.Error("no letters: not done")
	}
	next := StartLap(stats)
	if next["a"].Held || !stats["a"].Held || next["a"].TimeEMA != 700 {
		t.Errorf("StartLap: %+v (input %+v)", next["a"], stats["a"])
	}
	if LapDone(next, []string{"a", "b"}, 2, m.ForLap(1, lp)) {
		t.Error("the next lap is done at once")
	}
}
