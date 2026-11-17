// How many rounds a letter still needs to be mastered.

package engine

import "math"

type ForecastParams struct {
	Window    int
	MinRounds int
	MaxRounds int
}

type Point struct {
	Count    int
	Mistakes int
	ErrEMA   float64
	TimeEMA  float64
	Accuracy float64 // the level under Learn's accuracy rules, 0..100
}

// Blocking: the open letters that hold the next one back.
func Blocking(stats map[string]KeyStat, open []string, p Params) []string {
	var out []string
	for i, c := range open {
		s := stats[c]
		if p.accuracyRules() {
			if a := Accuracy(s, p); (i == len(open)-1 && a < p.AccMaster) || a < p.AccSlip {
				out = append(out, c)
			}
		} else if !Mastered(s, p) {
			out = append(out, c)
		}
	}
	return out
}

func (pt Point) level(p Params) float64 {
	if p.accuracyRules() {
		return pt.Accuracy
	}
	s := KeyStat{Count: pt.Count, ErrEMA: pt.ErrEMA, TimeEMA: pt.TimeEMA}
	if pt.TimeEMA > 0 {
		s.TimedCount = 1
	}
	return Level(s, p)
}

// RoundsToMastery: the presses still missing at the recent presses per
// round, and the level still missing at the slope of the least-squares
// line through its last rounds; the slower of the two. No answer while the
// level does not grow.
func RoundsToMastery(points []Point, p Params, fp ForecastParams) (rounds int, ok bool) {
	if len(points) == 0 {
		return 0, false
	}
	last := points[len(points)-1]
	minCount, target, timed := p.MasteryMinSamples, p.MasteryLevel, last.TimeEMA > 0
	if p.accuracyRules() {
		minCount, target, timed = 0, 100*p.AccMaster, true
	}
	countOK := last.Count >= minCount
	levelOK := timed && last.level(p) >= target
	if countOK && levelOK {
		return 0, true
	}
	w := points[max(0, len(points)-max(fp.Window, 2)):]
	if len(w) < max(fp.MinRounds, 2) || !timed {
		return 0, false
	}
	presses := last.Count - w[0].Count
	if presses <= 0 {
		return 0, false
	}
	per := float64(presses) / float64(len(w)-1)
	need := 0.0
	if !countOK {
		need = float64(minCount-last.Count) / per
	}
	if !levelOK {
		slope := lineSlope(w, func(pt Point) float64 { return pt.level(p) })
		if slope <= 0 {
			return 0, false
		}
		need = math.Max(need, (target-last.level(p))/slope)
	}
	if need > float64(fp.MaxRounds) {
		return fp.MaxRounds + 1, true
	}
	return max(1, int(math.Ceil(need))), true
}

func lineSlope(points []Point, y func(Point) float64) float64 {
	n := float64(len(points))
	if n < 2 {
		return 0
	}
	mx, my := (n-1)/2, 0.0
	for _, pt := range points {
		my += y(pt)
	}
	my /= n
	num, den := 0.0, 0.0
	for i, pt := range points {
		dx := float64(i) - mx
		num += dx * (y(pt) - my)
		den += dx * dx
	}
	return num / den
}
