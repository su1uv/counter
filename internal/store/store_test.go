package store_test

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

func TestQueriesAgainstMigratedSchema(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := migrate.Run(ctx, db, "../../db/migrations", "up", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = migrate.Run(ctx, db, "../../db/migrations", "down", &bytes.Buffer{}) })

	q := store.New(db)
	if got, err := q.Ping(ctx); err != nil || got != 1 {
		t.Fatalf("Ping = %d, %v; want 1, nil", got, err)
	}
	if got, err := q.CountSchemaProbe(ctx); err != nil || got != 0 {
		t.Fatalf("CountSchemaProbe = %d, %v; want 0, nil", got, err)
	}
}
