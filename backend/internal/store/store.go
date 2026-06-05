// Package store persists generated plans in Postgres so the workbench can
// surface a history of past productions. It is intentionally thin: a single
// table, the full Brief/Plan kept as jsonb, plus a few denormalized columns to
// make the history list cheap to render. When DATABASE_URL is unset the server
// runs without a Store (graceful degradation) — none of this code is reached.
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/ashley/drama-workbench/internal/model"
)

type Store struct {
	db *sql.DB
}

// Summary is a single row in the history list (lightweight).
type Summary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Genre     string    `json:"genre"`
	Episodes  int       `json:"episodes"`
	CreatedAt time.Time `json:"createdAt"`
}

// Record is a full stored plan, returned by the detail endpoint.
type Record struct {
	ID        string      `json:"id"`
	CreatedAt time.Time   `json:"createdAt"`
	Brief     model.Brief `json:"brief"`
	Plan      model.Plan  `json:"plan"`
}

// Open dials the database and verifies connectivity with a Ping.
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the underlying connection pool.
func (s *Store) Close() error { return s.db.Close() }

// Migrate creates the plans table if it does not already exist.
func (s *Store) Migrate() error {
	const ddl = `CREATE TABLE IF NOT EXISTS plans(
		id text primary key,
		created_at timestamptz not null default now(),
		genre text,
		title text,
		episodes int,
		brief jsonb not null,
		plan jsonb not null
	)`
	if _, err := s.db.Exec(ddl); err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}
	return nil
}

// Save persists a generated plan and returns its new id. brief/plan are stored
// as jsonb; genre/title/episodes are pulled into columns to keep List() cheap.
func (s *Store) Save(brief model.Brief, plan *model.Plan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("store: save: nil plan")
	}
	id, err := newID()
	if err != nil {
		return "", err
	}
	briefJSON, err := json.Marshal(brief)
	if err != nil {
		return "", fmt.Errorf("store: marshal brief: %w", err)
	}
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return "", fmt.Errorf("store: marshal plan: %w", err)
	}
	const q = `INSERT INTO plans(id, genre, title, episodes, brief, plan)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = s.db.Exec(q, id, brief.Genre, plan.Bible.Title, len(plan.Episodes), briefJSON, planJSON)
	if err != nil {
		return "", fmt.Errorf("store: insert: %w", err)
	}
	return id, nil
}

// List returns up to 100 plan summaries, newest first.
func (s *Store) List() ([]Summary, error) {
	const q = `SELECT id, title, genre, episodes, created_at
		FROM plans ORDER BY created_at DESC LIMIT 100`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("store: list: %w", err)
	}
	defer rows.Close()

	out := []Summary{}
	for rows.Next() {
		var sm Summary
		var title, genre sql.NullString
		var episodes sql.NullInt64
		if err := rows.Scan(&sm.ID, &title, &genre, &episodes, &sm.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: list scan: %w", err)
		}
		sm.Title = title.String
		sm.Genre = genre.String
		sm.Episodes = int(episodes.Int64)
		out = append(out, sm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list rows: %w", err)
	}
	return out, nil
}

// Get returns the full record for an id, or (nil, nil) if it does not exist.
func (s *Store) Get(id string) (*Record, error) {
	const q = `SELECT id, created_at, brief, plan FROM plans WHERE id = $1`
	var rec Record
	var briefJSON, planJSON []byte
	err := s.db.QueryRow(q, id).Scan(&rec.ID, &rec.CreatedAt, &briefJSON, &planJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get: %w", err)
	}
	if err := json.Unmarshal(briefJSON, &rec.Brief); err != nil {
		return nil, fmt.Errorf("store: unmarshal brief: %w", err)
	}
	if err := json.Unmarshal(planJSON, &rec.Plan); err != nil {
		return nil, fmt.Errorf("store: unmarshal plan: %w", err)
	}
	return &rec, nil
}

// newID returns 16 random bytes hex-encoded (32 chars).
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("store: rand: %w", err)
	}
	return hex.EncodeToString(b), nil
}
