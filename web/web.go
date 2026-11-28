// Package web is the page, built by Vite into dist/ and built into the program.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Files is the built page.
func Files() fs.FS {
	f, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return f
}
