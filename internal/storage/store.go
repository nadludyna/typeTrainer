// Package storage keeps all data behind the Store interface.
package storage

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"typingtrainer/internal/engine"
)

type Session struct {
	WPM         float64   `json:"wpm"`
	Accuracy    float64   `json:"accuracy"`
	Chars       int       `json:"chars"`
	Consistency *float64  `json:"consistency,omitempty"`
	FinishedAt  time.Time `json:"finishedAt"`
	Length      string    `json:"length,omitempty"`

	Keys     []engine.Sample `json:"keys,omitempty"`
	Unlocked int             `json:"unlocked,omitempty"`
	Level    float64         `json:"level,omitempty"`
	Lap      int             `json:"lap,omitempty"`
	Device   string          `json:"device,omitempty"`
	Kind     string          `json:"kind,omitempty"` // "" the usual round, or KindWeak, KindClean
	Epoch    int             `json:"epoch"`
	Aborted  bool            `json:"aborted,omitempty"`
	Seconds  float64         `json:"seconds,omitempty"`
	Round    string          `json:"-"`
	Mode     string          `json:"-"` // set by LastFreetypeRounds only
	seq      int64           // the memory store's order of saving, as PostgreSQL's id
}

const (
	VariantLong      = "long"
	VariantSentences = "sentences"
	VariantCustom    = "custom"
)

var FreetypeVariants = []string{"", VariantLong, VariantSentences, VariantCustom}

func Listless(variant string) bool { return variant == VariantSentences || variant == VariantCustom }

func FreetypeKey(n int, variant string) string {
	switch {
	case variant == "":
		return strconv.Itoa(n)
	case Listless(variant):
		return variant
	}
	return strconv.Itoa(n) + "-" + variant
}

// freetypePrefix starts the mode of every Freetype round.
const freetypePrefix = "freetype-"

func FreetypeMode(n int, variant string) string { return freetypePrefix + FreetypeKey(n, variant) }

// ParseFreetypeMode is ParseFreetypeKey of a round's mode.
func ParseFreetypeMode(mode string) (n int, variant string, ok bool) {
	key, found := strings.CutPrefix(mode, freetypePrefix)
	if !found {
		return 0, "", false
	}
	return ParseFreetypeKey(key)
}

func ParseFreetypeKey(key string) (n int, variant string, ok bool) {
	if Listless(key) {
		return 0, key, true
	}
	num, rest, dash := strings.Cut(key, "-")
	n, err := strconv.Atoi(num)
	if err != nil || n <= 0 || Listless(rest) || (dash && rest == "") {
		return 0, "", false
	}
	for _, v := range FreetypeVariants {
		if v == rest {
			return n, v, true
		}
	}
	return 0, "", false
}

type Day struct {
	Date    string  `json:"date"`
	Rounds  int     `json:"rounds"`
	Seconds float64 `json:"seconds"`
	Chars   int     `json:"chars"`
}

// TypedSeconds is the time a round counts toward the daily goal: the time
// the browser measured, or for rounds saved before it was sent, an estimate
// from speed and characters (aborted ones then count nothing).
func (s Session) TypedSeconds() float64 {
	if s.Seconds > 0 {
		return s.Seconds
	}
	if s.Aborted {
		return 0
	}
	return RoundSeconds(s.WPM, s.Accuracy, s.Chars)
}

func RoundSeconds(wpm, accuracy float64, chars int) float64 {
	if wpm <= 0 {
		return 0
	}
	return float64(chars) * accuracy / 100 / 5 / wpm * 60
}

// The kinds of round besides the usual one. A round of a kind is kept and
// shown but moves no word level or list.
const (
	KindWeak  = "weak"  // built from the typist's weak spots
	KindClean = "clean" // ends at the third mistake
)

func ValidKind(k string) bool { return k == "" || k == KindWeak || k == KindClean }

// Usual is the rounds of no special kind: the ones a level is judged by.
func Usual(rounds []Session) []Session {
	out := make([]Session, 0, len(rounds))
	for _, r := range rounds {
		if r.Kind == "" {
			out = append(out, r)
		}
	}
	return out
}

