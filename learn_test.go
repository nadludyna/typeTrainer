package main

// Learn through the whole app.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"typingtrainer/internal/demo"
	"typingtrainer/internal/language"
	"typingtrainer/internal/storage"
)

func TestApp_PortionLearnsWithoutAHistoryEntry(t *testing.T) {
	h, _, _ := newApp()
	body := `{"samples":[{"char":"a","errors":1,"ms":300},{"char":" ","errors":0,"ms":200}],"recent":["as"],"retry":["as"]}`
	code, out := call(t, h, "POST", "/api/v1/portion?lang=en", body)
	if code != 200 {
		t.Fatalf("portion: %d %v", code, out)
	}
	por := out["portions"].([]any)[0].(map[string]any)
	if len(por["words"].([]any)) == 0 || por["focus"] == "" || len(out["portions"].([]any)) != 1 {
		t.Fatalf("portion: %v", out)
	}
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if st["sessionCount"].(float64) != 0 {
		t.Fatal("a portion must not add a history entry")
	}
	counted := false
	for _, k := range st["keys"].([]any) {
		if k := k.(map[string]any); k["char"] == "a" && k["count"].(float64) > 0 {
			counted = true
		}
	}
	if counted {
		t.Fatal("a portion must not apply its samples to the model")
	}
	if code, out := call(t, h, "POST", "/api/v1/results?lang=en", `{"keys":[{"char":"a","errors":1,"ms":300}],"samples":[],"chars":40,"wpm":30,"accuracy":95}`); code != 200 {
		t.Fatalf("results: %d %v", code, out)
	}
	_, st = call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if st["sessionCount"].(float64) != 1 {
		t.Fatal("results must add one history entry")
	}
	counted = false
	for _, k := range st["keys"].([]any) {
		if k := k.(map[string]any); k["char"] == "a" && k["count"].(float64) == 1 {
			counted = true
		}
	}
	if !counted {
		t.Fatal("results must apply the whole round's keys once")
	}
	_, out = call(t, h, "POST", "/api/v1/portion?lang=en", `{"last":true}`)
	if len(out["portions"].([]any)) != 0 {
		t.Fatalf("last: want no portions, got %v", out)
	}
	for _, bad := range []string{`{"start":"yes"}`, `{"samples":[{"char":"ж","errors":0,"ms":1}]}`} {
		if code, _ := call(t, h, "POST", "/api/v1/portion?lang=en", bad); code != 400 {
			t.Fatalf("%s: got %d, want 400", bad, code)
		}
	}
	if code, _ := call(t, h, "POST", "/api/v1/results?lang=en", `{"samples":[],"wpm":30,"accuracy":95}`); code != 400 {
		t.Fatal("a run with nothing typed must be rejected")
	}
}

func TestApp_ResultsConsistency(t *testing.T) {
	h, _, _ := newApp()
	code, out := call(t, h, "POST", "/api/v1/results?lang=en", `{"samples":[],"chars":30,"wpm":30,"accuracy":95,"perSecond":[5,5,5,5,5,5]}`)
	if code != 200 || out["consistency"] != 100.0 {
		t.Fatalf("even pace: %d %v, want consistency 100", code, out)
	}
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if c := st["sessions"].([]any)[0].(map[string]any)["consistency"]; c != 100.0 {
		t.Fatalf("history: consistency %v, want 100", c)
	}
	_, out = call(t, h, "POST", "/api/v1/results?lang=en", `{"samples":[],"chars":10,"wpm":30,"accuracy":95,"perSecond":[5,5]}`)
	if c, there := out["consistency"]; !there || c != nil {
		t.Fatalf("short round: consistency %v, want null", c)
	}
	_, st = call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if _, there := st["sessions"].([]any)[0].(map[string]any)["consistency"]; there {
		t.Fatal("short round: the history entry must have no consistency")
	}
	if code, _ := call(t, h, "POST", "/api/v1/results?lang=en", `{"samples":[],"chars":10,"wpm":30,"accuracy":95,"perSecond":[5,-1,5,5,5]}`); code != 400 {
		t.Fatal("a negative count per second must be rejected")
	}
}

