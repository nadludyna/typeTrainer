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

func TestApp_CheckinReinforce(t *testing.T) {
	for _, c := range []struct {
		wpm, acc float64
		goal     float64
		level    float64
	}{
		{0, 0, 30, 1},      // no sample: the first step, the first level
		{47, 96, 60, 2},    // 47 + 5 → the 60 step; 96 % → 500 words
		{55, 97, 60, 3},    // 55 + 5 = 60 exactly
		{25, 93.9, 30, 1},  // under the first goal
		{230, 100, 200, 3}, // over the top: the last step
	} {
		h, _, _ := newApp()
		code, out := call(t, h, "POST", "/api/v1/checkin?lang=en", fmt.Sprintf(`{"mode":"reinforce","wpm":%v,"accuracy":%v}`, c.wpm, c.acc))
		if code != 200 || out["targetWpm"].(float64) != c.goal || out["wordLevel"].(float64) != c.level {
			t.Fatalf("%v wpm %v%%: %d %v, want goal %v level %v", c.wpm, c.acc, code, out, c.goal, c.level)
		}
		if _, st := call(t, h, "GET", "/api/v1/stats?lang=en", ""); st["targetWpm"].(float64) != c.goal {
			t.Fatalf("%v wpm: the goal is not kept: %v", c.wpm, st["targetWpm"])
		}
	}
	h, _, _ := newApp()
	if code, _ := call(t, h, "POST", "/api/v1/checkin?lang=en", `{"mode":"reinforce","wpm":999,"accuracy":50}`); code != 400 {
		t.Fatalf("an impossible speed: %d, want 400", code)
	}
}

func TestApp_CheckinFreetype(t *testing.T) {
	h, _, _ := newApp()
	for wpm, want := range map[float64]float64{0: 200, 39.9: 200, 40: 1000, 69: 1000, 70: 5000, 150: 5000} {
		code, out := call(t, h, "POST", "/api/v1/checkin?lang=en", fmt.Sprintf(`{"mode":"freetype","wpm":%v,"accuracy":95}`, wpm))
		if code != 200 || out["list"].(float64) != want {
			t.Fatalf("%v wpm: %d %v, want list %v", wpm, code, out, want)
		}
	}
	if code, _ := call(t, h, "POST", "/api/v1/checkin?lang=en", `{"mode":"climb"}`); code != 400 {
		t.Fatalf("an unknown mode: %d, want 400", code)
	}
}
