package layout

import (
	"os"
	"sort"
	"strings"
	"testing"
)

func TestRestingKeys(t *testing.T) {
	for file, want := range map[string]string{
		"qwerty-en": "asdfjkl;",
	} {
		data, err := os.ReadFile("../../data/layouts/" + file + ".json")
		if err != nil {
			t.Fatal(err)
		}
		l, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := l.RestingKeys()
		sort.Strings(got)
		w := strings.Split(want, "")
		sort.Strings(w)
		if strings.Join(got, "") != strings.Join(w, "") {
			t.Errorf("%s: got %v, want %v", file, got, w)
		}
	}
}

func TestRestingKeys_NoBumps(t *testing.T) {
	l := &Layout{Rows: [][]Key{{{Char: "a", Finger: "lp"}}}}
	if got := l.RestingKeys(); len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

func TestPlaces(t *testing.T) {
	l, err := Parse([]byte(`{"fingers":{"lp":{},"lr":{},"ri":{},"th":{}},"offsets":[0,0.25],"rows":[
		[{"char":"tab","finger":"lp","width":1.5,"modifier":true},{"char":"q","finger":"lp"},{"char":"w","finger":"lr"}],
		[{"char":"a","finger":"lp"},{"char":"j","finger":"ri"}],
		[{"char":" ","finger":"th","width":6}]]}`))
	if err != nil {
		t.Fatal(err)
	}
	got := l.Places()
	want := map[string]Place{"q": {"lp", 0, 2}, "w": {"lr", 0, 3}, "a": {"lp", 1, 0.75}, "j": {"ri", 1, 1.75}}
	if len(got) != len(want) {
		t.Fatalf("places %v, want %v (no modifiers, no space)", got, want)
	}
	for c, p := range want {
		if got[c] != p {
			t.Errorf("%s: %+v, want %+v", c, got[c], p)
		}
	}
	if o := l.FingerOrder(); len(o) != 3 || o[0] != "lp" || o[1] != "lr" || o[2] != "ri" {
		t.Errorf("fingers %v, want lp lr ri (no thumb)", o)
	}
}
