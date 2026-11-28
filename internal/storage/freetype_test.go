package storage

import "testing"

func TestFreetypeMode(t *testing.T) {
	if got := FreetypeMode(1000); got != "freetype-1000" {
		t.Errorf("FreetypeMode(1000) = %q", got)
	}
	if n, ok := ParseFreetypeMode("freetype-1000"); !ok || n != 1000 {
		t.Errorf("ParseFreetypeMode = %d, %v", n, ok)
	}
	for _, bad := range []string{"", "0", "-1", "01", "1000-long", "sentences", "x"} {
		if _, ok := ParseFreetypeKey(bad); ok {
			t.Errorf("ParseFreetypeKey(%q) accepted", bad)
		}
	}
}
