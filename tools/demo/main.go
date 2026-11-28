// Command demo prints a made-up history of every mode as an import file,
// to look at the screens with data in them: go run ./tools/demo > demo.json,
// then Settings → Import.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"typingtrainer/internal/config"
	"typingtrainer/internal/demo"
	"typingtrainer/internal/engine"
	"typingtrainer/internal/language"
	"typingtrainer/internal/layout"
)

func main() {
	lang := flag.String("lang", "en", "language id")
	flag.Parse()
	if err := run(*lang); err != nil {
		fmt.Fprintln(os.Stderr, "demo:", err)
		os.Exit(1)
	}
}

func run(id string) error {
	data, err := os.ReadFile("data/languages.json")
	if err != nil {
		return fmt.Errorf("%w (run it from the repository)", err)
	}
	langs, err := language.Parse(data)
	if err != nil {
		return err
	}
	for _, l := range langs {
		if l.ID != id {
			continue
		}
		list, err := os.ReadFile("data/langs/" + l.ID + "/free.txt")
		if err != nil {
			return err
		}
		layoutJSON, err := os.ReadFile("data/layouts/" + l.Layout + ".json")
		if err != nil {
			return err
		}
		lay, err := layout.Parse(layoutJSON)
		if err != nil {
			return err
		}
		cfg := config.Default()
		words := l.Words(string(list))
		gen := engine.NewGenerator(words, rand.New(rand.NewSource(1)), cfg.Engine)
		gen.SetHome(lay.RestingKeys(), cfg.Engine)
		out := json.NewEncoder(os.Stdout)
		return out.Encode(demo.Build(gen.Order(), words, cfg.Engine, time.Now(), 1))
	}
	return fmt.Errorf("no language %q", id)
}
