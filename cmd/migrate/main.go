// Command migrate applies database migrations from files on disk.
// Run it once per deploy, before any service instance starts.
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

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/su1uv/counter/internal/migrate"
)

const usage = "usage: migrate [-dir db/migrations] up|down|status"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout io.Writer) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", "db/migrations", "directory with migration files")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stdout, usage)
		return err
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stdout, usage)
		return errors.New("expected exactly one command")
	}
	command := fs.Arg(0)
	switch command {
	case "up", "down", "status":
	default:
		fmt.Fprintln(stdout, usage)
		return fmt.Errorf("unknown command %q", command)
	}
	url := getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(stdout, usage)
		return errors.New("DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer db.Close()
	return migrate.Run(ctx, db, *dir, command, stdout)
}
