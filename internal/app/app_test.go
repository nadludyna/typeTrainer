package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"typingtrainer/internal/storage"
)

// The whole program over HTTP, one guest per test unless a test asks for two.

func newApp(t *testing.T) http.Handler {
	t.Helper()
	h, _, err := New(storage.NewMemoryStore(), true)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func callAs(t *testing.T, h http.Handler, guest, method, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "tt_guest", Value: guest})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	out := map[string]any{}
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func call(t *testing.T, h http.Handler, method, path, body string) (int, map[string]any) {
	t.Helper()
	return callAs(t, h, "guest-1", method, path, body)
}

func get(m map[string]any, path ...string) any {
	var v any = m
	for _, k := range path {
		v = v.(map[string]any)[k]
	}
	return v
}

// presses is n clean presses of each letter, 200 ms apart.
func presses(letters string, n int) string {
	var keys []string
	for range n {
		for _, c := range letters {
			keys = append(keys, fmt.Sprintf(`{"char":%q,"errors":0,"ms":200}`, string(c)))
		}
	}
	return "[" + strings.Join(keys, ",") + "]"
}

func result(keys string, extra string) string {
	return `{"keys":` + keys + `,"wpm":30,"accuracy":100,"length":"words-30","perSecond":[5,5,5,5,5,5]` + extra + `}`
}

func TestLearnRound(t *testing.T) {
	h := newApp(t)
	code, out := call(t, h, "POST", "/api/learn/portion", `{"round":"r1","start":true}`)
	portions := out["portions"].([]any)
	if code != 200 || len(portions) != 3 || out["round"] != 5.0 {
		t.Fatalf("a round starts with the portion and two waiting, of five: %d %v", code, out)
	}
	for _, p := range portions {
		words := p.(map[string]any)["words"].([]any)
		if len(words) != 6 {
			t.Fatalf("a portion has 6 words: %v", words)
		}
		for _, w := range words {
			if strings.Trim(w.(string), "saldfe") != "" {
				t.Fatalf("%q has a locked letter", w)
			}
		}
	}
	if _, again := call(t, h, "POST", "/api/learn/portion", `{"round":"r1","start":true}`); fmt.Sprint(again) != fmt.Sprint(out) {
		t.Fatal("a portion request sent twice gets the same answer")
	}
	body := result(presses("saldfe", 30), `,"id":"r1"`)
	if code, out := call(t, h, "POST", "/api/learn/results", body); code != 200 || out["newLetter"] != "k" {
		t.Fatalf("30 clean presses of each letter open the next one: %d %v", code, out)
	}
	if _, out := call(t, h, "POST", "/api/learn/results", body); out["newLetter"] != "" || out["unlocked"] != 7.0 {
		t.Fatalf("the same round sent again opens nothing more: %v", out)
	}
	_, st := call(t, h, "GET", "/api/learn/stats", "")
	if get(st, "history", "count") != 1.0 || st["unlocked"] != 7.0 || len(st["letters"].([]any)) != 26 {
		t.Fatalf("a round sent twice is kept and learned once: %v", st["history"])
	}
	if letter := st["letters"].([]any)[0].(map[string]any); letter["char"] != "s" || letter["count"] != 30.0 || letter["open"] != true {
		t.Fatalf("the first letter: %v", letter)
	}
}

func TestLearnCheckin(t *testing.T) {
	h := newApp(t)
	if code, out := call(t, h, "POST", "/api/learn/checkin", `{"answer":"yes"}`); code != 200 || out["unlocked"] != 10.0 {
		t.Fatalf("yes opens ten letters: %d %v", code, out)
	}
	if code, _ := call(t, h, "POST", "/api/learn/checkin", `{"answer":"maybe"}`); code != 400 {
		t.Fatal("an unknown answer")
	}
	other := newApp(t)
	call(t, other, "POST", "/api/learn/results", result(presses("s", 1), ""))
	if _, out := call(t, other, "POST", "/api/learn/checkin", `{"answer":"yes"}`); out["unlocked"] != 6.0 {
		t.Fatalf("after a round the answer changes nothing: %v", out)
	}
}

