// Package layout loads keyboard layouts from data/layouts.
package layout

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Key struct {
	Char     string  `json:"char"`
	Label    string  `json:"label,omitempty"`
	Finger   string  `json:"finger"`
	Home     bool    `json:"home,omitempty"`
	Width    float64 `json:"width,omitempty"`
	Alt      string  `json:"alt,omitempty"`
	Modifier bool    `json:"modifier,omitempty"`
}

type Finger struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Layout struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Fingers map[string]Finger `json:"fingers"`
	Rows    [][]Key           `json:"rows"`
	Offsets []float64         `json:"offsets"`
}

func Parse(data []byte) (*Layout, error) {
	var l Layout
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("layout json: %w", err)
	}
	for _, row := range l.Rows {
		for _, k := range row {
			if _, ok := l.Fingers[k.Finger]; !ok {
				return nil, fmt.Errorf("key %q uses unknown finger %q", k.Char, k.Finger)
			}
		}
	}
	return &l, nil
}

// Place is where a key stands: its finger, its row and how far from the
// left edge of the letter block, in key widths (the row's offset counted).
type Place struct {
	Finger string
	Row    int
	X      float64
}

// Places is the place of every one-character key that is not a modifier.
func (l *Layout) Places() map[string]Place {
	out := map[string]Place{}
	for r, row := range l.Rows {
		x := 0.0
		if r < len(l.Offsets) {
			x = l.Offsets[r]
		}
		for _, k := range row {
			w := k.Width
			if w == 0 {
				w = 1
			}
			if !k.Modifier && utf8.RuneCountInString(k.Char) == 1 && k.Char != " " {
				out[k.Char] = Place{Finger: k.Finger, Row: r, X: x + w/2}
			}
			x += w
		}
	}
	return out
}

// FingerOrder is the fingers from the left little finger to the right one,
// as the layout lists them.
func (l *Layout) FingerOrder() []string {
	var out []string
	seen := map[string]bool{}
	for _, row := range l.Rows {
		for _, k := range row {
			seen[k.Finger] = true
		}
	}
	for _, f := range []string{"lp", "lr", "lm", "li", "ri", "rm", "rr", "rp"} {
		if seen[f] {
			out = append(out, f)
		}
	}
	return out
}

func (l *Layout) RestingKeys() []string {
	for _, row := range l.Rows {
		bump := false
		for _, k := range row {
			bump = bump || k.Home
		}
		if !bump {
			continue
		}
		var out []string
		taken := map[string]bool{}
		for _, k := range row {
			if k.Home && !taken[k.Finger] {
				taken[k.Finger] = true
				out = append(out, k.Char)
			}
		}
		for i := len(row) - 1; i >= 0; i-- {
			if k := row[i]; strings.HasPrefix(k.Finger, "l") && !taken[k.Finger] && !k.Modifier {
				taken[k.Finger] = true
				out = append(out, k.Char)
			}
		}
		for _, k := range row {
			if strings.HasPrefix(k.Finger, "r") && !taken[k.Finger] && !k.Modifier {
				taken[k.Finger] = true
				out = append(out, k.Char)
			}
		}
		return out
	}
	return nil
}
