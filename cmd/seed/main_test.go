package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/su1uv/counter/internal/migrate"
	"github.com/su1uv/counter/internal/seed"
)

const migrationsDir = "../../db/migrations"

func smallConfig() seed.Config {
	cfg := seed.Default()
	cfg.Months = 1
	cfg.Branches = 2
	cfg.Items = 20
	cfg.MinSalesPerDay = 1
	cfg.MaxSalesPerDay = 3
	return cfg
}

func envFunc(env map[string]string) func(string) string {
	return func(k string) string { return env[k] }
}

func TestRunErrors(t *testing.T) {
	withURL := map[string]string{"DATABASE_URL": "postgres://localhost:1/x"}
	tests := []struct {
		name string
		args []string
		env  map[string]string
	}{
		{"unknown flag", []string{"-bogus"}, withURL},
		{"bad seed", []string{"-seed", "abc"}, withURL},
		{"extra argument", []string{"up"}, withURL},
		{"missing database url", nil, map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(context.Background(), smallConfig(), tt.args, envFunc(tt.env), &out)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(out.String(), "usage: seed") {
				t.Fatalf("output lacks usage: %q", out.String())
			}
		})
	}
}

func TestRunInvalidConfig(t *testing.T) {
	cfg := smallConfig()
	cfg.Months = 0
	env := map[string]string{"DATABASE_URL": "postgres://localhost:1/x"}
	if err := run(context.Background(), cfg, nil, envFunc(env), &bytes.Buffer{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunUnreachableDatabase(t *testing.T) {
	env := map[string]string{"DATABASE_URL": "postgres://counter@127.0.0.1:1/x?sslmode=disable"}
	if err := run(context.Background(), smallConfig(), nil, envFunc(env), &bytes.Buffer{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunAgainstDatabase(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	downAll := func() {
		for migrate.Run(ctx, db, migrationsDir, "down", &bytes.Buffer{}) == nil {
		}
	}
	downAll()
	t.Cleanup(func() {
		downAll()
		db.Close()
	})
	env := envFunc(map[string]string{"DATABASE_URL": url})

	var out bytes.Buffer
	err = run(ctx, smallConfig(), nil, env, &out)
	if err == nil || !strings.Contains(err.Error(), "make migrate-up") {
		t.Fatalf("without schema: err = %v, want migrate-up hint", err)
	}
	if err := migrate.Run(ctx, db, migrationsDir, "up", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	steps := []struct {
		name    string
		args    []string
		wantErr string
		wantOut string
	}{
		{"first seed", nil, "", "seeded 2 branches"},
		{"refuses non-empty tables", nil, "-reset", ""},
		{"reset", []string{"-reset"}, "", "payments"},
		{"reset again", []string{"-reset"}, "", "with seed 1"},
		{"reset with another seed", []string{"-reset", "-seed", "7"}, "", "with seed 7"},
	}
	for _, s := range steps {
		out.Reset()
		err := run(ctx, smallConfig(), s.args, env, &out)
		if s.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), s.wantErr) {
				t.Fatalf("%s: err = %v, want it to mention %q", s.name, err, s.wantErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if !strings.Contains(out.String(), s.wantOut) {
			t.Fatalf("%s: output %q lacks %q", s.name, out.String(), s.wantOut)
		}
	}
}
