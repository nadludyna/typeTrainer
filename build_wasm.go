//go:build js && wasm

// What the GitHub Pages build sets up differently.
package main

import (
	"io/fs"

	"typingtrainer/internal/storage"
)

const guestsOn = false

func openStore() (storage.Store, error) {
	return storage.NewMemoryStore(), nil
}

// liveWeb: the Pages build always serves the files built in.
func liveWeb() fs.FS { return nil }