func TestApp_ResultsSentTwice(t *testing.T) {
	h, _, _ := newApp()
	body := `{"keys":[{"char":"a","errors":0,"ms":300},{"char":"s","errors":1,"ms":400}],"wpm":30,"accuracy":90,"round":"5f0c2a9e-1"}`
	for i := range 2 {
		if code, out := call(t, h, "POST", "/api/v1/results?lang=en", body); code != 200 {
			t.Fatalf("send %d: %d %v", i+1, code, out)
		}
	}
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if n := st["sessionCount"].(float64); n != 1 {
		t.Fatalf("%v rounds, want 1", n)
	}
	for _, k := range st["keys"].([]any) {
		if k := k.(map[string]any); (k["char"] == "a" || k["char"] == "s") && k["count"].(float64) != 1 {
			t.Fatalf("%v learned %v times, want 1", k["char"], k["count"])
		}
	}
	call(t, h, "POST", "/api/v1/results?lang=en", strings.Replace(body, "5f0c2a9e-1", "5f0c2a9e-2", 1))
	if _, st = call(t, h, "GET", "/api/v1/stats?lang=en", ""); st["sessionCount"].(float64) != 2 {
		t.Fatalf("a new id: %v rounds, want 2", st["sessionCount"])
	}
	if code, _ := call(t, h, "POST", "/api/v1/results?lang=en", strings.Replace(body, "5f0c2a9e-1", "a b", 1)); code != 400 {
		t.Fatalf("round id with a space: %d, want 400", code)
	}
}

type failingStore struct {
	storage.Store
	fail bool
}

var errStoreDown = errors.New("store down on purpose")

func (f *failingStore) AddSession(user, lang string, s storage.Session) error {
	if f.fail {
		f.fail = false
		return errStoreDown
	}
	return f.Store.AddSession(user, lang, s)
}

func (f *failingStore) SaveLearned(user, lang string, l storage.Learned) error {
	if f.fail && l.Session != nil {
		f.fail = false
		return errStoreDown
	}
	return f.Store.SaveLearned(user, lang, l)
}

func TestApp_ResultsAfterAFailedSave(t *testing.T) {
	store := &failingStore{Store: storage.NewMemoryStore(), fail: true}
	h, _, err := newAppWith(func() (storage.Store, error) { return store, nil })
	if err != nil {
		t.Fatal(err)
	}
	body := `{"keys":[{"char":"a","errors":0,"ms":300},{"char":"s","errors":1,"ms":400}],"wpm":30,"accuracy":90,"round":"5f0c2a9e-1"}`
	if code, _ := call(t, h, "POST", "/api/v1/results?lang=en", body); code != 500 {
		t.Fatalf("first send: %d, want 500 (the store fails)", code)
	}
	if code, out := call(t, h, "POST", "/api/v1/results?lang=en", body); code != 200 {
		t.Fatalf("second send: %d %v", code, out)
	}
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if n := st["sessionCount"].(float64); n != 1 {
		t.Fatalf("%v rounds, want 1", n)
	}
	for _, k := range st["keys"].([]any) {
		if k := k.(map[string]any); (k["char"] == "a" || k["char"] == "s") && k["count"].(float64) != 1 {
			t.Fatalf("%v learned %v times, want 1", k["char"], k["count"])
		}
	}
}

