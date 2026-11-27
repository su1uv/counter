package store_test

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/su1uv/counter/internal/migrate"
	"github.com/su1uv/counter/internal/store"
)

const migrationsDir = "../../db/migrations"

func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := migrate.Run(ctx, db, migrationsDir, "up", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for migrate.Run(ctx, db, migrationsDir, "down", &bytes.Buffer{}) == nil {
		}
		db.Close()
	})
	return db
}

func TestQueriesAgainstMigratedSchema(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	q := store.New(db)
	if got, err := q.Ping(ctx); err != nil || got != 1 {
		t.Fatalf("Ping = %d, %v; want 1, nil", got, err)
	}
	if got, err := q.CountSchemaProbe(ctx); err != nil || got != 0 {
		t.Fatalf("CountSchemaProbe = %d, %v; want 0, nil", got, err)
	}
}

func TestSeedQueries(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()
	q := store.New(db)
	day := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	branchID, err := q.InsertBranch(ctx, store.InsertBranchParams{
		Code: "S01", Name: "Centro", City: "Ciudad de México", State: "CDMX",
		Timezone: "America/Mexico_City", OpenedOn: day, Tier: "large",
	})
	if err != nil {
		t.Fatal(err)
	}
	categoryID, err := q.InsertCategory(ctx, store.InsertCategoryParams{Code: "AN", Name: "Anillos"})
	if err != nil {
		t.Fatal(err)
	}
	itemID, err := q.InsertItem(ctx, store.InsertItemParams{
		Sku: "AN-ORO14-0001", CategoryID: categoryID, Name: "Anillo solitario de oro 14k",
		Material: "oro 14k", UnitCostCents: 400000, ListPriceCents: 1000000, Active: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	employeeID, err := q.InsertEmployee(ctx, store.InsertEmployeeParams{
		Code: "S01-E01", BranchID: branchID, FullName: "Ana López García", Role: "manager", HiredOn: day,
		TerminatedOn: sql.NullTime{Time: day.AddDate(0, 1, 0), Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	saleID, err := q.InsertSale(ctx, store.InsertSaleParams{
		Folio: "S01-000001", BranchID: branchID, EmployeeID: employeeID, SoldAt: day.Add(12 * time.Hour),
		Status: "completed", SubtotalCents: 862069, TaxCents: 137931, TotalCents: 1000000,
	})
	if err != nil {
		t.Fatal(err)
	}
	lineID, err := q.InsertSaleLine(ctx, store.InsertSaleLineParams{
		SaleID: saleID, LineNo: 1, ItemID: itemID, Quantity: 1,
		UnitPriceCents: 1000000, UnitCostCents: 400000, DiscountCents: 0, TotalCents: 1000000,
	})
	if err != nil {
		t.Fatal(err)
	}
	payments := []store.InsertSalePaymentParams{
		{SaleID: saleID, Method: "card", AmountCents: 600000,
			CardBrand: sql.NullString{String: "Visa", Valid: true}, Last4: sql.NullString{String: "0042", Valid: true}},
		{SaleID: saleID, Method: "cash", AmountCents: 400000},
	}
	for _, p := range payments {
		if _, err := q.InsertSalePayment(ctx, p); err != nil {
			t.Fatalf("InsertSalePayment(%s): %v", p.Method, err)
		}
	}
	rejected := []struct {
		name string
		p    store.InsertSalePaymentParams
	}{
		{"card without brand", store.InsertSalePaymentParams{SaleID: saleID, Method: "card", AmountCents: 1,
			Last4: sql.NullString{String: "0042", Valid: true}}},
		{"cash with last4", store.InsertSalePaymentParams{SaleID: saleID, Method: "cash", AmountCents: 1,
			Last4: sql.NullString{String: "0042", Valid: true}}},
		{"bad last4", store.InsertSalePaymentParams{SaleID: saleID, Method: "card", AmountCents: 1,
			CardBrand: sql.NullString{String: "Visa", Valid: true}, Last4: sql.NullString{String: "42", Valid: true}}},
		{"zero amount", store.InsertSalePaymentParams{SaleID: saleID, Method: "cash"}},
		{"unknown method", store.InsertSalePaymentParams{SaleID: saleID, Method: "cheque", AmountCents: 1}},
	}
	for _, r := range rejected {
		if _, err := q.InsertSalePayment(ctx, r.p); err == nil {
			t.Fatalf("InsertSalePayment(%s) succeeded, want a check violation", r.name)
		}
	}
	returnID, err := q.InsertSaleReturn(ctx, store.InsertSaleReturnParams{
		Folio: "S01-000001-D", SaleID: saleID, EmployeeID: employeeID, ReturnedAt: day.Add(36 * time.Hour),
		SubtotalCents: 862069, TaxCents: 137931, TotalCents: 1000000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.InsertSaleReturnLine(ctx, store.InsertSaleReturnLineParams{
		ReturnID: returnID, LineNo: 1, SaleLineID: lineID, Quantity: 1, RefundCents: 1000000,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := q.CountSeedRows(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := store.CountSeedRowsRow{
		Branches: 1, Categories: 1, Items: 1, Employees: 1, Sales: 1, SaleLines: 1,
		SalePayments: 2, SaleReturns: 1, SaleReturnLines: 1,
	}
	if got != want {
		t.Fatalf("CountSeedRows = %+v, want %+v", got, want)
	}

	if err := q.ResetSeedData(ctx); err != nil {
		t.Fatal(err)
	}
	if got, err := q.CountSeedRows(ctx); err != nil || got != (store.CountSeedRowsRow{}) {
		t.Fatalf("after reset CountSeedRows = %+v, %v; want zero", got, err)
	}
	again, err := q.InsertBranch(ctx, store.InsertBranchParams{
		Code: "S01", Name: "Centro", City: "Ciudad de México", State: "CDMX",
		Timezone: "America/Mexico_City", OpenedOn: day, Tier: "large",
	})
	if err != nil || again != 1 {
		t.Fatalf("InsertBranch after reset = %d, %v; want identity restarted at 1", again, err)
	}
}
