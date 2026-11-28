//go:build js && wasm

// The GitHub Pages build: the same program as WebAssembly inside the page.
// It listens on no port; the page calls goApi(method, path, body) instead of
// fetch for /api/, and keeps the data in the browser (web/src/core/pages.ts).
package main

import (
	"net/http/httptest"
	"strings"
	"syscall/js"

	"typingtrainer/internal/app"
	"typingtrainer/internal/storage"
)

func main() {
	handler, _, err := app.New(storage.NewMemoryStore(), false)
	if err != nil {
		js.Global().Get("console").Call("error", "typeTrainer: "+err.Error())
		return
	}
	js.Global().Set("goApi", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 3 {
			return map[string]any{"status": 400, "body": `{"error":"goApi needs method, path, body"}`}
		}
		req := httptest.NewRequest(args[0].String(), args[1].String(), strings.NewReader(args[2].String()))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return map[string]any{"status": rec.Code, "body": rec.Body.String()}
	}))
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("go-ready"))
	select {}
}
