// Package data is the word list and the keyboard layout, built into the program.
package data

import _ "embed"

// Words is the word list, the most common first (tools/wordlists builds it).
//
//go:embed words.txt
var Words []byte

// Layout is the English QWERTY keyboard: its keys, rows and fingers.
//
//go:embed qwerty.json
var Layout []byte
