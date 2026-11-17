package engine

import (
	"math/rand"
	"slices"
	"strings"
	"testing"
)

var testWords = strings.Fields(`the to and a in is it you that he was for on are as with
at be this have from or one had by not but what all were we when your can said there
use an each which she do how their if will up other about out many then them these so
some her would make like him into time has look two more write go see number no way
could people my than first water been call who oil its now find long down day did get
come made may part over new sound take only little work know place year live me back
give most very after thing our just name good sentence man think say great where help
through much before line right too mean old any same tell boy follow came want show
also around form three small set put end does another well large must big even such
because turn here why ask went men read need land different home us move try kind hand`)

func newTestGen(seed int64) *Generator {
	return NewGenerator(testWords, rand.New(rand.NewSource(seed)), testParams)
}

func TestWeightedPick(t *testing.T) {
	w := []float64{1, 3}
	for _, c := range []struct {
		r    float64
		want int
	}{{0, 0}, {0.24, 0}, {0.25, 1}, {0.999, 1}} {
		if got := WeightedPick(w, c.r); got != c.want {
			t.Errorf("r=%v: got %d, want %d", c.r, got, c.want)
		}
	}
}

func TestWeightedPick_EdgeCases(t *testing.T) {
	if got := WeightedPick(nil, 0.5); got != -1 {
		t.Errorf("empty: got %d, want -1", got)
	}
	if got := WeightedPick([]float64{0, 0, 0, 0}, 0.6); got != 2 {
		t.Errorf("all zero (uniform fallback): got %d, want 2", got)
	}
	for _, r := range []float64{0, 0.3, 0.5, 0.999} {
		if WeightedPick([]float64{1, 0, 1}, r) == 1 {
			t.Errorf("r=%v picked a zero weight", r)
		}
	}
}

func TestLetterOrder_MostFrequentFirst(t *testing.T) {
	order := letterOrder([]string{"eee", "ee", "ta", "t"}, nil)
	if strings.Join(order, "") != "eta" {
		t.Fatalf("got %v, want e t a", order)
	}
}

func TestLetterOrder_HomeLettersFirst(t *testing.T) {
	words := []string{"eee", "eee", "ttt", "a", "ss"}
	if got := strings.Join(letterOrder(words, map[string]bool{"a": true, "s": true}), ""); got != "saet" {
		t.Errorf("got %s, want s a (home, by frequency), then e t", got)
	}
	if got := strings.Join(letterOrder(words, nil), ""); got != "etsa" {
		t.Errorf("no home: got %s, want e t s a", got)
	}
}

func TestSetHome_FirstLessonStillHasEnoughWords(t *testing.T) {
	g := newTestGen(1)
	plain := strings.Join(g.Order(), "")
	g.SetHome([]string{"a", "s", "d", "f", "j", "k", "l"}, testParams)
	if g.Order()[0] != "a" {
		t.Errorf("order starts with %q, want the home letter a", g.Order()[0])
	}
	if n := len(g.eligible(g.Start())); n < testParams.MinWords {
		t.Errorf("first lesson has only %d words", n)
	}
	g.SetHome(nil, testParams)
	if got := strings.Join(g.Order(), ""); got != plain {
		t.Errorf("no home letters: order %s, want the plain %s", got, plain)
	}
}

func TestFirstLesson_HasEnoughRealWords(t *testing.T) {
	g := newTestGen(1)
	if g.Start() < testParams.StartLetters {
		t.Fatalf("start = %d, below StartLetters", g.Start())
	}
	if n := len(g.eligible(g.Start())); n < testParams.MinWords {
		t.Fatalf("first lesson has only %d words", n)
	}
}

func TestNext_UsesOnlyOpenLetters(t *testing.T) {
	g := newTestGen(2)
	ex := g.Next(nil, 0, 0, testParams)
	open := strings.Join(ex.Unlocked, "") + " "
	for _, r := range ex.Text {
		if !strings.ContainsRune(open, r) {
			t.Fatalf("letter %q is locked but appeared in %q", r, ex.Text)
		}
	}
	if n := len(strings.Fields(ex.Text)); n != testParams.ExerciseWords {
		t.Fatalf("got %d words, want %d", n, testParams.ExerciseWords)
	}
	if len(ex.Unlocked)+len(ex.Locked) != len(g.Order()) {
		t.Fatal("unlocked + locked must cover every letter")
	}
}

