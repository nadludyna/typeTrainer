// The text is a stream of short portions, a few words each, built while the
// typist types. Learn and Reinforce build theirs from the letter model here;
// Freetype draws its own.

package trainer

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"typingtrainer/internal/engine"
	"typingtrainer/internal/profile"
	"typingtrainer/internal/storage"
)

// PortionRequest asks for the next portions of a round.
type PortionRequest struct {
	Round  string   `json:"round"`  // the browser's round id
	Index  int      `json:"index"`  // how many portions of the round were finished
	Ack    int      `json:"ack"`    // how many the browser has received
	Start  bool     `json:"start"`  // the round's first request: the portion and those waiting after it
	Last   bool     `json:"last"`   // the round has all its portions
	Recent []string `json:"recent"` // the words of the round so far
	Retry  []string `json:"retry"`  // words typed with a mistake, to come back
	Kind   string   `json:"kind"`
	Level  int      `json:"level"` // Reinforce's word level, 1..
}

// Portions is the answer: the portions, and Learn's round length in portions.
type Portions struct {
	Portions []engine.Portion `json:"portions"`
	Round    int              `json:"round"`
}

// Check is what is wrong with the request, "" when nothing is.
func (t *Trainer) CheckPortion(req PortionRequest) string {
	n := t.Cfg.MaxEvents
	switch {
	case len(req.Recent) > n || len(req.Retry) > n:
		return "recent or retry: too long"
	case !ValidID(req.Round):
		return "round: at most 64 letters, digits and -"
	case req.Index < 0 || req.Index > n || req.Ack < 0 || req.Ack > n:
		return "index or ack: out of range"
	case !storage.ValidKind(req.Kind):
		return "kind: unknown kind of round"
	}
	return ""
}

// LetterPortions builds Learn's (reinforce false) or Reinforce's portions
// from the letter model as it is: the round's presses reach it only at the
// end, so every portion of a round reads the same numbers.
func (t *Trainer) LetterPortions(user string, req PortionRequest, reinforce bool) (Portions, error) {
	if out, ok := t.portions.get(user, req); ok {
		return out, nil
	}
	t.Write.Lock()
	defer t.Write.Unlock()
	now, err := t.Learn(user, nil, reinforce)
	if err != nil {
		return Portions{}, err
	}
	gen, all := t.Words.Gen, len(t.Words.Gen.Order())
	open := now.Progress.Unlocked
	if reinforce {
		open = all
	}
	p := t.Params(now.Lap).ForStage(open, all)
	typed := 0
	if reinforce {
		gen, typed = t.Words.Levels[req.Level-1], math.MaxInt32
		switch req.Kind {
		case storage.KindWeak:
			p.PairBoost, p.FingerBoost = t.Cfg.WeakBoost, t.Cfg.WeakBoost
		case storage.KindSprint:
			p.OwnedBoost = t.Cfg.WeakBoost
		}
	} else if open == all {
		if typed, err = t.lettersTyped(user, all); err != nil {
			return Portions{}, err
		}
	}
	owned, seen, err := t.Vocabulary(user, req.Index == 0)
	if err != nil {
		return Portions{}, err
	}
	returning := profile.Returning(seen, gen.Order()[:min(open, len(gen.Order()))], time.Now(), t.Cfg.Time)
	spots, err := t.letterSpots(user, returning)
	if err != nil {
		return Portions{}, err
	}
	gen = gen.WithPairs(append(slices.Clone(returning), spots.Pairs...)).WithFinger(strings.Split(spots.Letters, ""))
	if reinforce {
		gen = gen.WithOwned(owned.Owned)
		if req.Kind == storage.KindSprint {
			spots.Owned = owned.Owned
		}
	}
	count := 1
	if req.Last {
		count = 0
	} else if req.Start {
		count = min(p.AheadPortions+1, p.RoundPortions)
	}
	out := Portions{Portions: make([]engine.Portion, count), Round: p.RoundPortions}
	recent, retry := slices.Clone(req.Recent), req.Retry
	t.gen.Lock()
	for i := range out.Portions {
		out.Portions[i] = gen.Portion(now.Progress.Letters, open, typed, recent, retry, p)
		out.Portions[i].Why = spots.Whys(out.Portions[i].Words, out.Portions[i].Focus)
		recent = append(recent, out.Portions[i].Words...)
		retry = nil
	}
	t.gen.Unlock()
	t.portions.put(user, req, out)
	return out, nil
}