func TestApp_RareWordsAfterTheAlphabet(t *testing.T) {
	h, cfg, err := newApp()
	if err != nil {
		t.Fatal(err)
	}
	list, _ := os.ReadFile("data/langs/en/free.txt")
	rank := map[string]int{}
	for i, w := range strings.Fields(string(list)) {
		if _, ok := rank[w]; !ok {
			rank[w] = i
		}
	}
	meanRank := func() float64 {
		sum, n := 0, 0
		for i := 0; i < 100; i++ {
			_, out := call(t, h, "POST", "/api/v1/portion?lang=en", `{"start":true}`)
			for _, p := range out["portions"].([]any) {
				for _, w := range p.(map[string]any)["words"].([]any) {
					sum += rank[w.(string)]
					n++
				}
			}
		}
		return float64(sum) / float64(n)
	}
	call(t, h, "POST", "/api/v1/import?lang=en", `{"unlocked":26}`)
	start := meanRank()
	call(t, h, "POST", "/api/v1/freetype/results?lang=en", `{"wpm":40,"accuracy":98,"chars":20000,"list":1000}`)
	if again := meanRank(); again > 1.5*start {
		t.Fatalf("Freetype moved the limit: mean rank %v, was %v", again, start)
	}
	body := fmt.Sprintf(`{"samples":[],"chars":%d,"wpm":40,"accuracy":98}`, int(cfg.Words.RampLetters))
	if code, out := call(t, h, "POST", "/api/v1/results?lang=en", body); code != 200 {
		t.Fatalf("results: %d %v", code, out)
	}
	if end := meanRank(); end < 1.8*start {
		t.Fatalf("after the ramp: mean rank %v, want much higher than %v", end, start)
	}
}

func TestApp_LearnLaps(t *testing.T) {
	h, _, _ := newApp()
	keys := []string{}
	for _, c := range "abcdefghijklmnopqrstuvwxyz" {
		keys = append(keys, fmt.Sprintf(`"%c":{"count":50,"errEma":0.01,"timeEma":50,"timedCount":50}`, c))
	}
	file := `{"unlocked":26,"keys":{` + strings.Join(keys, ",") + `}}`
	if code, out := call(t, h, "POST", "/api/v1/import?lang=en", file); code != 200 {
		t.Fatalf("import: %d %v", code, out)
	}
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if st["lap"].(float64) != 0 || st["targetWpm"].(float64) != 30 || st["lastLap"].(float64) != 17 {
		t.Fatalf("before: lap %v goal %v last %v", st["lap"], st["targetWpm"], st["lastLap"])
	}
	press := `{"keys":[{"char":"a","errors":0,"ms":50}],"chars":1,"wpm":30,"accuracy":100}`
	for lap := 1; lap <= 17; lap++ {
		_, out := call(t, h, "POST", "/api/v1/results?lang=en", press)
		if want := 30 + 10*float64(lap); out["newLap"] != want {
			t.Fatalf("press %d: newLap %v, want %v", lap, out["newLap"], want)
		}
	}
	_, out := call(t, h, "POST", "/api/v1/results?lang=en", press)
	if out["newLap"] != 0.0 {
		t.Fatalf("after the last lap: newLap %v, want 0", out["newLap"])
	}
	call(t, h, "POST", "/api/v1/results?lang=en", `{"samples":[],"chars":30,"wpm":50,"accuracy":100}`)
	_, st = call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if st["lap"].(float64) != 17 || st["targetWpm"].(float64) != 200 {
		t.Fatalf("after: lap %v goal %v", st["lap"], st["targetWpm"])
	}
	if s := st["sessions"].([]any)[0].(map[string]any); s["lap"] != 17.0 {
		t.Fatalf("the round keeps its lap: %v", s["lap"])
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, guestRequest("GET", "/api/v1/export?lang=en", ""))
	exported := rec.Body.String()
	if !strings.Contains(exported, `"lap":17`) {
		t.Fatalf("export lost the lap: %s", exported)
	}
	call(t, h, "POST", "/api/v1/reset?lang=en", "")
	call(t, h, "POST", "/api/v1/import?lang=en", exported)
	if _, st = call(t, h, "GET", "/api/v1/stats?lang=en", ""); st["lap"].(float64) != 17 {
		t.Fatalf("import lost the lap: %v", st["lap"])
	}
	for _, bad := range []string{`{"lap":19}`, `{"lap":-1}`, `{"sessions":[{"wpm":30,"accuracy":90,"lap":19}]}`} {
		if code, _ := call(t, h, "POST", "/api/v1/import?lang=en", bad); code != 400 {
			t.Errorf("%s: want 400, got %d", bad, code)
		}
	}
}

