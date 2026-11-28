//go:build !(js && wasm)

// The PostgreSQL Store, through the standard database/sql; pgx is only the
// driver. Not in the GitHub Pages build.

package storage

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"typingtrainer/internal/engine"
)

//go:embed migrations/*.sql
var migrations embed.FS

type PostgresStore struct {
	db *sql.DB
}

func OpenPostgres(url string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot reach the database: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}
	return &PostgresStore{db: db}, nil
}

func (p *PostgresStore) Close() error { return p.db.Close() }

// migrate applies, in number order, each migrations/NNN_*.sql file the
// database has not recorded yet, each in a transaction.
func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		version, err := strconv.Atoi(strings.SplitN(strings.TrimPrefix(name, "migrations/"), "_", 2)[0])
		if err != nil {
			return fmt.Errorf("%s: the file name must start with a number", name)
		}
		var done bool
		if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&done); err != nil {
			return err
		}
		if done {
			continue
		}
		body, err := migrations.ReadFile(name)
		if err != nil {
			return err
		}
		err = inTx(db, func(tx *sql.Tx) error {
			if _, err := tx.Exec(string(body)); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			_, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func inTx(db *sql.DB, f func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := f(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (p *PostgresStore) Progress(user string) (Progress, error) {
	var letters []byte
	out := Progress{Letters: map[string]engine.KeyStat{}}
	err := p.db.QueryRow(`SELECT letters, unlocked, lap FROM progress WHERE user_id = $1`, user).Scan(&letters, &out.Unlocked, &out.Lap)
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	return out, json.Unmarshal(letters, &out.Letters)
}

func (p *PostgresStore) Save(user string, pr *Progress, r *Round) error {
	return inTx(p.db, func(tx *sql.Tx) error {
		if pr != nil {
			if err := saveProgress(tx, user, *pr); err != nil {
				return err
			}
		}
		if r == nil {
			return nil
		}
		return addRound(tx, user, *r)
	})
}

func saveProgress(tx *sql.Tx, user string, pr Progress) error {
	if pr.Letters == nil {
		pr.Letters = map[string]engine.KeyStat{}
	}
	letters, err := json.Marshal(pr.Letters)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO progress (user_id, letters, unlocked, lap) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET letters = EXCLUDED.letters, unlocked = EXCLUDED.unlocked, lap = EXCLUDED.lap`,
		user, letters, pr.Unlocked, pr.Lap)
	return err
}

// addRound keeps a round once: a second round with the same ID is dropped.
func addRound(tx *sql.Tx, user string, r Round) error {
	var keys []byte
	if r.Keys != nil {
		var err error
		if keys, err = json.Marshal(r.Keys); err != nil {
			return err
		}
	}
	var id any
	if r.ID != "" {
		id = r.ID
	}
	_, err := tx.Exec(`INSERT INTO rounds
		(user_id, round_id, mode, kind, level, lap, length, wpm, accuracy, chars, consistency, unlocked, finished_at, keys)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (user_id, round_id) DO NOTHING`,
		user, id, r.Mode, r.Kind, r.Level, r.Lap, r.Length, r.WPM, r.Accuracy, r.Chars, r.Consistency, r.Unlocked, r.FinishedAt, keys)
	return err
}

func (p *PostgresStore) HasRound(user, id string) (bool, error) {
	if id == "" {
		return false, nil
	}
	var found bool
	err := p.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM rounds WHERE user_id = $1 AND round_id = $2)`, user, id).Scan(&found)
	return found, err
}

func (p *PostgresStore) Rounds(user, prefix string, limit int) ([]Round, error) {
	query := `SELECT COALESCE(round_id, ''), mode, kind, level, lap, length, wpm, accuracy, chars, consistency, unlocked, finished_at, keys
		FROM rounds WHERE user_id = $1 AND starts_with(mode, $2) ORDER BY id DESC`
	args := []any{user, prefix}
	if limit > 0 {
		query += ` LIMIT $3`
		args = append(args, limit)
	}
	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Round{}
	for rows.Next() {
		var r Round
		var keys []byte
		if err := rows.Scan(&r.ID, &r.Mode, &r.Kind, &r.Level, &r.Lap, &r.Length, &r.WPM, &r.Accuracy, &r.Chars,
			&r.Consistency, &r.Unlocked, &r.FinishedAt, &keys); err != nil {
			return nil, err
		}
		if keys != nil {
			if err := json.Unmarshal(keys, &r.Keys); err != nil {
				return nil, err
			}
		}
		r.FinishedAt = r.FinishedAt.UTC()
		out = append(out, r)
	}
	return out, rows.Err()
}

func (p *PostgresStore) Reset(user string) error {
	return inTx(p.db, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM progress WHERE user_id = $1`, user); err != nil {
			return err
		}
		_, err := tx.Exec(`DELETE FROM rounds WHERE user_id = $1`, user)
		return err
	})
}

func (p *PostgresStore) ResetLearn(user string) error {
	return inTx(p.db, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM progress WHERE user_id = $1`, user); err != nil {
			return err
		}
		_, err := tx.Exec(`DELETE FROM rounds WHERE user_id = $1 AND mode IN ($2, $3)`, user, ModeLearn, ModeReinforce)
		return err
	})
}

func (p *PostgresStore) Import(user string, pr Progress, rounds []Round) error {
	return inTx(p.db, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM rounds WHERE user_id = $1`, user); err != nil {
			return err
		}
		if err := saveProgress(tx, user, pr); err != nil {
			return err
		}
		for _, r := range rounds {
			if err := addRound(tx, user, r); err != nil {
				return err
			}
		}
		return nil
	})
}
