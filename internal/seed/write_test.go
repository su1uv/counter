package seed

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/su1uv/counter/internal/migrate"
	"github.com/su1uv/counter/internal/store"
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
	t.Cleanup(func() {
		for migrate.Run(context.Background(), db, migrationsDir, "down", &bytes.Buffer{}) == nil {
		}
		db.Close()
	})
	return db
}

func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openTestDB(t)
	if err := migrate.Run(context.Background(), db, migrationsDir, "up", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	return db
}

// fingerprint hashes every seed table in id order.
func fingerprint(t *testing.T, db *sql.DB) string {
	t.Helper()
	var fp string
	err := db.QueryRow(`SELECT concat_ws(',',
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM branches t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM categories t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM items t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM employees t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM sales t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM sale_lines t))`).Scan(&fp)
	if err != nil {
		t.Fatal(err)
	}
	return fp
}

func countLines(ds Dataset) int64 {
	var n int64
	for _, s := range ds.Sales {
		n += int64(len(s.Lines))
	}
	return n
}

func TestWrite(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()
	q := store.New(db)
	ds := mustGenerate(t, smallConfig())

	if err := Write(ctx, db, ds); err != nil {
		t.Fatal(err)
	}
	got, err := q.CountSeedRows(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := store.CountSeedRowsRow{
		Branches:   int64(len(ds.Branches)),
		Categories: int64(len(ds.Categories)),
		Items:      int64(len(ds.Items)),
		Employees:  int64(len(ds.Employees)),
		Sales:      int64(len(ds.Sales)),
		SaleLines:  countLines(ds),
	}
	if got != want {
		t.Fatalf("CountSeedRows = %+v, want %+v", got, want)
	}

	invariants := []struct {
		name, query string
	}{
		{"sale total equals its lines", `SELECT count(*) FROM sales s
			WHERE s.total_cents <> (SELECT sum(l.total_cents) FROM sale_lines l WHERE l.sale_id = s.id)`},
		{"seller works at the sale's branch", `SELECT count(*) FROM sales s
			JOIN employees e ON e.id = s.employee_id WHERE e.branch_id <> s.branch_id`},
		{"line cost matches item cost", `SELECT count(*) FROM sale_lines l
			JOIN items i ON i.id = l.item_id WHERE l.unit_cost_cents <> i.unit_cost_cents`},
		{"first sale keeps its folio", `SELECT count(*) FROM sales WHERE id = 1 AND folio <> '` + ds.Sales[0].Folio + `'`},
	}
	for _, inv := range invariants {
		var bad int
		if err := db.QueryRow(inv.query).Scan(&bad); err != nil || bad != 0 {
			t.Fatalf("%s: %d violations, %v", inv.name, bad, err)
		}
	}

	first := fingerprint(t, db)
	if err := Write(ctx, db, ds); err == nil {
		t.Fatal("second Write without reset succeeded")
	}
	if fingerprint(t, db) != first {
		t.Fatal("failed Write changed the data")
	}
	if err := q.ResetSeedData(ctx); err != nil {
		t.Fatal(err)
	}
	if err := Write(ctx, db, mustGenerate(t, smallConfig())); err != nil {
		t.Fatal(err)
	}
	if second := fingerprint(t, db); second != first {
		t.Fatalf("fingerprints differ after reset: %s vs %s", first, second)
	}
}

func TestWriteErrors(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()
	base := mustGenerate(t, smallConfig())

	// Each edit duplicates a unique key so that one insert kind fails.
	tests := []struct {
		name string
		edit func(*Dataset)
	}{
		{"branch", func(ds *Dataset) { ds.Branches[1].Code = ds.Branches[0].Code }},
		{"category", func(ds *Dataset) { ds.Categories[1].Code = ds.Categories[0].Code }},
		{"item", func(ds *Dataset) { ds.Items[1].SKU = ds.Items[0].SKU }},
		{"employee", func(ds *Dataset) { ds.Employees[1].Code = ds.Employees[0].Code }},
		{"sale", func(ds *Dataset) { ds.Sales[1].Folio = ds.Sales[0].Folio }},
		{"sale line", func(ds *Dataset) { ds.Sales[0].Lines = append(ds.Sales[0].Lines, ds.Sales[0].Lines[0]) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := mustGenerate(t, smallConfig())
			tt.edit(&ds)
			if err := Write(ctx, db, ds); err == nil {
				t.Fatal("expected error")
			}
			if got, err := store.New(db).CountSeedRows(ctx); err != nil || got != (store.CountSeedRowsRow{}) {
				t.Fatalf("after failed Write CountSeedRows = %+v, %v; want zero", got, err)
			}
		})
	}

	t.Run("cancelled context", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if err := Write(cancelled, db, base); err == nil {
			t.Fatal("expected error")
		}
	})
}
