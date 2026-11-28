//go:build !(js && wasm)

package storage

import (
	"math"
	"os"
	"testing"
)

// openTestDB is the database of TT_TEST_DATABASE_URL, emptied; the tests are
// skipped without one.
func openTestDB(t *testing.T) *PostgresStore {
	url := os.Getenv("TT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TT_TEST_DATABASE_URL not set: no test database")
	}
	p, err := OpenPostgres(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

func empty(t *testing.T, p *PostgresStore) {
	if _, err := p.db.Exec(`TRUNCATE progress, rounds RESTART IDENTITY`); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresStore(t *testing.T) {
	p := openTestDB(t)
	runStoreTests(t, func(t *testing.T) Store {
		empty(t, p)
		return p
	})
	again, err := OpenPostgres(os.Getenv("TT_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("a second open applies no migration twice: %v", err)
	}
	again.Close()
}

func TestPostgresSaveIsAllOrNothing(t *testing.T) {
	p := openTestDB(t)
	empty(t, p)
	p.Save(u, &Progress{Unlocked: 6}, nil)
	bad := round(ModeLearn, 40)
	bad.Chars = math.MaxInt32 + 1
	if err := p.Save(u, &Progress{Unlocked: 7}, &bad); err == nil {
		t.Fatal("a round that does not fit was stored")
	}
	pr, _ := p.Progress(u)
	rs, _ := p.Rounds(u, "", 0)
	if pr.Unlocked != 6 || len(rs) != 0 {
		t.Fatalf("half stored: %+v, %d rounds", pr, len(rs))
	}
}
