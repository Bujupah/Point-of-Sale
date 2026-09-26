// Package storage owns the single SQLite connection, pragma setup, the
// migration runner, and small transaction helpers used by every domain
// package. SQLite is the authoritative store for the whole application.
//
// Migrations are read as loose .sql files from disk (not go:embed) on
// purpose: the XP-targeting build uses a Go toolchain that predates
// go:embed (see docs/00-xp-compatibility-report.md), and keeping one code
// path for both build tiers is simpler than maintaining two.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the shared *sql.DB handle.
type DB struct {
	*sql.DB
}

// Open opens (creating if necessary) the SQLite database at path and applies
// the pragmas required for a reliable single-writer desktop POS workload.
func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL", path)
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// WAL mode allows multiple concurrent readers alongside one writer, so a
	// small pool is safe and far more forgiving than a single connection —
	// with only one connection, any query issued while another Rows on the
	// same *sql.DB is still mid-iteration would block forever waiting for a
	// connection that can never free up. busy_timeout above absorbs the
	// remaining writer/writer contention a single-process desktop POS can
	// produce.
	sqlDB.SetMaxOpenConns(4)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{sqlDB}, nil
}

// Migrate applies every *.sql file in dir, in filename order, that has not
// already been recorded in schema_migrations. Each file runs in its own
// transaction; a failure rolls back that file only and stops the run.
func (db *DB) Migrate(dir string) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version     INTEGER PRIMARY KEY,
		name        TEXT NOT NULL,
		applied_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	applied := map[int]bool{}
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("read applied migrations: %w", err)
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()

	for _, name := range files {
		version, verErr := versionFromFilename(name)
		if verErr != nil {
			return fmt.Errorf("migration file %s: %w", name, verErr)
		}
		if applied[version] {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, version, name); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}

	return nil
}

func versionFromFilename(name string) (int, error) {
	idx := strings.Index(name, "_")
	if idx <= 0 {
		return 0, fmt.Errorf("expected NNN_name.sql, got %q", name)
	}
	return strconv.Atoi(name[:idx])
}