func TestApp_PortionSentTwice(t *testing.T) {
	h, _, _ := newApp()
	body := `{"samples":[{"char":"a","errors":1,"ms":300}],"recent":[],"round":"r-1","portionIndex":0,"ack":0}`
	code, first := call(t, h, "POST", "/api/v1/portion?lang=en", body)
	if code != 200 {
		t.Fatalf("portion: %d %v", code, first)
	}
	code, again := call(t, h, "POST", "/api/v1/portion?lang=en", body)
	if code != 200 {
		t.Fatalf("portion again: %d %v", code, again)
	}
	if fmt.Sprint(again["portions"]) != fmt.Sprint(first["portions"]) || fmt.Sprint(again["newLetter"]) != fmt.Sprint(first["newLetter"]) {
		t.Fatalf("a repeated request got a different answer: %v vs %v", again, first)
	}
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	for _, k := range st["keys"].([]any) {
		if k := k.(map[string]any); k["char"] == "a" && k["count"].(float64) > 0 {
			t.Fatalf("a portion request must not reach the model: %v", k)
		}
	}
	next := `{"samples":[{"char":"a","errors":0,"ms":300}],"recent":[],"round":"r-1","portionIndex":1,"ack":1}`
	if code, out := call(t, h, "POST", "/api/v1/portion?lang=en", next); code != 200 || len(out["portions"].([]any)) != 1 {
		t.Fatalf("the next portion: %d %v", code, out)
	}
	for _, bad := range []string{
		`{"round":"r 1","portionIndex":0}`,
		`{"round":"r-1","portionIndex":-1}`,
		`{"round":"r-1","portionIndex":0,"ack":-1}`,
	} {
		if code, _ := call(t, h, "POST", "/api/v1/portion?lang=en", bad); code != 400 {
			t.Fatalf("%s: got %d, want 400", bad, code)
		}
	}
}

func TestApp_StartLearnOver(t *testing.T) {
	h, _, _ := newApp()
	call(t, h, "POST", "/api/v1/results?lang=en", `{"keys":[{"char":"a","errors":0,"ms":300}],"chars":1,"wpm":40,"accuracy":100}`)
	call(t, h, "POST", "/api/v1/freetype/results?lang=en", `{"keys":[{"char":"a","errors":0,"ms":300}],"chars":1,"wpm":40,"accuracy":100,"list":1000}`)
	if code, _ := call(t, h, "POST", "/api/v1/learn/reset?lang=en", ""); code != 200 {
		t.Fatalf("reset: %d", code)
	}
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if st["learnRounds"].(float64) != 0 {
		t.Fatalf("Learn rounds after starting over: %v", st["learnRounds"])
	}
	_, fp := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&list=1000", "")
	if fp["sessionCount"].(float64) != 1 {
		t.Fatalf("Freetype lost its round: %v", fp["sessionCount"])
	}
}

