// Package store persists Gemini fruit classifications in SQLite so a
// dashboard can show per-label counts.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Labels lists every label a classification can be stored under, in display
// order. "other" is the catch-all for unrecognised model output.
var Labels = []string{"ripe", "unripe", "overripe", "not fruit", "other"}

// Normalize maps raw model output to one of Labels. Case, surrounding
// punctuation and quoting, and "-"/"_" separators are ignored; anything that
// is not a known label becomes "other".
func Normalize(raw string) string {
	s := strings.ToLower(raw)
	s = strings.NewReplacer("-", " ", "_", " ").Replace(s)
	s = strings.Trim(s, " \t\r\n.!?\"'")
	s = strings.Join(strings.Fields(s), " ")
	for _, l := range Labels[:4] {
		if s == l {
			return s
		}
	}
	return "other"
}

// Store is a SQLite-backed record of classifications.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database at path, including its
// parent directory, and ensures the schema exists.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
PRAGMA journal_mode=WAL;
CREATE TABLE IF NOT EXISTS classifications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  source TEXT NOT NULL,
  label TEXT NOT NULL,
  raw TEXT NOT NULL
);`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Record stores one classification from the given source ("mqtt", "http"),
// keeping both the raw model output and its normalized label.
func (s *Store) Record(ctx context.Context, source, raw string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO classifications (source, label, raw) VALUES (?, ?, ?)`,
		source, Normalize(raw), raw)
	if err != nil {
		return fmt.Errorf("record classification: %w", err)
	}
	return nil
}

// Counts returns the number of classifications per label. Every entry of
// Labels is present, with 0 when nothing has been recorded.
func (s *Store) Counts(ctx context.Context) (map[string]int, error) {
	counts := make(map[string]int, len(Labels))
	for _, l := range Labels {
		counts[l] = 0
	}
	rows, err := s.db.QueryContext(ctx, `SELECT label, COUNT(*) FROM classifications GROUP BY label`)
	if err != nil {
		return nil, fmt.Errorf("query counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var label string
		var n int
		if err := rows.Scan(&label, &n); err != nil {
			return nil, fmt.Errorf("scan counts: %w", err)
		}
		counts[label] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read counts: %w", err)
	}
	return counts, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}
