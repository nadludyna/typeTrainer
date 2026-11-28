// Package web is the page, built by Vite into dist/page and built into the
// program; dist/.keep keeps the folder for go:embed before the first build.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Files is the built page.
func Files() fs.FS {
	f, err := fs.Sub(dist, "dist/page")
	if err != nil {
		panic(err)
	}
	return f
}