func TestApp_ReinforceWordLevelsAndGoalSteps(t *testing.T) {
	h, _, _ := newApp()
	if code, out := call(t, h, "POST", "/api/v1/learn/step?lang=en", `{"step":2}`); code != 200 || out["targetWpm"].(float64) != 50 {
		t.Fatalf("Reinforce's speed goal does not wait for Learn: %d %v", code, out)
	}
	call(t, h, "POST", "/api/v1/learn/step?lang=en", `{"step":0}`)
	keys := []string{}
	for _, c := range "abcdefghijklmnopqrstuvwxyz" {
		keys = append(keys, fmt.Sprintf(`"%c":{"count":50,"errEma":0.3,"timeEma":600,"timedCount":50}`, c))
	}
	if code, out := call(t, h, "POST", "/api/v1/import?lang=en", `{"unlocked":26,"keys":{`+strings.Join(keys, ",")+`}}`); code != 200 {
		t.Fatalf("import: %d %v", code, out)
	}
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	levels := st["wordLevels"].([]any)
	if len(levels) != 10 || levels[0].(float64) != 200 {
		t.Fatalf("word levels: %v", levels)
	}
	top := map[string]bool{}
	for _, w := range strings.Fields(readData(t, "data/langs/en/free.txt"))[:200] {
		top[w] = true
	}
	_, out := call(t, h, "POST", "/api/v1/portion?lang=en", `{"start":true,"level":1}`)
	for _, p := range out["portions"].([]any) {
		for _, w := range p.(map[string]any)["words"].([]any) {
			if !top[w.(string)] {
				t.Errorf("level 1 brought %q, not one of the 200 most frequent words", w)
			}
		}
	}
	if code, _ := call(t, h, "POST", "/api/v1/portion?lang=en", `{"start":true,"level":11}`); code != 400 {
		t.Fatalf("level 11: %d, want 400", code)
	}

	code, out := call(t, h, "POST", "/api/v1/learn/step?lang=en", `{"step":3}`)
	if code != 200 || out["targetWpm"].(float64) != 60 {
		t.Fatalf("step 3: %d %v", code, out)
	}
	_, st = call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if st["lap"].(float64) != 3 || st["targetWpm"].(float64) != 60 {
		t.Fatalf("after a step by hand: lap %v goal %v", st["lap"], st["targetWpm"])
	}
	if code, _ := call(t, h, "POST", "/api/v1/learn/step?lang=en", `{"step":18}`); code != 400 {
		t.Fatalf("a step past 200 wpm: %d, want 400", code)
	}
	call(t, h, "POST", "/api/v1/results?lang=en", `{"keys":[{"char":"a","errors":0,"ms":300}],"chars":1,"wpm":40,"accuracy":100,"level":4}`)
	_, st = call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if s := st["sessions"].([]any)[0].(map[string]any); s["level"] != 4.0 {
		t.Fatalf("the round keeps its word level: %v", s["level"])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, guestRequest("GET", "/api/v1/export?lang=en", ""))
	if code, out := call(t, h, "POST", "/api/v1/import?lang=en", rec.Body.String()); code != 200 {
		t.Fatalf("export with a word level does not import: %d %v", code, out)
	}
}

