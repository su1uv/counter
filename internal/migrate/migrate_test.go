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

func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2)`, table, column).Scan(&exists)
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

var (
	retailTables  = []string{"branches", "categories", "items", "employees", "sales", "sale_lines"}
	realismTables = []string{"sale_payments", "sale_returns", "sale_return_lines"}
	// realismColumns are added to retail tables by 00003.
	realismColumns = [][2]string{{"branches", "tier"}, {"employees", "terminated_on"}, {"sales", "status"}}
	// inventoryTables come from 00004; stock_levels is a view.
	inventoryTables = []string{"suppliers", "purchase_orders", "purchase_order_lines", "inventory_movements", "stock_levels"}
	cashTables      = []string{"cash_registers", "register_shifts", "cash_movements"}
)

func TestRunLifecycle(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	var out bytes.Buffer
	all := []string{"00001_schema_probe.sql", "00002_retail_core.sql", "00003_sales_realism.sql", "00004_inventory.sql", "00005_cash_register.sql"}

	steps := []struct {
		command       string
		wantOut       []string
		wantProbe     bool
		wantRetail    bool
		wantRealism   bool
		wantInventory bool
		wantCash      bool
	}{
		{"up", all, true, true, true, true, true},
		{"status", all, true, true, true, true, true},
		{"down", []string{"00005_cash_register.sql"}, true, true, true, true, false},
		{"down", []string{"00004_inventory.sql"}, true, true, true, false, false},
		{"down", []string{"00003_sales_realism.sql"}, true, true, false, false, false},
		{"down", []string{"00002_retail_core.sql"}, true, false, false, false, false},
		{"down", []string{"00001_schema_probe.sql"}, false, false, false, false, false},
		{"up", all, true, true, true, true, true},
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
		for _, table := range realismTables {
			if got := tableExists(t, db, table); got != s.wantRealism {
				t.Fatalf("after %s: %s exists = %v, want %v", s.command, table, got, s.wantRealism)
			}
		}
		for _, c := range realismColumns {
			if got := columnExists(t, db, c[0], c[1]); got != s.wantRealism {
				t.Fatalf("after %s: %s.%s exists = %v, want %v", s.command, c[0], c[1], got, s.wantRealism)
			}
		}
		for _, table := range inventoryTables {
			if got := tableExists(t, db, table); got != s.wantInventory {
				t.Fatalf("after %s: %s exists = %v, want %v", s.command, table, got, s.wantInventory)
			}
		}
		for _, table := range cashTables {
			if got := tableExists(t, db, table); got != s.wantCash {
				t.Fatalf("after %s: %s exists = %v, want %v", s.command, table, got, s.wantCash)
			}
		}
		if got := columnExists(t, db, "sales", "shift_id"); got != s.wantCash {
			t.Fatalf("after %s: sales.shift_id exists = %v, want %v", s.command, got, s.wantCash)
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