func TestNext_NoWordTwiceInARow(t *testing.T) {
	g := newTestGen(3)
	for i := 0; i < 50; i++ {
		w := strings.Fields(g.Next(nil, 0, 0, testParams).Text)
		for j := 1; j < len(w); j++ {
			if w[j] == w[j-1] {
				t.Fatalf("repeated word %q in %v", w[j], w)
			}
		}
	}
}

func TestNext_WeakLetterGetsFocus(t *testing.T) {
	all := len(newTestGen(0).Order())
	good := repeat(KeyStat{}, Sample{Ms: 250}, 60)
	bad := repeat(KeyStat{}, Sample{Errors: 1}, 20)
	bad = repeat(bad, Sample{Ms: 800}, 20)

	share := func(stats map[string]KeyStat, letter string) float64 {
		g := newTestGen(42)
		hit, total := 0, 0
		for i := 0; i < 200; i++ {
			for _, r := range g.Next(stats, all, 0, testParams).Text {
				if r != ' ' {
					total++
					if string(r) == letter {
						hit++
					}
				}
			}
		}
		return float64(hit) / float64(total)
	}

	even := map[string]KeyStat{}
	for _, c := range newTestGen(0).Order() {
		even[c] = good
	}
	weak := map[string]KeyStat{}
	for k, v := range even {
		weak[k] = v
	}
	weak["w"] = bad

	before, after := share(even, "w"), share(weak, "w")
	if after < before*2 {
		t.Fatalf("'w' share went from %.3f to %.3f, expected at least 2x", before, after)
	}
	if ex := newTestGen(1).Next(weak, all, 0, testParams); ex.Focus != "w" {
		t.Fatalf("focus = %q, want w", ex.Focus)
	}
}

func TestNextUnlocked(t *testing.T) {
	g := newTestGen(4)
	start := g.Start()
	stats := map[string]KeyStat{}
	if got := g.NextUnlocked(stats, 0, testParams); got != start {
		t.Fatalf("no stats: got %d, want %d (nothing mastered yet)", got, start)
	}
	mastered := repeat(KeyStat{}, Sample{Ms: 250}, 30)
	for _, c := range g.Order()[:start] {
		stats[c] = mastered
	}
	if got := g.NextUnlocked(stats, start, testParams); got != start+1 {
		t.Fatalf("all mastered: got %d, want %d", got, start+1)
	}
	stats[g.Order()[0]] = repeat(KeyStat{}, Sample{Errors: 1}, 30)
	if got := g.NextUnlocked(stats, start, testParams); got != start {
		t.Fatalf("one weak letter: got %d, want %d", got, start)
	}
	if got := g.NextUnlocked(stats, 999, testParams); got != len(g.Order()) {
		t.Fatalf("clamp: got %d", got)
	}
}

func TestNext_MultiByteLetters(t *testing.T) {
	for name, words := range map[string][]string{
		"polish":    strings.Fields("się że jest już tak nie mnie coś będzie wiem pan może są tylko ją ta za na"),
		"ukrainian": strings.Fields("я не що на це ти так у і з а ми все тебе він як за ви до мене їх її"),
		"russian":   strings.Fields("я не что в и ты это на с он мы как вы да все мне у нет меня так но"),
	} {
		p := testParams
		p.StartLetters, p.MinWords = 4, 3
		g := NewGenerator(words, rand.New(rand.NewSource(7)), p)
		for _, c := range g.Order() {
			if n := len([]rune(c)); n != 1 {
				t.Fatalf("%s: %q is %d letters, want 1", name, c, n)
			}
		}
		ex := g.Next(nil, 0, 0, p)
		if ex.Text == "" {
			t.Fatalf("%s: empty exercise", name)
		}
		open := strings.Join(ex.Unlocked, "") + " "
		for _, r := range ex.Text {
			if !strings.ContainsRune(open, r) {
				t.Fatalf("%s: locked letter %q in %q", name, r, ex.Text)
			}
		}
		all := g.Next(nil, len(g.Order()), 0, p)
		if len(strings.Fields(all.Text)) != p.ExerciseWords {
			t.Fatalf("%s: got %q", name, all.Text)
		}
	}
}

