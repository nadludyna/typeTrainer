// Every browser is its own guest: a random token in a cookie, kept in the
// store only as its hash, so the data never holds what opens it.

package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"typingtrainer/internal/storage"
)

const guestCookie = "tt_guest"

func (s *Server) userOf(r *http.Request) (string, error) {
	if !s.cfg.Guests {
		return storage.LocalUser, nil
	}
	if c, err := r.Cookie(guestCookie); err == nil && c.Value != "" {
		sum := sha256.Sum256([]byte(c.Value))
		return storage.GuestPrefix + hex.EncodeToString(sum[:]), nil
	}
	return storage.LocalUser, nil
}

// WithGuest gives a browser without the cookie a new guest token. The page
// itself is the first request, so the cookie is there before any API call.
func (s *Server) WithGuest(next http.Handler) http.Handler {
	if !s.cfg.Guests {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(guestCookie); err != nil || c.Value == "" {
			raw := make([]byte, 32)
			if _, err := rand.Read(raw); err != nil {
				serverError(w, err)
				return
			}
			token := hex.EncodeToString(raw)
			age := time.Duration(s.cfg.GuestDays) * 24 * time.Hour
			http.SetCookie(w, &http.Cookie{
				Name: guestCookie, Value: token, Path: "/", MaxAge: int(age.Seconds()),
				HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil,
			})
			r.AddCookie(&http.Cookie{Name: guestCookie, Value: token})
		}
		next.ServeHTTP(w, r)
	})
}