// letterSpots is the weak spots of the typist's last learning rounds: the
// letters coming back, the slow pairs and the weak finger.
func (t *Trainer) letterSpots(user string, returning []string) (Spots, error) {
	rounds, err := t.Rounds(user, Learning, max(t.Cfg.Levels.WeakRounds, t.Cfg.Profile.Rounds))
	if err != nil {
		return Spots{}, err
	}
	s := Spots{Back: returning, Share: t.Cfg.Engine.FingerShare}
	for _, p := range t.Pairs(rounds[:min(len(rounds), t.Cfg.Levels.WeakRounds)]) {
		s.Pairs = append(s.Pairs, p.Pair)
	}
	pr := t.Profile(rounds)
	s.Finger, s.Letters = pr.Weak, strings.Join(pr.Letters, "")
	return s, nil
}

// WeakSpots is how many weak spots a weak-spots round of Reinforce would be
// built from: the slow pairs and the weak finger.
func (t *Trainer) WeakSpots(rounds []storage.Round) int {
	n := len(t.Pairs(rounds[:min(len(rounds), t.Cfg.Levels.WeakRounds)]))
	if t.Profile(rounds).Weak != "" {
		n++
	}
	return n
}

// lettersTyped is the letters of the Learn rounds typed with every letter
// open: how far the ramp to rarer words has come.
func (t *Trainer) lettersTyped(user string, all int) (int, error) {
	rounds, err := t.Store.Rounds(user, storage.ModeLearn, 0)
	n := 0
	for _, r := range rounds {
		if r.Unlocked >= all {
			n += r.Chars
		}
	}
	return n, err
}

// ---------- a portion answer kept for a request sent twice ----------

// portionCache keeps each answer by (typist, round, index), so a request a
// flaky network sends twice gets the same portions. It lives in the process
// memory only: a restart forgets it, and a retry after one is a first
// request. Acked answers go at once, abandoned rounds after the TTL and the
// size cap in the config.
type portionCache struct {
	mu      sync.Mutex
	entries map[string]*cachedPortion
	ttl     time.Duration
	cap     int
}

type cachedPortion struct {
	user, round string
	from, count int
	out         Portions
	at          time.Time
}

func cacheKey(user string, req PortionRequest) string {
	return user + "\x00" + req.Round + "\x00" + strconv.Itoa(req.Index)
}

func (c *portionCache) get(user string, req PortionRequest) (Portions, bool) {
	if req.Round == "" {
		return Portions{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.entries {
		if e.user == user && e.round == req.Round && e.from < req.Ack && e.from+e.count <= req.Ack {
			delete(c.entries, k)
		}
	}
	e, ok := c.entries[cacheKey(user, req)]
	if !ok {
		return Portions{}, false
	}
	e.at = time.Now()
	return e.out, true
}

func (c *portionCache) put(user string, req PortionRequest, out Portions) {
	if req.Round == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.entries[cacheKey(user, req)] = &cachedPortion{user: user, round: req.Round, from: req.Ack, count: len(out.Portions), out: out, at: now}
	for k, e := range c.entries {
		if now.Sub(e.at) > c.ttl {
			delete(c.entries, k)
		}
	}
	for len(c.entries) > c.cap {
		oldest := ""
		var at time.Time
		for k, e := range c.entries {
			if oldest == "" || e.at.Before(at) {
				oldest, at = k, e.at
			}
		}
		delete(c.entries, oldest)
	}
}
