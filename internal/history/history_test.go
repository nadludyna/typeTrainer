package history

import (
	"testing"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/storage"
)

func keys(spec ...engine.Sample) []engine.Sample { return spec }

func TestConfusions(t *testing.T) {
	s := []storage.Session{
		{Keys: keys(
			engine.Sample{Char: "r", Errors: 1, Typed: "t"},
			engine.Sample{Char: "r", Errors: 2, Typed: "t"},
			engine.Sample{Char: "e", Errors: 1, Typed: "w"},
			engine.Sample{Char: "e", Errors: 1},
		)},
		{Keys: keys(
			engine.Sample{Char: "r", Errors: 1, Typed: "t"},
			engine.Sample{Char: "a", Errors: 1, Typed: "s"},
			engine.Sample{Char: "e", Errors: 1, Typed: "w"},
		)},
	}
	got := Confusions(s, 2, 5)
	want := []Confusion{{Want: "r", Typed: "t", N: 3}, {Want: "e", Typed: "w", N: 2}}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	if got := Confusions(s, 1, 1); len(got) != 1 || got[0].Want != "r" {
		t.Errorf("limit 1: got %+v", got)
	}
	if got := Confusions(nil, 1, 5); len(got) != 0 {
		t.Errorf("no rounds: got %+v", got)
	}
}
