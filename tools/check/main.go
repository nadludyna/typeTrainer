// Command check runs every check a pull request needs, in order, and says
// which failed: go run ./tools/check [-quick]. The same command runs in CI.
// -quick leaves out the browser checks. Run it from the repository.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// staticcheck is pinned: a new version may find new things, and that is a
// change of its own.
const staticcheck = "honnef.co/go/tools/cmd/staticcheck@v0.8.2"

type step struct {
	name string
	dir  string
	cmd  []string
	env  []string
	// ok judges the output too: gofmt -l lists unformatted files and still exits 0.
	ok func(out string) bool
}

func main() {
	quick := flag.Bool("quick", false, "leave out the browser checks")
	flag.Parse()
	if _, err := os.Stat("go.mod"); err != nil {
		fmt.Fprintln(os.Stderr, "check: run it from the repository (where go.mod is)")
		os.Exit(2)
	}
	if _, err := os.Stat("web/node_modules"); err != nil {
		fmt.Fprintln(os.Stderr, "check: the page's tools are not installed: cd web && npm ci")
		os.Exit(2)
	}
	wasm := filepath.Join(os.TempDir(), "typetrainer-check.wasm")
	steps := []step{
		{name: "gofmt", cmd: []string{"gofmt", "-l", "."}, ok: func(out string) bool { return strings.TrimSpace(out) == "" }},
		{name: "go vet", cmd: []string{"go", "vet", "./..."}},
		{name: "staticcheck", cmd: []string{"go", "run", staticcheck, "./..."}},
		{name: "go test", cmd: []string{"go", "test", "./..."}},
		{name: "WebAssembly build", cmd: []string{"go", "build", "-o", wasm, "./cmd/typetrainer"}, env: []string{"GOOS=js", "GOARCH=wasm"}},
		{name: "page: types, lint, unused, tests", dir: "web", cmd: npm("run", "check")},
		{name: "page build", dir: "web", cmd: npm("run", "build")},
	}
	if !*quick {
		steps = append(steps, step{name: "browser checks", dir: "web", cmd: npx("playwright", "test")})
	}
	failed := 0
	start := time.Now()
	for _, s := range steps {
		t := time.Now()
		out, err := run(s)
		good := err == nil && (s.ok == nil || s.ok(out))
		took := time.Since(t).Round(100 * time.Millisecond)
		if good {
			fmt.Printf("✓ %s  %s\n", s.name, took)
			continue
		}
		failed++
		fmt.Printf("✗ %s  %s\n%s\n", s.name, took, tail(out, 60))
	}
	os.Remove(wasm)
	if failed > 0 {
		fmt.Printf("%d of %d failed (%s)\n", failed, len(steps), time.Since(start).Round(time.Second))
		os.Exit(1)
	}
	fmt.Printf("all %d passed (%s)\n", len(steps), time.Since(start).Round(time.Second))
}

// npm and npx: on Windows they are .cmd files, which exec does not find by
// their bare name.
func npm(args ...string) []string { return append([]string{node("npm")}, args...) }
func npx(args ...string) []string { return append([]string{node("npx")}, args...) }

func node(tool string) string {
	if os.PathSeparator == '\\' {
		return tool + ".cmd"
	}
	return tool
}

func run(s step) (string, error) {
	c := exec.Command(s.cmd[0], s.cmd[1:]...)
	c.Dir = s.dir
	c.Env = append(os.Environ(), s.env...)
	var buf bytes.Buffer
	c.Stdout, c.Stderr = &buf, &buf
	err := c.Run()
	return buf.String(), err
}

// tail is the last n lines: where a failure's own message is.
func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return strings.Join(lines[max(0, len(lines)-n):], "\n")
}
