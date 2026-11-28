package main

// Freetype through the whole app.

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApp_Freetype(t *testing.T) {
	h, _, _ := newApp()
	_, st := call(t, h, "GET", "/api/v1/freetype/stats?lang=en", "")
	lists := fmt.Sprint(st["lists"])
	if lists != "[200 1000 5000 10000 25000]" || st["list"] != float64(200) || st["sessionCount"] != float64(0) {
		t.Fatalf("new user: lists %v, list %v, rounds %v", lists, st["list"], st["sessionCount"])
	}
	code, out := call(t, h, "POST", "/api/v1/freetype/portion?lang=en", `{"start":true,"list":200,"words":10}`)
	if code != 200 || len(out["portions"].([]any)) != 2 || out["level"] != nil {
		t.Fatalf("first portions: %d %v", code, out)
	}
	for _, p := range out["portions"].([]any) {
		for _, w := range p.(map[string]any)["words"].([]any) {
			if w != strings.ToLower(w.(string)) || strings.ContainsAny(w.(string), ",.") {
				t.Fatalf("Freetype words are plain: %q", w)
			}
		}
	}
	for _, bad := range []string{`{"start":true,"list":300}`, `{"start":true}`, `{"start":true,"list":200,"words":101}`} {
		if code, _ := call(t, h, "POST", "/api/v1/freetype/portion?lang=en", bad); code != 400 {
			t.Fatalf("%s: a list not offered must be rejected", bad)
		}
	}
	round := func(list int, wpm float64) int {
		code, _ := call(t, h, "POST", "/api/v1/freetype/results?lang=en", fmt.Sprintf(
			`{"wpm":%v,"accuracy":98,"chars":150,"perSecond":[5,5,5,5,5,5],"list":%d,"keys":[{"char":"t","errors":0,"ms":200}]}`, wpm, list))
		return code
	}
	if round(1000, 80) != 200 || round(1000, 85) != 200 || round(25000, 60) != 200 {
		t.Fatal("results not stored")
	}
	if code := round(300, 80); code != 400 {
		t.Fatal("a result on a list not offered must be rejected")
	}
	if code, _ := call(t, h, "POST", "/api/v1/freetype/results?lang=en", `{"wpm":50,"accuracy":98,"chars":10,"list":200,"keys":[{"char":"ж","errors":0,"ms":200}]}`); code != 400 {
		t.Fatal("a letter of another alphabet is not a Freetype letter")
	}

	code, out = call(t, h, "POST", "/api/v1/freetype/portion?lang=en", `{"start":true,"list":1000,"variant":"long"}`)
	if code != 200 {
		t.Fatalf("long portions: %d %v", code, out)
	}
	for _, w := range out["portions"].([]any)[0].(map[string]any)["words"].([]any) {
		if len(w.(string)) < 6 {
			t.Fatalf("a round of long words has %q", w)
		}
	}
	code, out = call(t, h, "POST", "/api/v1/freetype/portion?lang=en", `{"start":true,"variant":"sentences"}`)
	if code != 200 {
		t.Fatalf("sentence portions: %d %v", code, out)
	}
	words := out["portions"].([]any)[0].(map[string]any)["words"].([]any)
	if first := words[0].(string); first[:1] != strings.ToUpper(first[:1]) || !strings.ContainsAny(words[len(words)-1].(string), ".?!") {
		t.Fatalf("a portion of sentences: %v", words)
	}
	for _, bad := range []string{`{"list":1000,"variant":"custom"}`, `{"list":1000,"variant":"punct"}`, `{"list":1000,"variant":"sentences"}`, `{"variant":"long"}`} {
		if code, _ := call(t, h, "POST", "/api/v1/freetype/portion?lang=en", bad); code != 400 {
			t.Fatalf("%s: your own text, unknown variants and a list where none belongs have no portions", bad)
		}
	}
	variant := func(list int, v string) int {
		code, _ := call(t, h, "POST", "/api/v1/freetype/results?lang=en", fmt.Sprintf(
			`{"wpm":70,"accuracy":97,"chars":20,"list":%d,"variant":%q,"keys":[{"char":"T","errors":0,"ms":200},{"char":"?","errors":0,"ms":200}]}`, list, v))
		return code
	}
	if variant(0, "sentences") != 200 || variant(0, "custom") != 200 || variant(1000, "custom") != 400 || variant(0, "long") != 400 {
		t.Fatal("variant results: sentences and custom need no list, long one")
	}
	_, pv := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&variant=sentences", "")
	_, cv := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&variant=custom", "")
	if pv["sessionCount"] != float64(1) || cv["sessionCount"] != float64(1) || cv["variant"] != "custom" {
		t.Fatalf("variant histories: sentences %v, custom %v", pv["sessionCount"], cv)
	}
	_, k := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&list=1000", "")
	_, big := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&list=25000", "")
	_, learn := call(t, h, "GET", "/api/v1/stats?lang=en", "")
	if k["sessionCount"] != float64(2) || big["sessionCount"] != float64(1) || learn["sessionCount"] != float64(0) {
		t.Fatalf("histories: 1k %v, 25k %v, learning %v", k["sessionCount"], big["sessionCount"], learn["sessionCount"])
	}
	if code, _ := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&list=7", ""); code != 400 {
		t.Fatal("stats of a list not offered must be rejected")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, guestRequest("GET", "/api/v1/export?lang=en", ""))
	file := rec.Body.String()
	if !strings.Contains(file, `"freetype":{"1000":`) || !strings.Contains(file, `"custom":`) || !strings.Contains(file, `"sentences":`) {
		t.Fatalf("export lacks Freetype: %.300s", file)
	}
	call(t, h, "POST", "/api/v1/reset?lang=en", "")
	if code, out := call(t, h, "POST", "/api/v1/import?lang=en", file); code != 200 {
		t.Fatalf("import: %d %v", code, out)
	}
	if _, k := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&list=1000", ""); k["sessionCount"] != float64(2) {
		t.Fatalf("import lost Freetype: %v", k["sessionCount"])
	}
	if _, cv := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&variant=custom", ""); cv["sessionCount"] != float64(1) {
		t.Fatalf("import lost your own text: %v", cv["sessionCount"])
	}
	bad := strings.Replace(file, `"freetype":{"1000":`, `"freetype":{"999":`, 1)
	if code, _ := call(t, h, "POST", "/api/v1/import?lang=en", bad); code != 400 {
		t.Fatal("an import with an unknown list must be rejected")
	}
}

