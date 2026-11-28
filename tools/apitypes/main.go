// Command apitypes writes web/src/core/api-types.ts: the TypeScript shape of
// every request and answer of the API, read from the Go types themselves, so
// the page and the server cannot drift apart. go run ./tools/apitypes; the
// test fails while the file is out of date.
package main

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"typingtrainer/internal/app"
	"typingtrainer/internal/layout"
	"typingtrainer/internal/modes/freetype"
	"typingtrainer/internal/modes/learn"
	"typingtrainer/internal/modes/reinforce"
	"typingtrainer/internal/trainer"
)

const out = "web/src/core/api-types.ts"

// types is what the page sends and gets, each under the name the page uses.
var types = []struct {
	name string
	v    any
}{
	{"PortionRequest", trainer.PortionRequest{}},
	{"Portions", trainer.Portions{}},
	{"Result", trainer.Result{}},
	{"OK", trainer.OK{}},
	{"LearnResult", learn.Result{}},
	{"LearnStats", learn.Stats{}},
	{"Checkin", learn.Checkin{}},
	{"ReinforceResult", reinforce.Result{}},
	{"ReinforceStats", reinforce.Stats{}},
	{"Step", reinforce.Step{}},
	{"FreetypePortionRequest", freetype.PortionRequest{}},
	{"FreetypePortion", freetype.Portion{}},
	{"FreetypePortions", freetype.Portions{}},
	{"FreetypeResult", freetype.Result{}},
	{"FreetypeStats", freetype.Stats{}},
	{"File", app.File{}},
	{"Imported", app.Imported{}},
	{"Layout", layout.Layout{}},
	{"LayoutFinger", layout.Finger{}},
	{"Key", layout.Key{}},
}

func main() {
	if err := os.WriteFile(out, []byte(generate()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "apitypes:", err)
		os.Exit(1)
	}
}

type gen struct {
	names map[reflect.Type]string
	defs  map[string]string
}

func generate() string {
	g := gen{names: map[reflect.Type]string{}, defs: map[string]string{}}
	for _, t := range types {
		g.names[reflect.TypeOf(t.v)] = t.name
	}
	for _, t := range types {
		g.named(reflect.TypeOf(t.v))
	}
	keys := make([]string, 0, len(g.defs))
	for k := range g.defs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("// Written by go run ./tools/apitypes from the Go types: do not edit.\n")
	for _, k := range keys {
		b.WriteString("\n" + g.defs[k])
	}
	return b.String()
}

// named is the interface for a struct type, written once under its name.
func (g *gen) named(t reflect.Type) string {
	name, ok := g.names[t]
	if !ok {
		name = t.Name()
		for other, n := range g.names {
			if n == name && other != t {
				panic(fmt.Sprintf("two types are called %s: give one a name in types", name))
			}
		}
		g.names[t] = name
	}
	if _, done := g.defs[name]; done {
		return name
	}
	g.defs[name] = "" // a type that holds itself stops here
	var b strings.Builder
	b.WriteString("export interface " + name + " {\n")
	g.fields(&b, t)
	b.WriteString("}\n")
	g.defs[name] = b.String()
	return name
}

func (g *gen) fields(b *strings.Builder, t reflect.Type) {
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Anonymous {
			g.fields(b, f.Type)
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" || !f.IsExported() {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if name == "" {
			name = f.Name
		}
		optional := ""
		if strings.Contains(opts, "omitempty") {
			optional = "?"
		}
		fmt.Fprintf(b, "  %s%s: %s;\n", name, optional, g.ts(f.Type))
	}
}

func (g *gen) ts(t reflect.Type) string {
	if t == reflect.TypeOf(time.Time{}) {
		return "string"
	}
	switch t.Kind() {
	case reflect.Pointer:
		return g.ts(t.Elem()) + " | null"
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int64, reflect.Float64:
		return "number"
	case reflect.Slice:
		return wrap(g.ts(t.Elem())) + "[]"
	case reflect.Map:
		return "Record<string, " + g.ts(t.Elem()) + ">"
	case reflect.Struct:
		return g.named(t)
	}
	panic("no TypeScript for " + t.String())
}

func wrap(s string) string {
	if strings.Contains(s, " ") {
		return "(" + s + ")"
	}
	return s
}
