package engine

import "testing"

var fp = ForecastParams{Window: 6, MinRounds: 3, MaxRounds: 50}

var fp30 = Params{TargetWPM: 30, MasteryMinSamples: 20, MasteryLevel: 80, PriorDifficulty: 0.7, PriorStrength: 5, SlowMs: 1000, MaxErrorRate: 0.25, ErrorWeight: 0.6, SpeedWeight: 0.4, Alpha: 0.1}

func line(n, count, per int, err, errStep, ms, msStep float64) []Point {
	out := make([]Point, n)
	for i := range out {
		out[i] = Point{Count: count + i*per, ErrEMA: err + float64(i)*errStep, TimeEMA: ms + float64(i)*msStep}
	}
	return out
}

func slip(pts []Point, n int, err float64) []Point {
	pts[len(pts)-1].Mistakes = pts[0].Mistakes + n
	pts[len(pts)-1].ErrEMA = err
	return pts
}

func TestRoundsToMastery(t *testing.T) {
	cases := []struct {
		name   string
		points []Point
		want   int
		ok     bool
	}{
		{"no data", nil, 0, false},
		{"at the goal already", []Point{{Count: 1000, ErrEMA: 0.02, TimeEMA: 380}}, 0, true},
		{"too few rounds", line(2, 1000, 10, 0.2, -0.02, 600, -20), 0, false},
		{"never timed", line(6, 1000, 10, 0.2, -0.02, 0, 0), 0, false},
		{"speed decides", line(6, 1000, 10, 0, 0, 880, -12), 11, true},
		{"errors decide", line(6, 1000, 10, 0.2, -0.005, 400, 0), 19, true},
		{"presses decide", line(6, 2, 2, 0, 0, 300, 0), 4, true},
		{"not getting better", line(6, 1000, 10, 0.03, 0, 900, 5), 0, false},
		{"one slip pulls the line down", slip(line(6, 1000, 10, 0, 0, 760, -1), 1, 0.1), 0, false},
		{"no presses come", line(6, 10, 0, 0.01, 0, 350, 0), 0, false},
		{"far away", line(6, 1000, 10, 0, 0, 1000, -1), 51, true},
		{"almost", line(6, 1000, 10, 0, 0, 755, -10), 1, true},
	}
	for _, c := range cases {
		got, ok := RoundsToMastery(c.points, fp30, fp)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: %d, %v; want %d, %v", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestRoundsToMastery_Window(t *testing.T) {
	pts := append(line(10, 1000, 5, 0, 0, 500, 30), line(10, 1050, 5, 0, 0, 880, -12)...)
	if got, ok := RoundsToMastery(pts, fp30, fp); !ok || got != 7 {
		t.Errorf("window: %d, %v; want 7 (only the last rounds count: level 75.1, 0.8 more a round)", got, ok)
	}
}

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
