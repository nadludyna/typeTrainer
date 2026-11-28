//go:build !(js && wasm)

// The web server: go run ./cmd/typetrainer. With DATABASE_URL the data is
// kept in PostgreSQL, without it in memory until the server stops; ADDR
// changes where it listens (localhost:3000).
package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"typingtrainer/internal/app"
	"typingtrainer/internal/config"
	"typingtrainer/internal/storage"
)

func main() {
	store, err := openStore()
	if err != nil {
		log.Fatal(err)
	}
	handler, cfg, err := app.New(store, true)
	if err != nil {
		log.Fatal(err)
	}
	if addr := os.Getenv("ADDR"); addr != "" {
		cfg.Addr = addr // another address, as the browser checks run it
	}
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("typeTrainer running. Open http://%s in your browser", cfg.Addr)
	if err := serve(ctx, newServer(cfg, handler), ln, cfg.Server); err != nil {
		log.Fatal(err)
	}
	log.Println("typeTrainer stopped")
}

func openStore() (storage.Store, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Println("storage: memory (set DATABASE_URL to keep data in PostgreSQL)")
		return storage.NewMemoryStore(), nil
	}
	p, err := storage.OpenPostgres(url)
	if err == nil {
		log.Println("storage: PostgreSQL")
	}
	return p, err
}

// newServer limits how long a slow client may hold a connection.
func newServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.Server.ReadHeader,
		ReadTimeout:       cfg.Server.Read,
		WriteTimeout:      cfg.Server.Write,
		IdleTimeout:       cfg.Server.Idle,
	}
}

// serve stops on ctx: no new requests, and the running ones may finish, so a
// round being saved during a restart is not lost.
func serve(ctx context.Context, srv *http.Server, ln net.Listener, limits config.Server) error {
	done := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), limits.Shutdown)
		defer cancel()
		done <- srv.Shutdown(shut)
	}()
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return <-done
}
