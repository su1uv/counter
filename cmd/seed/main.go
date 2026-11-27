// Command seed fills an already migrated database with deterministic
// synthetic data. It never runs migrations; run `make migrate-up` first.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/su1uv/counter/internal/seed"
	"github.com/su1uv/counter/internal/store"
)

const usage = "usage: seed [-seed N] [-reset]"

// undefinedTable is the Postgres SQLSTATE for a missing table.
const undefinedTable = "42P01"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, seed.Default(), os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run seeds with base, overridden by the flags in args.
func run(ctx context.Context, base seed.Config, args []string, getenv func(string) string, stdout io.Writer) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	seedValue := fs.Uint64("seed", base.Seed, "random seed; the same value always gives the same data")
	reset := fs.Bool("reset", false, "truncate the seed tables before seeding")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stdout, usage)
		return err
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stdout, usage)
		return errors.New("unexpected arguments")
	}
	url := getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(stdout, usage)
		return errors.New("DATABASE_URL is required")
	}

	began := time.Now()
	cfg := base
	cfg.Seed = *seedValue
	ds, err := seed.Generate(cfg)
	if err != nil {
		return err
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer db.Close()
	q := store.New(db)

	counts, err := q.CountSeedRows(ctx)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == undefinedTable {
		return errors.New("seed tables are missing: run `make migrate-up` first")
	}
	if err != nil {
		return err
	}
	if *reset {
		if err := q.ResetSeedData(ctx); err != nil {
			return err
		}
	} else if counts != (store.CountSeedRowsRow{}) {
		return fmt.Errorf("database already has seed data (%d branches, %d sales); rerun with -reset to replace it",
			counts.Branches, counts.Sales)
	}
	if err := seed.Write(ctx, db, ds); err != nil {
		return err
	}

	lines, payments := 0, 0
	for _, s := range ds.Sales {
		lines += len(s.Lines)
		payments += len(s.Payments)
	}
	fmt.Fprintf(stdout, "seeded %d branches, %d categories, %d items, %d employees, %d sales, %d sale lines, %d payments, %d returns, %d purchase orders, %d inventory movements with seed %d in %s\n",
		len(ds.Branches), len(ds.Categories), len(ds.Items), len(ds.Employees), len(ds.Sales), lines, payments, len(ds.Returns),
		len(ds.PurchaseOrders), len(ds.Movements), cfg.Seed, time.Since(began).Round(time.Millisecond))
	return nil
}
