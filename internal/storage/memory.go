// The in-memory Store: lost when the program stops.

package storage

import (
	"maps"
	"slices"
	"strings"
	"sync"

	"typingtrainer/internal/engine"
)

type MemoryStore struct {
	mu    sync.Mutex
	users map[string]*typist
}

type typist struct {
	progress Progress
	rounds   []Round // oldest first
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{users: map[string]*typist{}} }

func (m *MemoryStore) of(user string) *typist {
	t, ok := m.users[user]
	if !ok {
		t = &typist{progress: Progress{Letters: map[string]engine.KeyStat{}}}
		m.users[user] = t
	}
	return t
}

func (m *MemoryStore) Progress(user string) (Progress, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.of(user).progress
	p.Letters = maps.Clone(p.Letters)
	return p, nil
}

func (m *MemoryStore) Save(user string, p *Progress, r *Round) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.of(user)
	if p != nil {
		t.progress = *p
		t.progress.Letters = maps.Clone(p.Letters)
		if t.progress.Letters == nil {
			t.progress.Letters = map[string]engine.KeyStat{}
		}
	}
	if r != nil && !t.has(r.ID) {
		t.rounds = append(t.rounds, *r)
	}
	return nil
}

func (m *MemoryStore) HasRound(user, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.of(user).has(id), nil
}

func (t *typist) has(id string) bool {
	return id != "" && slices.ContainsFunc(t.rounds, func(r Round) bool { return r.ID == id })
}

func (m *MemoryStore) Rounds(user, prefix string, limit int) ([]Round, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Round{}
	all := m.of(user).rounds
	for i := len(all) - 1; i >= 0 && (limit <= 0 || len(out) < limit); i-- {
		if strings.HasPrefix(all[i].Mode, prefix) {
			out = append(out, all[i])
		}
	}
	return out, nil
}

func (m *MemoryStore) Reset(user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, user)
	return nil
}

func (m *MemoryStore) ResetLearn(user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.of(user)
	t.progress = Progress{Letters: map[string]engine.KeyStat{}}
	t.rounds = slices.DeleteFunc(t.rounds, func(r Round) bool { return r.Mode == ModeLearn || r.Mode == ModeReinforce })
	return nil
}

func (m *MemoryStore) Import(user string, p Progress, rounds []Round) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := &typist{progress: p}
	t.progress.Letters = maps.Clone(p.Letters)
	if t.progress.Letters == nil {
		t.progress.Letters = map[string]engine.KeyStat{}
	}
	for _, r := range rounds {
		if !t.has(r.ID) {
			t.rounds = append(t.rounds, r)
		}
	}
	m.users[user] = t
	return nil
}