func TestLearnRejects(t *testing.T) {
	h := newApp(t)
	for _, body := range []string{
		result(`[{"char":"ж","errors":0,"ms":1}]`, ""),
		result(presses("s", 1), `,"kind":"weak"`),
		result(presses("s", 1), `,"level":2`),
		strings.Replace(result(presses("s", 1), ""), "words-30", "words-31", 1),
		result(presses("s", 1), `,"id":"no spaces"`),
		`{"keys":[`,
	} {
		if code, _ := call(t, h, "POST", "/api/learn/results", body); code != 400 {
			t.Errorf("%.60s: %d, want 400", body, code)
		}
	}
}

func TestReinforce(t *testing.T) {
	h := newApp(t)
	if code, _ := call(t, h, "POST", "/api/reinforce/portion", `{"start":true}`); code != 400 {
		t.Fatal("Reinforce needs a word level")
	}
	code, out := call(t, h, "POST", "/api/reinforce/portion", `{"start":true,"level":1,"kind":"weak"}`)
	if code != 200 || len(out["portions"].([]any)) != 3 {
		t.Fatalf("portions: %d %v", code, out)
	}
	pass := result(presses("the", 3), `,"level":1,"wpm":45,"accuracy":99`)
	for i := range 3 {
		if code, out := call(t, h, "POST", "/api/reinforce/results", pass); code != 200 {
			t.Fatalf("round %d: %d %v", i, code, out)
		}
	}
	_, st := call(t, h, "GET", "/api/reinforce/stats", "")
	if get(st, "standing", "level") != 2.0 || get(st, "standing", "moved") != 1.0 || st["goal"] != 30.0 {
		t.Fatalf("three rounds over the goal move the level up: %v, goal %v", st["standing"], st["goal"])
	}
	if code, out := call(t, h, "POST", "/api/reinforce/step", `{"step":2}`); code != 200 || out["goal"] != 50.0 {
		t.Fatalf("step by hand: %d %v", code, out)
	}
	if _, st := call(t, h, "GET", "/api/reinforce/stats", ""); st["goal"] != 50.0 || st["step"] != 2.0 {
		t.Fatalf("the step is kept: %v", st["goal"])
	}
	if _, learn := call(t, h, "GET", "/api/learn/stats", ""); get(learn, "history", "count") != 0.0 {
		t.Fatal("Reinforce's rounds are not Learn's")
	}
}

func TestFreetypeRhythm(t *testing.T) {
	h := newApp(t)
	stats := func(list int) map[string]any {
		t.Helper()
		_, st := call(t, h, "GET", fmt.Sprintf("/api/freetype/stats?list=%d", list), "")
		return st
	}
	if st := stats(200); get(st, "standing", "level") != 1.0 || get(st, "rhythm", "presses") != 0.0 {
		t.Fatalf("a new typist: list 1, no rhythm: %v", st)
	}
	// "the quick " ten times: every press 150 ms, and "qu" 700 ms in half of them.
	var keys []string
	for i := range 10 {
		for _, c := range "the quick " {
			ms := 150
			if c == 'u' && i%2 == 0 {
				ms = 700
			}
			keys = append(keys, fmt.Sprintf(`{"char":%q,"errors":0,"ms":%d}`, string(c), ms))
		}
	}
	round := func(list int, perSecond string) {
		t.Helper()
		body := fmt.Sprintf(`{"wpm":60,"accuracy":99,"chars":100,"length":"time-30","perSecond":[%s],"list":%d,"keys":[%s]}`, perSecond, list, strings.Join(keys, ","))
		if code, out := call(t, h, "POST", "/api/freetype/results", body); code != 200 {
			t.Fatalf("results: %d %v", code, out)
		}
	}
	even, uneven := "5,5,5,5,5,5", "1,9,1,9,1,9"
	round(200, even)
	round(200, even)
	round(200, even)
	st := stats(200)
	if get(st, "standing", "level") != 2.0 || get(st, "history", "count") != 3.0 {
		t.Fatalf("three even rounds move the list up: %v", st["standing"])
	}
	if pairs := get(st, "rhythm", "pairs").([]any); len(pairs) != 1 || pairs[0].(map[string]any)["pair"] != "qu" {
		t.Fatalf("the stumble pair: %v", st["rhythm"])
	}
	round(1000, uneven)
	round(1000, uneven)
	if get(stats(1000), "standing", "moved") != -1.0 {
		t.Fatal("two uneven rounds move the list down")
	}
	code, out := call(t, h, "POST", "/api/freetype/portion", `{"start":true,"list":200,"words":50}`)
	if code != 200 || len(out["portions"].([]any)) != 9 {
		t.Fatalf("enough portions for 50 words: %d %v", code, out)
	}
	if code, _ := call(t, h, "POST", "/api/freetype/portion", `{"list":300}`); code != 400 {
		t.Fatal("a list not offered")
	}
}

