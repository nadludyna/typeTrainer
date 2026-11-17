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
}

func (pt Point) level(p Params) float64 {
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
	countOK := last.Count >= p.MasteryMinSamples
	levelOK := last.TimeEMA > 0 && last.level(p) >= p.MasteryLevel
	if countOK && levelOK {
		return 0, true
	}
	w := points[max(0, len(points)-max(fp.Window, 2)):]
	if len(w) < max(fp.MinRounds, 2) || last.TimeEMA <= 0 {
		return 0, false
	}
	presses := last.Count - w[0].Count
	if presses <= 0 {
		return 0, false
	}
	per := float64(presses) / float64(len(w)-1)
	need := 0.0
	if !countOK {
		need = float64(p.MasteryMinSamples-last.Count) / per
	}
	if !levelOK {
		slope := lineSlope(w, func(pt Point) float64 { return pt.level(p) })
		if slope <= 0 {
			return 0, false
		}
		need = math.Max(need, (p.MasteryLevel-last.level(p))/slope)
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