func readData(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Reinforce before Learn is finished: every letter comes, the presses count
// for each letter, but only Learn opens the next one.
func TestApp_ReinforceBeforeLearnIsFinished(t *testing.T) {
	h, _, _ := newApp()
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	open := 0
	for _, k := range st["keys"].([]any) {
		if k.(map[string]any)["unlocked"] == true {
			open++
		}
	}
	if open >= 26 {
		t.Fatalf("a new user should start with a few letters, has %d", open)
	}
	_, out := call(t, h, "POST", "/api/v1/portion?lang=en", `{"start":true,"level":3}`)
	locked := map[string]bool{}
	for _, c := range out["portions"].([]any)[0].(map[string]any)["locked"].([]any) {
		locked[c.(string)] = true
	}
	if len(locked) != 0 {
		t.Fatalf("a Reinforce portion has every letter open, locked: %v", locked)
	}
	sawLocked := false
	_, learnPor := call(t, h, "POST", "/api/v1/portion?lang=en", `{"start":true}`)
	learnLocked := map[string]bool{}
	for _, c := range learnPor["portions"].([]any)[0].(map[string]any)["locked"].([]any) {
		learnLocked[c.(string)] = true
	}
	for _, p := range out["portions"].([]any) {
		for _, w := range p.(map[string]any)["words"].([]any) {
			for _, c := range w.(string) {
				if learnLocked[string(c)] {
					sawLocked = true
				}
			}
		}
	}
	if !sawLocked {
		t.Fatal("Reinforce should bring words with letters Learn has not opened yet")
	}
	var presses []string
	for i := 0; i < 120; i++ {
		presses = append(presses, `{"char":"z","errors":0,"ms":200}`)
	}
	body := `{"keys":[` + strings.Join(presses, ",") + `],"chars":120,"wpm":60,"accuracy":100,"level":3}`
	if code, out := call(t, h, "POST", "/api/v1/results?lang=en", body); code != 200 || out["newLetter"] != nil && out["newLetter"] != "" {
		t.Fatalf("a Reinforce round must not open a letter: %d %v", code, out)
	}
	_, after := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	n := 0
	var z map[string]any
	for _, k := range after["keys"].([]any) {
		k := k.(map[string]any)
		if k["unlocked"] == true {
			n++
		}
		if k["char"] == "z" {
			z = k
		}
	}
	if n != open {
		t.Fatalf("open letters changed in Reinforce: %d, was %d", n, open)
	}
	if z == nil || z["count"].(float64) != 120 {
		t.Fatalf("Reinforce's presses count for the letter z: %v", z)
	}
	if s := after["sessions"].([]any)[0].(map[string]any); s["level"] != 3.0 {
		t.Fatalf("a Reinforce round keeps its word level before Learn is finished: %v", s["level"])
	}
}

// The stats of each mode: Learn's rounds and Reinforce's (with a word level)
// are counted apart, and so are their days.
func TestApp_StatsPerMode(t *testing.T) {
	h, _, _ := newApp()
	for _, round := range []string{`"wpm":30`, `"wpm":50,"level":2`, `"wpm":60,"level":2`} {
		if code, out := call(t, h, "POST", "/api/v1/results?lang=en", `{"keys":[{"char":"a","errors":0,"ms":300}],"chars":1,"accuracy":100,`+round+`}`); code != 200 {
			t.Fatalf("results: %d %v", code, out)
		}
	}
	for part, want := range map[string]float64{"": 3, "learn": 1, "reinforce": 2} {
		_, st := call(t, h, "GET", "/api/v1/stats?lang=en&part="+part, "")
		if st["sessionCount"].(float64) != want || len(st["sessions"].([]any)) != int(want) {
			t.Fatalf("part %q: %v rounds, want %v", part, st["sessionCount"], want)
		}
		if _, pairs := st["pairs"]; pairs != (part == "reinforce") {
			t.Fatalf("part %q: pairs sent = %v, only Reinforce has them", part, pairs)
		}
	}
	_, re := call(t, h, "GET", "/api/v1/stats?lang=en&part=reinforce", "")
	if best := re["bests"].(map[string]any)["words-30"].(map[string]any)["wpm"]; best != 60.0 {
		t.Fatalf("Reinforce's best: %v, want 60", best)
	}
	for _, path := range []string{"/api/v1/stats?lang=en&part=freetype"} {
		if code, _ := call(t, h, "GET", path, ""); code != 400 {
			t.Fatalf("%s: %d, want 400", path, code)
		}
	}
}

// Reinforce's word level moves by its rounds: three at the goal speed with
// 96 % or more move it up, two under 90 % of the goal move it down, a level
// picked by hand starts a run of its own.
func TestApp_ReinforceLevelMovesByItself(t *testing.T) {
	h, _, _ := newApp()
	round := func(level int, wpm float64) map[string]any {
		t.Helper()
		body := fmt.Sprintf(`{"keys":[{"char":"a","errors":0,"ms":300}],"chars":1,"accuracy":99,"wpm":%v,"level":%d}`, wpm, level)
		if code, out := call(t, h, "POST", "/api/v1/results?lang=en", body); code != 200 {
			t.Fatalf("results: %d %v", code, out)
		}
		_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
		return st["standing"].(map[string]any)
	}
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	goal := st["targetWpm"].(float64)
	if s := st["standing"].(map[string]any); s["level"] != 1.0 || s["moved"] != 0.0 || s["passesUp"] != 3.0 || s["failsDown"] != 2.0 {
		t.Fatalf("no rounds yet: %v", s)
	}
	round(3, goal+5)
	if s := round(3, goal); s["passes"] != 2.0 || s["moved"] != 0.0 {
		t.Fatalf("two passes: %v", s)
	}
	if s := round(3, goal+10); s["level"] != 4.0 || s["moved"] != 1.0 {
		t.Fatalf("the third pass moves up: %v", s)
	}
	if s := round(2, goal); s["level"] != 2.0 || s["passes"] != 1.0 || s["moved"] != 0.0 {
		t.Fatalf("a level picked by hand starts its own run: %v", s)
	}
	if s := round(2, goal*0.95); s["level"] != 2.0 || s["passes"] != 1.0 || s["fails"] != 0.0 {
		t.Fatalf("just under the goal holds: %v", s)
	}
	round(2, goal*0.5)
	if s := round(2, goal*0.5); s["level"] != 1.0 || s["moved"] != -1.0 {
		t.Fatalf("two rounds far under the goal move down: %v", s)
	}
	if _, learn := call(t, h, "GET", "/api/v1/stats?lang=en&part=learn", ""); learn["standing"] != nil {
		t.Fatalf("Learn's stats carry no word level: %v", learn["standing"])
	}
}

// The demo history (go run ./tools/demo) is a file the server takes, with
// rounds in every mode, Reinforce's at a word level.
func TestDemo_Imports(t *testing.T) {
	h, cfg, _ := newApp()
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	var order []string
	for _, c := range st["order"].([]any) {
		order = append(order, c.(string))
	}
	list, err := os.ReadFile("data/langs/en/free.txt")
	if err != nil {
		t.Fatal(err)
	}
	info, _ := language.Parse([]byte(`[{"id":"en","name":"English","label":"EN","layout":"qwerty-en","alphabet":"abcdefghijklmnopqrstuvwxyz"}]`))
	body, _ := json.Marshal(demo.Build(order, info[0].Words(string(list)), cfg.Engine, time.Now(), 1))
	if code, out := call(t, h, "POST", "/api/v1/import?lang=en", string(body)); code != 200 {
		t.Fatalf("import: %d %v", code, out)
	}
	for part, want := range map[string]float64{"learn": 37, "reinforce": 21} {
		if _, s := call(t, h, "GET", "/api/v1/stats?lang=en&part="+part, ""); s["sessionCount"].(float64) != want {
			t.Errorf("%s: %v rounds, want %v", part, s["sessionCount"], want)
		}
	}
	if _, s := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&list=1000&length=words-30", ""); s["sessionCount"].(float64) != 12 {
		t.Errorf("freetype: %v rounds, want 12", s["sessionCount"])
	}
	if _, s := call(t, h, "GET", "/api/v1/stats?lang=en&part=learn", ""); len(s["confusions"].([]any)) == 0 {
		t.Errorf("no keys mixed up: the learn grid and its mix-ups would be empty")
	}
}