func TestLetterRune(t *testing.T) {
	for in, want := range map[string]rune{"e": 'e', "ą": 'ą', "ж": 'ж', "ї": 'ї'} {
		if got := letterRune(in); got != want {
			t.Errorf("letterRune(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnmasteredBonus_SlowLetterGetsTheFocus(t *testing.T) {
	g := newTestGen(1)
	open := g.Order()[:10]
	stats := map[string]KeyStat{}
	for _, c := range open {
		stats[c] = KeyStat{Count: 200, TimeEMA: 700, TimedCount: 200, Held: true}
	}
	slow := open[3]
	stats[slow] = KeyStat{Count: 200, TimeEMA: 730, TimedCount: 200}
	share := func(p Params) float64 {
		pl := g.newPool(stats, 10, 0, p)
		total := 0.0
		for _, w := range pl.letterW {
			total += w
		}
		return pl.letterW[3] / total
	}
	off := testParams
	off.UnmasteredBonus = 0
	with, without := share(testParams), share(off)
	if with < 0.4 {
		t.Errorf("slow letter gets %.0f%% of the focus with the bonus, want at least 40%%", with*100)
	}
	if without > 0.2 {
		t.Errorf("without the bonus it gets %.0f%%: the test no longer shows the problem", without*100)
	}
}

func TestWithWords_ALevelDrawsOnlyItsWords(t *testing.T) {
	g := newTestGen(1)
	level := g.WithWords([]string{"then", "the", "a", "x9", "when", "the"}, testParams)
	if level.Size() != 4 || g.Size() == 4 {
		t.Fatalf("level size %d (want then, the, when, the: no 1-letter word, no foreign letter), base %d", level.Size(), g.Size())
	}
	all := len(g.Order())
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		for _, w := range level.Portion(nil, all, 1<<30, nil, nil, testParams).Words {
			seen[w] = true
		}
	}
	for w := range seen {
		if w != "then" && w != "the" && w != "when" {
			t.Errorf("a word from outside the level: %q", w)
		}
	}
	if !slices.Equal(level.Order(), g.Order()) {
		t.Error("the letter order changed")
	}
}

func TestWithPairs_WeakPairsComeMoreOften(t *testing.T) {
	g := newTestGen(3).WithWords([]string{"they", "them", "want", "went", "fold", "gold"}, testParams)
	p := testParams
	p.PairBoost = 20
	count := func(gen *Generator) (n int) {
		for i := 0; i < 200; i++ {
			for _, w := range gen.Portion(nil, len(gen.Order()), 1<<30, nil, nil, p).Words {
				if strings.Contains(w, "th") {
					n++
				}
			}
		}
		return
	}
	plain, weak := count(g), count(g.WithPairs([]string{"th"}))
	if weak <= plain*3/2 {
		t.Errorf("words with the weak pair: %d with it, %d without", weak, plain)
	}
	if g.WithPairs(nil) != g {
		t.Error("no pairs: the same generator")
	}
}

func TestSetHome_MostCommonOtherLetterClosesTheFirstLesson(t *testing.T) {
	words := []string{"eee", "eee", "ttt", "aaaa", "sss", "dd", "ff"}
	g := NewGenerator(words, newRng(1), testParams)
	p := testParams
	p.StartLetters, p.MinWords = 3, 0
	g.SetHome([]string{"a", "s", "d", "f"}, p)
	if got := strings.Join(g.Order(), ""); got != "asedft" {
		t.Errorf("got %s, want a s (home), e (most common other), then d f, then t", got)
	}
	p.StartLetters = 5
	g.SetHome([]string{"a", "s", "d", "f"}, p)
	if got := strings.Join(g.Order(), ""); got != "asdfet" {
		t.Errorf("a first lesson past the home letters: got %s, want the plain asdfet", got)
	}
}
