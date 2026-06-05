package store

import (
	"os"
	"testing"

	"github.com/ashley/drama-workbench/internal/model"
)

// TestRoundTrip exercises Save -> List -> Get against a real Postgres. It is
// skipped (not failed) when DATABASE_URL is unset, so `go test ./...` stays
// green without a database. Point it at the docker-compose Postgres to run:
//   DATABASE_URL=postgres://drama:drama@localhost:5432/drama?sslmode=disable go test ./internal/store
func TestRoundTrip(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping store integration test")
	}

	s, err := Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	brief := model.Brief{Genre: "测试题材", Episodes: 2, BrandFocus: "sofas"}
	plan := &model.Plan{
		Brief:    brief,
		Bible:    model.SeriesBible{Title: "测试剧名"},
		Episodes: []model.Episode{{Number: 1}, {Number: 2}},
	}

	id, err := s.Save(brief, plan)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if id == "" {
		t.Fatal("save returned empty id")
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found bool
	for _, sm := range list {
		if sm.ID == id {
			found = true
			if sm.Title != "测试剧名" {
				t.Errorf("title = %q, want 测试剧名", sm.Title)
			}
			if sm.Genre != "测试题材" {
				t.Errorf("genre = %q, want 测试题材", sm.Genre)
			}
			if sm.Episodes != 2 {
				t.Errorf("episodes = %d, want 2", sm.Episodes)
			}
		}
	}
	if !found {
		t.Fatalf("saved id %s not present in list", id)
	}

	rec, err := s.Get(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if rec == nil {
		t.Fatal("get returned nil for existing id")
	}
	if rec.Plan.Bible.Title != "测试剧名" {
		t.Errorf("get plan title = %q, want 测试剧名", rec.Plan.Bible.Title)
	}
	if len(rec.Plan.Episodes) != 2 {
		t.Errorf("get plan episodes = %d, want 2", len(rec.Plan.Episodes))
	}
	if rec.Brief.Genre != "测试题材" {
		t.Errorf("get brief genre = %q, want 测试题材", rec.Brief.Genre)
	}

	// Non-existent id returns (nil, nil).
	missing, err := s.Get("does-not-exist")
	if err != nil {
		t.Fatalf("get missing: unexpected err %v", err)
	}
	if missing != nil {
		t.Fatal("get missing: expected nil record")
	}
}
