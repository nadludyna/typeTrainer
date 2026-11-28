// The words a learner owns (profile.ReadWords), for the word draws of
// Reinforce and Freetype and for their stats.

package api

import (
	"slices"
	"time"

	"typingtrainer/internal/profile"
	"typingtrainer/internal/storage"
)

type vocabAt struct {
	at   time.Time
	v    profile.Vocabulary
	seen profile.Seen // when each letter was last typed in Learn and Reinforce
}

// vocabKept is how long a learner's owned words are kept between the
// portions of one round.
const vocabKept = time.Hour

// vocabulary reads the owned words from the last Owned.Rounds rounds of
// Learn and Reinforce and as many of Freetype, typed on a keyboard. It is
// read again when a round starts (fresh) and kept for the round's other
// portions, so a portion does not read the history again.
func (s *Server) vocabulary(user, lang string, fresh bool) (profile.Vocabulary, profile.Seen, error) {
	key := user + "\x00" + lang
	s.vocabMu.Lock()
	kept, ok := s.vocab[key]
	s.vocabMu.Unlock()
	if ok && !fresh && time.Since(kept.at) < vocabKept {
		return kept.v, kept.seen, nil
	}
	n := s.cfg.Owned.Rounds
	learning, err := s.store.RecentSessions(user, lang, "", n)
	if err != nil {
		return profile.Vocabulary{}, profile.Seen{}, err
	}
	seen := profile.Seen{Last: map[string]time.Time{}}
	for _, r := range learning {
		if seen.Oldest.IsZero() || r.FinishedAt.Before(seen.Oldest) {
			seen.Oldest = r.FinishedAt
		}
		for _, k := range r.Keys {
			if r.FinishedAt.After(seen.Last[k.Char]) {
				seen.Last[k.Char] = r.FinishedAt
			}
		}
	}
	free, err := s.store.LastFreetypeRounds(user, lang, n)
	if err != nil {
		return profile.Vocabulary{}, profile.Seen{}, err
	}
	rounds := append(learning, free...)
	slices.SortStableFunc(rounds, func(a, b storage.Session) int { return a.FinishedAt.Compare(b.FinishedAt) })
	keys := make([][]profile.Key, len(rounds))
	for i, r := range rounds {
		keys[i] = make([]profile.Key, len(r.Keys))
		for j, k := range r.Keys {
			keys[i][j] = profile.Key{Char: k.Char, Errors: k.Errors, Ms: k.Ms}
		}
	}
	v := profile.ReadWords(keys, s.cfg.Owned)
	s.vocabMu.Lock()
	for k, old := range s.vocab {
		if time.Since(old.at) >= vocabKept {
			delete(s.vocab, k)
		}
	}
	s.vocab[key] = vocabAt{at: time.Now(), v: v, seen: seen}
	s.vocabMu.Unlock()
	return v, seen, nil
}

// ownedPerList is how many of the n most common words the learner owns, for each n.
func ownedPerList(v profile.Vocabulary, l *Language, sizes []int) []int {
	out := make([]int, len(sizes))
	for i, n := range sizes {
		out[i] = v.OwnedOf(l.Freetype.Top(n))
	}
	return out
}
