package main

import (
	"fmt"
	"testing"
)

func openLetters(t *testing.T, st map[string]any) int {
	t.Helper()
	n := 0
	for _, k := range st["keys"].([]any) {
		if k.(map[string]any)["unlocked"] == true {
			n++
		}
	}
	return n
}

func TestApp_CheckinLearn(t *testing.T) {
	for _, c := range []struct {
		answer string
		want   int
	}{{"no", 6}, {"little", 10}, {"yes", 10}} {
		h, _, _ := newApp()
		code, out := call(t, h, "POST", "/api/v1/checkin?lang=en", fmt.Sprintf(`{"mode":"learn","answer":%q}`, c.answer))
		if code != 200 || out["unlocked"].(float64) != float64(c.want) {
			t.Fatalf("%s: %d %v, want %d letters", c.answer, code, out, c.want)
		}
		if _, st := call(t, h, "GET", "/api/v1/stats?lang=en", ""); openLetters(t, st) != c.want {
			t.Fatalf("%s: %d letters open, want %d", c.answer, openLetters(t, st), c.want)
		}
	}
	h, _, _ := newApp()
	if code, _ := call(t, h, "POST", "/api/v1/checkin?lang=en", `{"mode":"learn","answer":"maybe"}`); code != 400 {
		t.Fatalf("an unknown answer: %d, want 400", code)
	}
	call(t, h, "POST", "/api/v1/results?lang=en", `{"keys":[{"char":"a","errors":0,"ms":300}],"chars":1,"wpm":40,"accuracy":100}`)
	if _, out := call(t, h, "POST", "/api/v1/checkin?lang=en", `{"mode":"learn","answer":"yes"}`); out["unlocked"].(float64) != 6 {
		t.Fatalf("after a Learn round the check-in changes nothing: %v", out)
	}
}
