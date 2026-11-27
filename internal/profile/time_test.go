package profile

import (
	"slices"
	"testing"
	"time"
)

var timeParams = TimeParams{Forget: 5 * 24 * time.Hour, Returning: 2, Sitting: 30 * time.Minute, TiredRuns: 3, TiredDrop: 3, ColdAfter: 6 * time.Hour}

var now = time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)

func days(n float64) time.Time { return now.Add(-time.Duration(n * 24 * float64(time.Hour))) }

func TestReturning(t *testing.T) {
	letters := []string{"a", "q", "x", "z"}
	if got := Returning(Seen{}, letters, now, timeParams); got != nil {
		t.Fatalf("no rounds: %v", got)
	}
	seen := Seen{Oldest: days(20), Last: map[string]time.Time{"a": days(0.1), "q": days(6), "x": days(4.9), "z": days(9)}}
	if got := Returning(seen, letters, now, timeParams); !slices.Equal(got, []string{"z", "q"}) {
		t.Fatalf("away for 5 days or more, the longest first: %v", got)
	}
	// j was never typed in the rounds read: it counts from the oldest round
	if got := Returning(seen, []string{"a", "j", "x"}, now, timeParams); !slices.Equal(got, []string{"j"}) {
		t.Fatalf("a letter never typed: %v", got)
	}
	fresh := Seen{Oldest: days(2), Last: map[string]time.Time{"a": days(1)}}
	if got := Returning(fresh, []string{"a", "q"}, now, timeParams); len(got) != 0 {
		t.Fatalf("a history of two days has forgotten nothing: %v", got)
	}
	three := Seen{Oldest: days(30), Last: map[string]time.Time{}}
	if got := Returning(three, letters, now, timeParams); len(got) != 2 {
		t.Fatalf("at most two letters come back at once: %v", got)
	}
}

func TestTired(t *testing.T) {
	sitting := func(gap time.Duration, acc ...float64) []Stamp {
		out := make([]Stamp, len(acc))
		for i, a := range acc {
			out[i] = Stamp{At: now.Add(time.Duration(i) * gap), Accuracy: a}
		}
		return out
	}
	for _, c := range []struct {
		name   string
		rounds []Stamp
		want   bool
	}{
		{"no rounds", nil, false},
		{"three rounds are too few", sitting(time.Minute, 98, 96, 94), false},
		{"three falls in a row", sitting(time.Minute, 98, 97, 96, 94), true},
		{"only the last four count", sitting(time.Minute, 90, 99, 97, 96, 94), true},
		{"a round that holds breaks the row", sitting(time.Minute, 98, 97, 97, 94), false},
		{"a fall of two points is noise", sitting(time.Minute, 98, 97.5, 97, 96.5), false},
		{"a break between the rounds", sitting(time.Hour, 98, 97, 96, 94), false},
	} {
		if got := Tired(c.rounds, timeParams); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	if Tired(sitting(time.Minute, 98, 97, 96, 94), TimeParams{}) {
		t.Error("switched off with no runs")
	}
}

func TestCold(t *testing.T) {
	ends := []time.Time{now, now.Add(time.Minute), now.Add(7 * time.Hour), now.Add(7*time.Hour + time.Minute)}
	if got := Cold(ends, timeParams); !slices.Equal(got, []bool{true, false, true, false}) {
		t.Fatalf("the first round and the first after a long break: %v", got)
	}
	if got := Cold(ends, TimeParams{}); slices.Contains(got, true) {
		t.Fatalf("switched off: %v", got)
	}
	if got := Cold(nil, timeParams); len(got) != 0 {
		t.Fatalf("no rounds: %v", got)
	}
}
