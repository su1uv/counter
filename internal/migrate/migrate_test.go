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

func TestRunLifecycle(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	var out bytes.Buffer

	steps := []struct {
		command     string
		wantOut     string
		wantTable   bool
		checkTables bool
	}{
		{"up", "00001_schema_probe.sql", true, true},
		{"status", "00001_schema_probe.sql", true, true},
		{"down", "00001_schema_probe.sql", false, true},
		{"up", "00001_schema_probe.sql", true, true},
	}
	t.Cleanup(func() { _ = Run(ctx, db, migrationsDir, "down", &bytes.Buffer{}) })
	for _, s := range steps {
		out.Reset()
		if err := Run(ctx, db, migrationsDir, s.command, &out); err != nil {
			t.Fatalf("%s: %v", s.command, err)
		}
		if !strings.Contains(out.String(), s.wantOut) {
			t.Fatalf("%s output %q lacks %q", s.command, out.String(), s.wantOut)
		}
		if got := tableExists(t, db, "schema_probe"); got != s.wantTable {
			t.Fatalf("after %s: schema_probe exists = %v, want %v", s.command, got, s.wantTable)
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
