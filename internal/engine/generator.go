package engine

// What the user types: letter order, unlocking and word choice.

import (
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

type Generator struct {
	words []string
	rank  map[string]int
	order []string
	start int
	wp    WordParams
	floor map[string]int
	rng   *rand.Rand
	pairs []string
	weak  string // the letters of the typist's weak finger
	owned map[string]bool
	bases *bases
}

type WordParams struct {
	FloorWords  int
	RampLetters float64
}

func (g *Generator) SetWordParams(wp WordParams) {
	g.wp = wp
	g.floor = map[string]int{}
	if wp.FloorWords <= 0 {
		return
	}
	count := map[rune]int{}
	for i, w := range g.words {
		seen := map[rune]bool{}
		for _, r := range w {
			if seen[r] {
				continue
			}
			seen[r] = true
			if count[r] < wp.FloorWords {
				count[r]++
				g.floor[string(r)] = i
			}
		}
	}
}

func NewGenerator(words []string, rng *rand.Rand, p Params) *Generator {
	if p.MinWordLetters > 1 {
		long := make([]string, 0, len(words))
		for _, w := range words {
			if utf8.RuneCountInString(w) >= p.MinWordLetters {
				long = append(long, w)
			}
		}
		words = long
	}
	g := &Generator{words: words, rng: rng, rank: map[string]int{}, bases: newBases()}
	for i, w := range words {
		if _, ok := g.rank[w]; !ok {
			g.rank[w] = i
		}
	}
	g.order = letterOrder(words, nil)
	g.start = g.firstLessonSize(p)
	return g
}

// SetHome puts the letters under the resting fingers first, then every
// other letter by how often it occurs. The first lesson is StartLetters
// letters: the most common resting ones and the most common other letter
// (a vowel in practice, without which there are hardly any words), more if
// needed for MinWords words; the other resting letters come next.
func (g *Generator) SetHome(home []string, p Params) {
	set := map[string]bool{}
	for _, c := range home {
		set[c] = true
	}
	g.order = letterOrder(g.words, set)
	g.bases = newBases()
	homes := 0
	for homes < len(g.order) && set[g.order[homes]] {
		homes++
	}
	if at := p.StartLetters - 1; at >= 0 && at < homes && homes < len(g.order) {
		first := g.order[homes]
		copy(g.order[at+1:homes+1], g.order[at:homes])
		g.order[at] = first
	}
	g.start = g.firstLessonSize(p)
}

func letterOrder(words []string, home map[string]bool) []string {
	count := map[string]float64{}
	for _, w := range words {
		for _, r := range w {
			count[string(r)]++
		}
	}
	order := make([]string, 0, len(count))
	for c := range count {
		order = append(order, c)
	}
	sort.Slice(order, func(i, j int) bool {
		if home[order[i]] != home[order[j]] {
			return home[order[i]]
		}
		if count[order[i]] != count[order[j]] {
			return count[order[i]] > count[order[j]]
		}
		return order[i] < order[j]
	})
	return order
}

func (g *Generator) firstLessonSize(p Params) int {
	n := p.StartLetters
	if n > len(g.order) {
		n = len(g.order)
	}
	for n < len(g.order) && len(g.eligible(n)) < p.MinWords {
		n++
	}
	return n
}

// WithWords is this generator over another word list, most frequent first:
// a Reinforce word level. The letter order stays; words with a letter
// outside it are dropped.
func (g *Generator) WithWords(words []string, p Params) *Generator {
	in := map[rune]bool{}
	for _, c := range g.order {
		in[letterRune(c)] = true
	}
	keep := make([]string, 0, len(words))
	for _, w := range words {
		ok := utf8.RuneCountInString(w) >= max(p.MinWordLetters, 1)
		for _, r := range w {
			ok = ok && in[r]
		}
		if ok {
			keep = append(keep, w)
		}
	}
	out := *g
	out.words, out.rank, out.bases = keep, map[string]int{}, newBases()
	for i, w := range keep {
		if _, ok := out.rank[w]; !ok {
			out.rank[w] = i
		}
	}
	out.SetWordParams(g.wp)
	return &out
}

// Size is how many words the generator draws from.
func (g *Generator) Size() int { return len(g.words) }

// WithPairs is this generator with letter pairs that slow the learner down:
// words with them come more often (PairBoost).
func (g *Generator) WithPairs(pairs []string) *Generator {
	if len(pairs) == 0 {
		return g
	}
	out := *g
	out.pairs = pairs
	return &out
}

// WithOwned is this generator with the words the learner owns: they come
// less often (OwnedPenalty).
func (g *Generator) WithOwned(owned map[string]bool) *Generator {
	if len(owned) == 0 {
		return g
	}
	out := *g
	out.owned = owned
	return &out
}

// WithFinger is this generator with the letters of the learner's weak
// finger: words made of them by FingerShare or more come more often
// (FingerBoost).
func (g *Generator) WithFinger(letters []string) *Generator {
	if len(letters) == 0 {
		return g
	}
	out := *g
	out.weak = strings.Join(letters, "")
	return &out
}

func (g *Generator) Order() []string { return g.order }

func (g *Generator) Start() int { return g.start }

func (g *Generator) Clamp(unlocked int) int {
	if unlocked < g.start {
		return g.start
	}
	if unlocked > len(g.order) {
		return len(g.order)
	}
	return unlocked
}

func (g *Generator) NextUnlocked(stats map[string]KeyStat, unlocked int, p Params) int {
	unlocked = g.Clamp(unlocked)
	if unlocked == len(g.order) {
		return unlocked
	}
	if p.accuracyRules() {
		if Accuracy(stats[g.order[unlocked-1]], p) < p.AccMaster {
			return unlocked
		}
		for _, c := range g.order[:unlocked] {
			if Accuracy(stats[c], p) < p.AccSlip {
				return unlocked
			}
		}
		return unlocked + 1
	}
	for _, c := range g.order[:unlocked] {
		if !Mastered(stats[c], p) {
			return unlocked
		}
	}
	return unlocked + 1
}

// FocusLetter is the focus of the accuracy rules (section 14.3): in the
// first lesson the lowest letter under AccMaster; later a slipped letter,
// the lowest first, else the newest letter.
func (g *Generator) FocusLetter(stats map[string]KeyStat, unlocked int, p Params) string {
	unlocked = g.Clamp(unlocked)
	open := g.order[:unlocked]
	lowest := func(keep func(KeyStat) bool) string {
		best, bestAcc := "", 2.0
		for _, c := range open {
			if a := Accuracy(stats[c], p); keep(stats[c]) && a < bestAcc {
				best, bestAcc = c, a
			}
		}
		return best
	}
	if unlocked <= g.start {
		if c := lowest(func(s KeyStat) bool { return Accuracy(s, p) < p.AccMaster }); c != "" {
			return c
		}
		return lowest(func(KeyStat) bool { return true })
	}
	newest := open[unlocked-1]
	open = open[:unlocked-1]
	if c := lowest(func(s KeyStat) bool { return s.Slipped || Accuracy(s, p) < p.AccSlip }); c != "" {
		return c
	}
	return newest
}

func (g *Generator) Words(unlocked int) []string { return g.eligible(g.Clamp(unlocked)) }

func letterRune(c string) rune {
	r, _ := utf8.DecodeRuneInString(c)
	return r
}

func (g *Generator) eligible(n int) []string {
	open := map[rune]bool{}
	for _, c := range g.order[:n] {
		open[letterRune(c)] = true
	}
	var out []string
	for _, w := range g.words {
		ok := true
		for _, r := range w {
			if !open[r] {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, w)
		}
	}
	return out
}

type Exercise struct {
	Text     string   `json:"text"`
	Focus    string   `json:"focus"`
	Unlocked []string `json:"unlocked"`
	Locked   []string `json:"locked"`
}

type pool struct {
	open     []string
	words    []string
	index    map[string]int
	letterW  []float64
	byLetter map[string][]int
	diffW    []float64
	rank     []int
	maxRank  float64
	floor    map[string]int
	weakest  string
}

// bases keeps the part of a word pool that depends only on how many letters
// are open: the words, their places, which words have each letter and each
// word's letters. It is built the first time a number of open letters is
// asked for and kept; copies with the same words (WithPairs) share it.
type bases struct {
	mu sync.Mutex
	by map[int]*poolBase
}

type poolBase struct {
	words    []string
	index    map[string]int
	byLetter map[string][]int
	letters  [][]uint8 // each word's letters, as places in the open letters
	rank     []int
}

func newBases() *bases { return &bases{by: map[int]*poolBase{}} }

func (g *Generator) base(unlocked int) *poolBase {
	g.bases.mu.Lock()
	defer g.bases.mu.Unlock()
	if b := g.bases.by[unlocked]; b != nil {
		return b
	}
	place := map[rune]uint8{}
	for i, c := range g.order[:unlocked] {
		place[letterRune(c)] = uint8(i)
	}
	b := &poolBase{words: g.eligible(unlocked), index: map[string]int{}, byLetter: map[string][]int{}}
	b.letters = make([][]uint8, len(b.words))
	b.rank = make([]int, len(b.words))
	for i, w := range b.words {
		b.index[w] = i
		b.rank[i] = g.rank[w]
		seen := map[rune]bool{}
		for _, r := range w {
			b.letters[i] = append(b.letters[i], place[r])
			if !seen[r] {
				seen[r] = true
				b.byLetter[string(r)] = append(b.byLetter[string(r)], i)
			}
		}
	}
	g.bases.by[unlocked] = b
	return b
}

func (g *Generator) newPool(stats map[string]KeyStat, unlocked, typed int, p Params) *pool {
	b := g.base(unlocked)
	pl := &pool{open: g.order[:unlocked], words: b.words, index: b.index, byLetter: b.byLetter, rank: b.rank,
		maxRank: g.MaxRank(unlocked, typed, p), floor: g.floor}
	if n := p.MinWords; n > 0 && len(pl.words) > 0 {
		pl.maxRank = max(pl.maxRank, float64(g.rank[pl.words[min(n, len(pl.words))-1]]))
	}
	diff := make([]float64, len(pl.open))
	pl.letterW = make([]float64, len(pl.open))
	best := -1.0
	for i, c := range pl.open {
		d := Difficulty(stats[c], p)
		if !Mastered(stats[c], p) {
			d += p.UnmasteredBonus
		}
		diff[i] = d
		pl.letterW[i] = SoftmaxWeight(d, p)
		if d > best {
			best, pl.weakest = d, c
		}
	}
	pl.diffW = make([]float64, len(pl.words))
	for i, w := range pl.words {
		sum := 0.0
		for _, l := range b.letters[i] {
			sum += diff[l]
		}
		pl.diffW[i] = SoftmaxWeight(sum/float64(len(w)), p)
		for _, pair := range g.pairs {
			if strings.Contains(w, pair) {
				pl.diffW[i] *= 1 + p.PairBoost
			}
		}
		if g.owned[w] {
			pl.diffW[i] *= p.OwnedPenalty
		}
		if g.weak != "" && len(w) > 0 {
			on := 0
			for _, c := range w {
				if strings.ContainsRune(g.weak, c) {
					on++
				}
			}
			if float64(on) >= p.FingerShare*float64(utf8.RuneCountInString(w)) {
				pl.diffW[i] *= 1 + p.FingerBoost
			}
		}
	}
	return pl
}

func (pl *pool) letterMaxRank(letter string) float64 {
	floor, ok := pl.floor[letter]
	if !ok {
		return pl.maxRank
	}
	return max(pl.maxRank, float64(floor))
}

func (g *Generator) Rank(word string) int {
	if r, ok := g.rank[word]; ok {
		return r
	}
	return -1
}

func (g *Generator) MaxRank(unlocked, typed int, p Params) float64 {
	start, end := float64(p.RankStart), float64(len(g.words))
	if end < start {
		end = start
	}
	if g.Clamp(unlocked) < len(g.order) {
		return start
	}
	progress := 1.0
	if g.wp.RampLetters > 0 {
		progress = min(1, max(0, float64(typed)/g.wp.RampLetters))
	}
	return start + (end-start)*progress
}

func RankWeight(rank int, maxRank float64, p Params) float64 {
	over := float64(rank) - maxRank
	if over <= 0 || p.RankStep <= 0 || p.RankPenalty <= 0 || p.RankPenalty >= 1 {
		return 1
	}
	return math.Pow(p.RankPenalty, over/p.RankStep)
}

func (pl *pool) pickFocus(r float64) string { return pl.open[WeightedPick(pl.letterW, r)] }

func (pl *pool) pickWord(focus string, uses []int, p Params, r float64, avoid ...int) int {
	neighbour := func(idx int) bool { return contains(avoid, idx) }
	used := func(idx int) bool { return neighbour(idx) || uses[idx] > 0 }
	capped := func(idx int) bool { return p.MaxRepeats > 0 && uses[idx] >= p.MaxRepeats }
	all := make([]int, len(pl.words))
	for i := range all {
		all[i] = i
	}
	tries := []struct {
		cands   []int
		maxRank float64
		banned  func(int) bool
	}{
		{pl.byLetter[focus], pl.letterMaxRank(focus), used},
		{all, pl.maxRank, used},
		{all, pl.maxRank, func(i int) bool { return neighbour(i) || capped(i) }},
		{all, pl.maxRank, neighbour},
		{all, pl.maxRank, func(int) bool { return false }},
	}
	for _, try := range tries {
		w := make([]float64, len(try.cands))
		total := 0.0
		for j, idx := range try.cands {
			if try.banned(idx) {
				continue
			}
			w[j] = pl.diffW[idx] * RankWeight(pl.rank[idx], try.maxRank, p)
			for u := 0; u < uses[idx]; u++ {
				w[j] *= p.RepeatPenalty
			}
			total += w[j]
		}
		if total > 0 {
			return try.cands[WeightedPick(w, r)]
		}
	}
	return 0
}

func (g *Generator) letters(unlocked int) (open, locked []string) {
	return append([]string{}, g.order[:unlocked]...), append([]string{}, g.order[unlocked:]...)
}

func (g *Generator) Next(stats map[string]KeyStat, unlocked, typed int, p Params) Exercise {
	unlocked = g.Clamp(unlocked)
	ex := Exercise{}
	ex.Unlocked, ex.Locked = g.letters(unlocked)
	pl := g.newPool(stats, unlocked, typed, p)
	if len(pl.words) == 0 {
		return ex
	}
	ex.Focus = pl.weakest

	uses := make([]int, len(pl.words))
	out := make([]string, 0, p.ExerciseWords)
	last := -1
	for len(out) < p.ExerciseWords {
		focus := pl.pickFocus(g.rng.Float64())
		idx := pl.pickWord(focus, uses, p, g.rng.Float64(), last)
		uses[idx]++
		last = idx
		out = append(out, pl.words[idx])
	}
	ex.Text = strings.Join(out, " ")
	return ex
}

type Portion struct {
	Words    []string `json:"words"`
	Focus    string   `json:"focus"`
	Weakest  string   `json:"weakest"`
	Unlocked []string `json:"unlocked"`
	Locked   []string `json:"locked"`
}

func (g *Generator) Portion(stats map[string]KeyStat, unlocked, typed int, recent, retry []string, p Params) Portion {
	unlocked = g.Clamp(unlocked)
	por := Portion{}
	por.Unlocked, por.Locked = g.letters(unlocked)
	pl := g.newPool(stats, unlocked, typed, p)
	if len(pl.words) == 0 || p.PortionWords <= 0 {
		return por
	}
	por.Weakest = pl.weakest
	if p.accuracyRules() && unlocked < len(g.order) {
		por.Focus = g.FocusLetter(stats, unlocked, p)
	} else {
		por.Focus = pl.pickFocus(g.rng.Float64())
	}

	uses := make([]int, len(pl.words))
	if len(recent) > p.RecentWords {
		recent = recent[len(recent)-p.RecentWords:]
	}
	last := -1
	for _, w := range recent {
		if idx, ok := pl.index[w]; ok {
			uses[idx]++
			last = idx
		} else {
			last = -1
		}
	}

	slots := make([]int, p.PortionWords)
	for i := range slots {
		slots[i] = -1
	}
	placed := 0
	for _, w := range retry {
		idx, ok := pl.index[w]
		if !ok || placed >= p.RetryWords || contains(slots, idx) || (p.MaxRepeats > 0 && uses[idx] >= p.MaxRepeats) {
			continue
		}
		free := []int{}
		for i, s := range slots {
			if s == -1 && !(i == 0 && idx == last) {
				free = append(free, i)
			}
		}
		if len(free) == 0 {
			continue
		}
		slots[free[g.rng.Intn(len(free))]] = idx
		uses[idx]++
		placed++
	}

	for i := range slots {
		if slots[i] == -1 {
			next := -1
			if i+1 < len(slots) {
				next = slots[i+1]
			}
			slots[i] = pl.pickWord(por.Focus, uses, p, g.rng.Float64(), last, next)
			uses[slots[i]]++
		}
		last = slots[i]
	}
	por.Words = make([]string, len(slots))
	for i, idx := range slots {
		por.Words[i] = pl.words[idx]
	}
	return por
}

func contains(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func WeightedPick(weights []float64, r float64) int {
	if len(weights) == 0 {
		return -1
	}
	total := 0.0
	for _, w := range weights {
		if w > 0 {
			total += w
		}
	}
	if total == 0 {
		return int(r * float64(len(weights)))
	}
	point := r * total
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		if point < w {
			return i
		}
		point -= w
	}
	for i := len(weights) - 1; i >= 0; i-- {
		if weights[i] > 0 {
			return i
		}
	}
	return len(weights) - 1
}