// The typist's profile in /stats: a left little finger that misses its keys
// by slipping to the next one is named weak, with its letters and the kind
// of mistake; phone rounds say nothing.
func TestApp_ProfileFingersAndMistakes(t *testing.T) {
	h, _, _ := newApp()
	var b strings.Builder
	for i := range 12 {
		for j, c := range "a sad lass falls as a salad " {
			if i > 0 || j > 0 {
				b.WriteString(",")
			}
			if c == 'a' {
				b.WriteString(`{"char":"a","errors":1,"ms":300,"typed":"q"}`)
			} else {
				fmt.Fprintf(&b, `{"char":%q,"errors":0,"ms":200}`, string(c))
			}
		}
	}
	send := func() {
		t.Helper()
		body := fmt.Sprintf(`{"keys":[%s],"chars":300,"accuracy":90,"wpm":30}`, b.String())
		if code, out := call(t, h, "POST", "/api/v1/results?lang=en", body); code != 200 {
			t.Fatalf("results: %d %v", code, out)
		}
	}
	profile := func() map[string]any {
		_, st := call(t, h, "GET", "/api/v1/stats?lang=en&part=learn", "")
		return st["profile"].(map[string]any)
	}
	if p := profile(); p["presses"] != 0.0 || p["weak"] != "" {
		t.Fatalf("no rounds: %v", p)
	}
	send()
	p := profile()
	if p["weak"] != "lp" || fmt.Sprint(p["letters"]) != "[a q z]" {
		t.Fatalf("weak finger %v, letters %v", p["weak"], p["letters"])
	}
	kinds := p["kinds"].([]any)
	if len(kinds) != 1 || kinds[0].(map[string]any)["kind"] != "slip" {
		t.Fatalf("kinds %v, want slip only", kinds)
	}
}

