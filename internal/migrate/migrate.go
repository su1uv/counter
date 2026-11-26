// Package migrate applies goose migrations from a directory on disk.
// It is used by cmd/migrate and tests only; the service must not import it.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Run executes command ("up", "down" or "status") against db using the
// migration files in dir. An advisory lock makes overlapping runs wait.
func Run(ctx context.Context, db *sql.DB, dir, command string, out io.Writer) error {
	switch command {
	case "up", "down", "status":
	default:
		return fmt.Errorf("unknown command %q", command)
	}
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}
	p, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(dir), goose.WithSessionLocker(locker))
	if err != nil {
		return err
	}
	switch command {
	case "up":
		results, err := p.Up(ctx)
		if err != nil {
			return err
		}
		for _, r := range results {
			fmt.Fprintf(out, "applied %s\n", r.Source.Path)
		}
	case "down":
		r, err := p.Down(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "reverted %s\n", r.Source.Path)
	case "status":
		statuses, err := p.Status(ctx)
		if err != nil {
			return err
		}
		for _, s := range statuses {
			fmt.Fprintf(out, "%-8s %s\n", s.State, s.Source.Path)
		}
	}
	return nil
}