func TestExportImport(t *testing.T) {
	h := newApp(t)
	call(t, h, "POST", "/api/learn/results", result(presses("saldfe", 30), `,"id":"a"`))
	call(t, h, "POST", "/api/freetype/results", `{"wpm":60,"accuracy":99,"chars":3,"length":"time-30","list":1000,"keys":`+presses("the", 1)+`}`)
	_, file := call(t, h, "GET", "/api/export", "")
	body, _ := json.Marshal(file)
	if rounds := file["rounds"].([]any); len(rounds) != 2 || rounds[0].(map[string]any)["mode"] != "learn" {
		t.Fatalf("export, oldest first: %v", rounds)
	}
	other := newApp(t)
	if code, out := call(t, other, "POST", "/api/import", string(body)); code != 200 || out["rounds"] != 2.0 {
		t.Fatalf("import: %d %v", code, out)
	}
	if _, again := call(t, other, "GET", "/api/export", ""); fmt.Sprint(again) != fmt.Sprint(file) {
		t.Fatal("an export of the import is the same file")
	}
	for _, bad := range []string{
		strings.Replace(string(body), `"mode":"learn"`, `"mode":"climb"`, 1),
		strings.Replace(string(body), `"unlocked":7`, `"unlocked":70`, 1),
		strings.Replace(string(body), `"length":"words-30"`, `"length":"words-7"`, 1),
	} {
		if code, _ := call(t, other, "POST", "/api/import", bad); code != 400 {
			t.Errorf("a bad file was taken: %.120s", bad)
		}
	}
	if _, st := call(t, other, "GET", "/api/learn/stats", ""); st["unlocked"] != 7.0 {
		t.Fatal("a rejected import changed the data")
	}
}

func TestResets(t *testing.T) {
	h := newApp(t)
	call(t, h, "POST", "/api/learn/results", result(presses("saldfe", 30), ""))
	call(t, h, "POST", "/api/freetype/results", `{"wpm":60,"accuracy":99,"chars":3,"length":"time-30","list":1000,"keys":`+presses("the", 1)+`}`)
	call(t, h, "POST", "/api/learn/reset", "")
	_, learn := call(t, h, "GET", "/api/learn/stats", "")
	_, free := call(t, h, "GET", "/api/freetype/stats?list=1000", "")
	if learn["unlocked"] != 6.0 || get(learn, "history", "count") != 0.0 || get(free, "history", "count") != 1.0 {
		t.Fatalf("Learn starts over, Freetype stays: %v %v", learn["unlocked"], free["history"])
	}
	call(t, h, "POST", "/api/reset", "")
	if _, free := call(t, h, "GET", "/api/freetype/stats?list=1000", ""); get(free, "history", "count") != 0.0 {
		t.Fatal("a reset deletes everything")
	}
}

func TestGuestsApart(t *testing.T) {
	h := newApp(t)
	callAs(t, h, "one", "POST", "/api/learn/results", result(presses("saldfe", 30), ""))
	if _, st := callAs(t, h, "two", "GET", "/api/learn/stats", ""); get(st, "history", "count") != 0.0 {
		t.Fatal("one guest sees another's rounds")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if c := rec.Result().Cookies(); len(c) != 1 || c[0].Name != "tt_guest" || !c[0].HttpOnly || len(c[0].Value) != 64 {
		t.Fatalf("a new browser gets a guest cookie: %v", c)
	}
}

func TestPageNotBuilt(t *testing.T) {
	rec := httptest.NewRecorder()
	page(fstest.MapFS{}).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "npm run build") {
		t.Fatalf("no page: %d %s", rec.Code, rec.Body)
	}
}
