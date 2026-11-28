//go:build !(js && wasm)

// What the normal server build sets up differently.
package main

import (
	"io/fs"
	"log"
	"os"

	"typingtrainer/internal/storage"
)

const guestsOn = true

func openStore() (storage.Store, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Println("storage: memory (set DATABASE_URL to keep data in PostgreSQL)")
		return storage.NewMemoryStore(), nil
	}
	p, err := storage.OpenPostgres(url)
	if err != nil {
		return nil, err
	}
	log.Println("storage: PostgreSQL")
	return p, nil
}

// liveWeb: with TT_DEV=1, run from the repository, the page is read from
// web/ on the disk at every request, so a saved change shows at the next
// reload without building the server again. nil means the files built in.
func liveWeb() fs.FS {
	if os.Getenv("TT_DEV") != "1" {
		return nil
	}
	if _, err := os.Stat("web/index.html"); err != nil {
		log.Println("TT_DEV=1, but there is no web/ here: serving the files built in")
		return nil
	}
	log.Println("web: from the disk, as it is at every request (TT_DEV=1)")
	return os.DirFS("web")
}