// Freetype's loop through the app: the
// word list moves by consistency, three even rounds up and two uneven ones
// down, and the rhythm finds the pair typed with a stumble, from rounds on a
// keyboard only.
func TestApp_FreetypeRhythm(t *testing.T) {
	h, _, _ := newApp()
	stats := func(list int) map[string]any {
		t.Helper()
		_, st := call(t, h, "GET", fmt.Sprintf("/api/v1/freetype/stats?lang=en&list=%d", list), "")
		return st
	}
	if st := stats(200); st["standing"].(map[string]any)["level"] != 1.0 || st["rhythm"].(map[string]any)["presses"] != 0.0 {
		t.Fatalf("new user: list 1, no rhythm: %v %v", st["standing"], st["rhythm"])
	}
	// A round of "the quick " ten times: every press 150 ms, and "qu" 700 ms in half of them.
	keys := func() string {
		var b strings.Builder
		for i := range 10 {
			for j, c := range "the quick " {
				ms := 150
				if c == 'u' && i%2 == 0 {
					ms = 700
				}
				if j > 0 || i > 0 {
					b.WriteString(",")
				}
				fmt.Fprintf(&b, `{"char":%q,"errors":0,"ms":%d}`, string(c), ms)
			}
		}
		return b.String()
	}()
	round := func(list int, perSecond, device string) {
		t.Helper()
		body := fmt.Sprintf(`{"wpm":60,"accuracy":99,"chars":100,"perSecond":[%s],"list":%d,"device":%q,"keys":[%s]}`, perSecond, list, device, keys)
		if code, out := call(t, h, "POST", "/api/v1/freetype/results?lang=en", body); code != 200 {
			t.Fatalf("results: %d %v", code, out)
		}
	}
	even, uneven := "5,5,5,5,5,5", "1,9,1,9,1,9"
	for range 3 {
		round(200, even, "touch")
	}
	if st := stats(200); st["standing"].(map[string]any)["level"] != 1.0 || st["rhythm"].(map[string]any)["presses"] != 0.0 {
		t.Fatalf("rounds on a phone move nothing: %v %v", st["standing"], st["rhythm"])
	}
	round(200, even, "keyboard")
	round(200, even, "keyboard")
	if s := stats(200)["standing"].(map[string]any); s["level"] != 1.0 || s["passes"] != 2.0 {
		t.Fatalf("two even rounds: %v", s)
	}
	round(200, even, "keyboard")
	st := stats(200)
	if s := st["standing"].(map[string]any); s["level"] != 2.0 || s["moved"] != 1.0 {
		t.Fatalf("the third even round moves the list up: %v", s)
	}
	pairs := st["rhythm"].(map[string]any)["pairs"].([]any)
	if len(pairs) != 1 || pairs[0].(map[string]any)["pair"] != "qu" {
		t.Fatalf("the stumble pair: %v", st["rhythm"])
	}
	round(1000, uneven, "keyboard")
	round(1000, uneven, "keyboard")
	if s := stats(1000)["standing"].(map[string]any); s["level"] != 1.0 || s["moved"] != -1.0 {
		t.Fatalf("two uneven rounds move the list down: %v", s)
	}
	if _, st := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&variant=sentences", ""); st["standing"] != nil {
		t.Fatalf("sentences have no list to move: %v", st["standing"])
	}
}