// Accuracy that falls three rounds in a row in one sitting: /stats says the
// typist is tired; a round that holds its accuracy clears it.
func TestApp_Tired(t *testing.T) {
	h, _, _ := newApp()
	tired := func() any {
		_, st := call(t, h, "GET", "/api/v1/stats?lang=en&part=learn", "")
		return st["tired"]
	}
	send := func(acc float64) {
		t.Helper()
		body := fmt.Sprintf(`{"keys":[{"char":"a","errors":0,"ms":300}],"chars":1,"accuracy":%v,"wpm":30}`, acc)
		if code, out := call(t, h, "POST", "/api/v1/results?lang=en", body); code != 200 {
			t.Fatalf("results: %d %v", code, out)
		}
	}
	if tired() != false {
		t.Fatal("no rounds: not tired")
	}
	for _, acc := range []float64{98, 97, 96} {
		send(acc)
	}
	if tired() != false {
		t.Fatal("two falls are not three")
	}
	send(93)
	if tired() != true {
		t.Fatal("three falls in a row, five points in all: tired")
	}
	send(95)
	if tired() != false {
		t.Fatal("a better round clears it")
	}
}

// A round of a kind is kept and counted, but moves no level: three
// weak-spots rounds at the goal leave Reinforce's level where it was, and
// Learn has only the usual round.
func TestApp_RoundKinds(t *testing.T) {
	h, _, _ := newApp()
	_, st := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	goal := st["targetWpm"].(float64)
	send := func(kind string, level int) (int, map[string]any) {
		body := fmt.Sprintf(`{"keys":[{"char":"a","errors":0,"ms":300}],"chars":1,"accuracy":99,"wpm":%v,"level":%d,"kind":%q}`, goal+5, level, kind)
		return call(t, h, "POST", "/api/v1/results?lang=en", body)
	}
	for range 3 {
		if code, out := send("weak", 3); code != 200 {
			t.Fatalf("a weak-spots round: %d %v", code, out)
		}
	}
	_, st = call(t, h, "GET", "/api/v1/stats?lang=en&part=reinforce", "")
	if s := st["standing"].(map[string]any); s["passes"] != 0.0 || s["moved"] != 0.0 || st["sessionCount"] != 3.0 {
		t.Fatalf("three weak-spots rounds are shown and judge nothing: %v, %v rounds", s, st["sessionCount"])
	}
	if st["sessions"].([]any)[0].(map[string]any)["kind"] != "weak" || st["weakSpots"] == nil {
		t.Fatalf("the round keeps its kind, and the stats count the weak spots: %v", st["weakSpots"])
	}
	if code, _ := send("race", 3); code != 400 {
		t.Fatalf("an unknown kind: %d, want 400", code)
	}
	// Three sprints at the goal judge nothing either, and have a best of their own.
	for range 3 {
		if code, out := send("sprint", 3); code != 200 {
			t.Fatalf("a sprint: %d %v", code, out)
		}
	}
	_, st = call(t, h, "GET", "/api/v1/stats?lang=en&part=reinforce", "")
	if s := st["standing"].(map[string]any); s["passes"] != 0.0 || s["moved"] != 0.0 {
		t.Fatalf("three sprints judge nothing: %v", s)
	}
	if b := st["bests"].(map[string]any); b["sprint"] == nil {
		t.Fatalf("the stats have the best sprint: %v", b)
	}
	if code, _ := call(t, h, "POST", "/api/v1/portion?lang=en", `{"start":true,"level":3,"kind":"sprint","round":"r2","portionIndex":0}`); code != 200 {
		t.Fatalf("a sprint portion with nothing owned is a usual one: %d", code)
	}
	if code, _ := send("weak", 0); code != 400 {
		t.Fatalf("Learn has only the usual round: %d, want 400", code)
	}
	if code, out := call(t, h, "POST", "/api/v1/portion?lang=en", `{"start":true,"level":3,"kind":"weak","round":"r1","portionIndex":0}`); code != 200 || len(out["portions"].([]any)) == 0 {
		t.Fatalf("a weak-spots portion with no weak spots yet is a usual one: %d %v", code, out)
	}
	if code, _ := call(t, h, "POST", "/api/v1/freetype/portion?lang=en", `{"start":true,"list":200,"words":10,"kind":"weak"}`); code != 200 {
		t.Fatalf("Freetype's weak-spots portion: %d", code)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, guestRequest("GET", "/api/v1/export?lang=en", ""))
	if file := rec.Body.String(); !strings.Contains(file, `"kind":"weak"`) {
		t.Fatalf("the export keeps the kind: %.200s", file)
	}
}
