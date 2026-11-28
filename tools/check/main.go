// Command check runs every check a pull request needs, in order, and says
// which failed: go run ./tools/check [-quick].
// -quick leaves out the browser checks. Run it from the repository.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type step struct {
	name string
	dir  string
	cmd  []string
	env  []string
	// ok judges the output too: gofmt -l lists unformatted files and still
	// exits 0.
	ok func(out string) bool
}

func main() {
	quick := flag.Bool("quick", false, "leave out the browser checks")
	flag.Parse()
	if _, err := os.Stat("go.mod"); err != nil {
		fmt.Fprintln(os.Stderr, "check: run it from the repository (where go.mod is)")
		os.Exit(2)
	}
	wasm := filepath.Join(os.TempDir(), "typetrainer-check.wasm")
	steps := []step{
		{name: "gofmt", cmd: []string{"gofmt", "-l", "."}, ok: func(out string) bool { return strings.TrimSpace(out) == "" }},
		{name: "go vet", cmd: []string{"go", "vet", "./..."}},
		{name: "go test", cmd: []string{"go", "test", "./..."}},
		{name: "Pages build (WebAssembly)", cmd: []string{"go", "build", "-o", wasm, "."}, env: []string{"GOOS=js", "GOARCH=wasm"}},
	}
	if !*quick {
		if _, err := os.Stat("tools/browser/node_modules"); err != nil {
			fmt.Println("✗ browser checks: their tools are not installed (cd tools/browser && npm ci)")
			os.Exit(1)
		}
		if busy("localhost:3000") {
			fmt.Println("! something already serves localhost:3000: the browser checks test that server, not this code")
		}
		steps = append(steps, step{name: "browser checks", dir: "tools/browser", cmd: []string{"npx", "playwright", "test"}})
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

func busy(addr string) bool {
	c, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}
