// HTTP helpers: who a request belongs to, and JSON in and out.

package trainer

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"typingtrainer/internal/storage"
)

// Handler serves one route for one typist.
type Handler func(w http.ResponseWriter, r *http.Request, user string)

// Route turns h into a handler that first finds the typist of the request.
func (t *Trainer) Route(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { h(w, r, t.userOf(r)) }
}

const guestCookie = "tt_guest"

// userOf: every browser is its own guest, kept by a random token in a
// cookie; the store holds only the token's hash, never what opens the data.
// Without guests (GitHub Pages, one person in one browser) there is one typist.
func (t *Trainer) userOf(r *http.Request) string {
	if !t.Cfg.Guests {
		return storage.LocalUser
	}
	if c, err := r.Cookie(guestCookie); err == nil && c.Value != "" {
		sum := sha256.Sum256([]byte(c.Value))
		return storage.GuestPrefix + hex.EncodeToString(sum[:])
	}
	return storage.LocalUser
}

// WithGuest gives a browser without the cookie a new guest token. The page
// itself is the first request, so the cookie is there before any API call.
func (t *Trainer) WithGuest(next http.Handler) http.Handler {
	if !t.Cfg.Guests {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(guestCookie); err != nil || c.Value == "" {
			raw := make([]byte, 32)
			if _, err := rand.Read(raw); err != nil {
				Fail(w, err)
				return
			}
			token := hex.EncodeToString(raw)
			http.SetCookie(w, &http.Cookie{
				Name: guestCookie, Value: token, Path: "/", MaxAge: int((time.Duration(t.Cfg.GuestDays) * 24 * time.Hour).Seconds()),
				HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil,
			})
			r.AddCookie(&http.Cookie{Name: guestCookie, Value: token})
		}
		next.ServeHTTP(w, r)
	})
}

// Read decodes a JSON body of at most limit bytes into v; on a bad body it
// answers 400 and returns false.
func Read(w http.ResponseWriter, r *http.Request, limit int, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, int64(limit))).Decode(v); err != nil {
		Bad(w, "invalid JSON")
		return false
	}
	return true
}

func JSON(w http.ResponseWriter, v any) { writeJSON(w, http.StatusOK, v) }

// OK is the answer of a route that only does something.
type OK struct {
	OK bool `json:"ok"`
}

func Done(w http.ResponseWriter) { JSON(w, OK{OK: true}) }

func Bad(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
}

func Fail(w http.ResponseWriter, err error) {
	log.Println("error:", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