// The words a typist owns, through the app: "the" and "and" typed clean and
// even in three Freetype rounds on a keyboard are owned, in Freetype's
// stats and in Reinforce's; phone rounds own nothing.
func TestApp_OwnedWords(t *testing.T) {
	h, _, _ := newApp()
	var b strings.Builder
	for i := range 10 {
		for j, c := range "the and " {
			if i > 0 || j > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, `{"char":%q,"errors":0,"ms":150}`, string(c))
		}
	}
	round := func(device string) {
		t.Helper()
		body := fmt.Sprintf(`{"wpm":60,"accuracy":100,"chars":80,"perSecond":[5,5,5,5,5,5],"list":200,"device":%q,"keys":[%s]}`, device, b.String())
		if code, out := call(t, h, "POST", "/api/v1/freetype/results?lang=en", body); code != 200 {
			t.Fatalf("results: %d %v", code, out)
		}
	}
	owned := func(path string) string {
		_, st := call(t, h, "GET", path, "")
		return fmt.Sprint(st["owned"])
	}
	for range 3 {
		round("touch")
	}
	round("keyboard")
	round("keyboard")
	if got := owned("/api/v1/freetype/stats?lang=en&list=200"); got != "[0 0 0 0 0]" {
		t.Fatalf("two keyboard rounds (and three on a phone) own nothing: %s", got)
	}
	round("keyboard")
	if got := owned("/api/v1/freetype/stats?lang=en&list=200"); got != "[2 2 2 2 2]" {
		t.Fatalf("the and and are owned in every list: %s", got)
	}
	if got := owned("/api/v1/stats?lang=en&part=reinforce"); !strings.HasPrefix(got, "[2 2 ") {
		t.Fatalf("Reinforce counts the same words: %s", got)
	}
	if _, st := call(t, h, "GET", "/api/v1/freetype/stats?lang=en&variant=sentences", ""); st["owned"] != nil {
		t.Fatalf("sentences have no list to own: %v", st["owned"])
	}
	if code, _ := call(t, h, "POST", "/api/v1/freetype/portion?lang=en", `{"start":true,"list":200,"words":10}`); code != 200 {
		t.Fatalf("portion with owned words: %d", code)
	}
	// A sprint brings the owned words far more often, each saying why it
	// came. Two owned words are not repeated next to each other, so they
	// cannot be the whole round; of 200 words they are 1 % of a usual draw.
	ownedShare := func(kind string) float64 {
		mine, all := 0, 0
		for range 5 {
			_, out := call(t, h, "POST", "/api/v1/freetype/portion?lang=en", fmt.Sprintf(`{"start":true,"list":200,"words":40,"kind":%q}`, kind))
			for _, p := range out["portions"].([]any) {
				p := p.(map[string]any)
				for i, w := range p["words"].([]any) {
					all++
					if w != "the" && w != "and" {
						continue
					}
					mine++
					if kind == "sprint" && p["why"].([]any)[i] != "owned:" {
						t.Fatalf("an owned word in a sprint: why %v", p["why"].([]any)[i])
					}
				}
			}
		}
		return float64(mine) / float64(max(all, 1))
	}
	if sprint, usual := ownedShare("sprint"), ownedShare(""); sprint < 0.1 || sprint < 5*usual {
		t.Fatalf("owned words: %.2f in a sprint, %.2f in a usual round", sprint, usual)
	}
}