const (
	DeviceKeyboard = "keyboard"
	DeviceTouch    = "touch"
)

// OnKeyboard is the rounds typed on a keyboard: the only ones the trainer
// learns from. Thumbs on glass make other mistakes than fingers on keys, so a
// phone round is kept and shown but changes no letter, pair or word level. A
// round from before devices were kept has none and counts as a keyboard's.
func OnKeyboard(rounds []Session) []Session {
	out := make([]Session, 0, len(rounds))
	for _, r := range rounds {
		if r.Device != DeviceTouch {
			out = append(out, r)
		}
	}
	return out
}

const DefaultLength = "words-30"

func (s Session) WithLength() Session {
	if s.Length == "" {
		s.Length = DefaultLength
	}
	return s
}

func (s Session) Summary() Session {
	s.Keys = nil
	return s
}

const LocalUser = "local"

const GuestPrefix = "g"

func IsGuest(user string) bool {
	return user == LocalUser || strings.HasPrefix(user, GuestPrefix)
}

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

var (
	ErrNameTaken = errors.New("storage: name taken")
	ErrNotFound  = errors.New("storage: not found")
)

// ImportData is one whole progress state for a language, applied all at once.
type ImportData struct {
	Keys     map[string]engine.KeyStat
	Unlocked int
	Lap      int
	Sessions []Session
	Freetype map[string][]Session
}

type Learned struct {
	Stats    map[string]engine.KeyStat
	Unlocked int
	Lap      int
	Session  *Session
}

// The parts of a language's rounds, one per mode: Learn and Reinforce share
// the learning rounds, and a Reinforce round has a word level.
const (
	PartLearn     = "learn"
	PartReinforce = "reinforce"
	PartFreetype  = "freetype"
)

// InPart: is a learning round (word level 0 or not) in this part ("" = all)?
func InPart(level float64, part string) bool {
	switch part {
	case PartLearn:
		return level == 0
	case PartReinforce:
		return level > 0
	}
	return part == ""
}

type Store interface {
	KeyStats(user, lang string) (map[string]engine.KeyStat, error)
	SaveKeyStats(user, lang string, stats map[string]engine.KeyStat) error

	Unlocked(user, lang string) (int, error)
	SetUnlocked(user, lang string, n int) error
	Lap(user, lang string) (int, error)
	SetLap(user, lang string, n int) error

	AddSession(user, lang string, s Session) error
	RecentSessions(user, lang, length string, limit int) ([]Session, error)
	SessionCount(user, lang, length string) (int, error)
	// LettersTyped is the letters of the learning rounds typed on a keyboard
	// with at least unlocked letters open (OnKeyboard).
	LettersTyped(user, lang string, unlocked int) (int, error)

	AddFreetypeSession(user, lang, mode string, s Session) error
	RecentFreetypeSessions(user, lang, mode, length string, limit int) ([]Session, error)
	// LastFreetypeRounds is the last Freetype rounds of every list and
	// text, newest first, each with its Mode.
	LastFreetypeRounds(user, lang string, limit int) ([]Session, error)
	FreetypeSessionCount(user, lang, mode, length string) (int, error)
	FreetypeSessionLengths(user, lang, mode string) ([]string, error)
	BestFreetypeRounds(user, lang, mode string) (map[string]Session, error)

	HasRound(user, lang, round string) (bool, error)
	SaveLearned(user, lang string, l Learned) error

	// PracticeDays: part is "" for every round, or one mode's (PartLearn ...).
	PracticeDays(user, lang, part string, tz time.Duration) ([]Day, error)

	Reset(user, lang string) error
	// ResetLearn starts Learn over: its letters and rounds; the
	// other modes stay.
	ResetLearn(user, lang string) error
	Import(user, lang string, data ImportData) error

	CreateUser(name, passwordHash string, now time.Time, from string) (User, error)
	UserByName(name string) (User, error)
	UserByID(id string) (User, error)

	AddLogin(tokenHash, userID string, expires time.Time) error
	LoginUser(tokenHash string, now time.Time) (string, error)
	DeleteLogin(tokenHash string) error
}
