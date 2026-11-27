package migrate

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const migrationsDir = "../../db/migrations"

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, name).Scan(&exists)
	if err != nil {
		t.Fatal(err)
	}
	return exists
}

// downAll reverts every applied migration; "down" reverts one at a time.
func downAll(ctx context.Context, db *sql.DB) {
	for Run(ctx, db, migrationsDir, "down", &bytes.Buffer{}) == nil {
	}
}

var retailTables = []string{"branches", "categories", "items", "employees", "sales", "sale_lines"}

func TestRunLifecycle(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	var out bytes.Buffer

	steps := []struct {
		command    string
		wantOut    []string
		wantProbe  bool
		wantRetail bool
	}{
		{"up", []string{"00001_schema_probe.sql", "00002_retail_core.sql"}, true, true},
		{"status", []string{"00001_schema_probe.sql", "00002_retail_core.sql"}, true, true},
		{"down", []string{"00002_retail_core.sql"}, true, false},
		{"down", []string{"00001_schema_probe.sql"}, false, false},
		{"up", []string{"00001_schema_probe.sql", "00002_retail_core.sql"}, true, true},
	}
	downAll(ctx, db)
	t.Cleanup(func() { downAll(ctx, db) })
	for _, s := range steps {
		out.Reset()
		if err := Run(ctx, db, migrationsDir, s.command, &out); err != nil {
			t.Fatalf("%s: %v", s.command, err)
		}
		for _, want := range s.wantOut {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("%s output %q lacks %q", s.command, out.String(), want)
			}
		}
		if got := tableExists(t, db, "schema_probe"); got != s.wantProbe {
			t.Fatalf("after %s: schema_probe exists = %v, want %v", s.command, got, s.wantProbe)
		}
		for _, table := range retailTables {
			if got := tableExists(t, db, table); got != s.wantRetail {
				t.Fatalf("after %s: %s exists = %v, want %v", s.command, table, got, s.wantRetail)
			}
		}
	}
}

func TestRunErrors(t *testing.T) {
	db := openTestDB(t)
	tests := []struct {
		name, dir, command string
	}{
		{"missing dir", "does-not-exist", "up"},
		{"unknown command", migrationsDir, "sideways"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Run(context.Background(), db, tt.dir, tt.command, &bytes.Buffer{}); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
