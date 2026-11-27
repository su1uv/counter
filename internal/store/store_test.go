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
	registerID, err := q.InsertCashRegister(ctx, store.InsertCashRegisterParams{
		BranchID: branchID, Code: "S01-C1", Name: "Caja 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.InsertCashRegister(ctx, store.InsertCashRegisterParams{
		BranchID: branchID, Code: "S01-C1", Name: "Caja 1",
	}); err == nil {
		t.Fatal("second register with the same code succeeded, want a unique violation")
	}
	opened := day.Add(11 * time.Hour)
	shiftID, err := q.InsertRegisterShift(ctx, store.InsertRegisterShiftParams{
		RegisterID: registerID, OpenedBy: employeeID, OpenedAt: opened, ClosedAt: opened.Add(10 * time.Hour),
		OpeningFloatCents: 300000, ExpectedCashCents: 250000, CountedCashCents: 248000,
	})
	if err != nil {
		t.Fatal(err)
	}
	rejectedShifts := []struct {
		name string
		p    store.InsertRegisterShiftParams
	}{
		{"closes when it opens", store.InsertRegisterShiftParams{
			RegisterID: registerID, OpenedBy: employeeID, OpenedAt: opened, ClosedAt: opened,
			OpeningFloatCents: 300000, ExpectedCashCents: 0, CountedCashCents: 0}},
		{"negative float", store.InsertRegisterShiftParams{
			RegisterID: registerID, OpenedBy: employeeID, OpenedAt: opened, ClosedAt: opened.Add(time.Hour),
			OpeningFloatCents: -1, ExpectedCashCents: 0, CountedCashCents: 0}},
	}
	for _, r := range rejectedShifts {
		if _, err := q.InsertRegisterShift(ctx, r.p); err == nil {
			t.Fatalf("InsertRegisterShift(%s) succeeded, want a check violation", r.name)
		}
	}
	saleID, err := q.InsertSale(ctx, store.InsertSaleParams{
		Folio: "S01-000001", BranchID: branchID, EmployeeID: employeeID, SoldAt: day.Add(12 * time.Hour),
		Status: "completed", SubtotalCents: 862069, TaxCents: 137931, TotalCents: 1000000,
		ShiftID: sql.NullInt64{Int64: shiftID, Valid: true},
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
	returnLineID, err := q.InsertSaleReturnLine(ctx, store.InsertSaleReturnLineParams{
		ReturnID: returnID, LineNo: 1, SaleLineID: lineID, Quantity: 1, RefundCents: 1000000,
	})
	if err != nil {
		t.Fatal(err)
	}

	supplierID, err := q.InsertSupplier(ctx, store.InsertSupplierParams{
		Code: "PR01", Name: "Oro Fino de Guadalajara", City: "Guadalajara", LeadTimeDays: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	orderID, err := q.InsertPurchaseOrder(ctx, store.InsertPurchaseOrderParams{
		Folio: "OC-S01-000001", SupplierID: supplierID, BranchID: branchID, OrderedAt: day.Add(-48 * time.Hour),
		ReceivedAt: sql.NullTime{Time: day.Add(10 * time.Hour), Valid: true}, Status: "received",
	})
	if err != nil {
		t.Fatal(err)
	}
	orderLineID, err := q.InsertPurchaseOrderLine(ctx, store.InsertPurchaseOrderLineParams{
		PurchaseOrderID: orderID, LineNo: 1, ItemID: itemID, Quantity: 2, UnitCostCents: 400000,
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := func(id int64) sql.NullInt64 { return sql.NullInt64{Int64: id, Valid: true} }
	movement := func(kind string, qty int32) store.InsertInventoryMovementParams {
		return store.InsertInventoryMovementParams{
			BranchID: branchID, ItemID: itemID, MovedAt: day, Kind: kind, Quantity: qty,
			UnitCostCents: 400000, Reference: "REF",
		}
	}
	withRef := func(p store.InsertInventoryMovementParams, set func(*store.InsertInventoryMovementParams)) store.InsertInventoryMovementParams {
		set(&p)
		return p
	}
	movements := []store.InsertInventoryMovementParams{
		movement("opening", 3),
		withRef(movement("receipt", 2), func(p *store.InsertInventoryMovementParams) { p.PurchaseOrderLineID = ref(orderLineID) }),
		withRef(movement("sale", -1), func(p *store.InsertInventoryMovementParams) { p.SaleLineID = ref(lineID) }),
		withRef(movement("return", 1), func(p *store.InsertInventoryMovementParams) { p.SaleReturnLineID = ref(returnLineID) }),
		movement("transfer_out", -2),
		movement("transfer_in", 1),
		movement("adjustment", -1),
	}
	// Rejected rows go first, so the unique source-line columns cannot mask
	// the checks under test.
	rejectedMovements := []struct {
		name string
		p    store.InsertInventoryMovementParams
	}{
		{"positive sale", withRef(movement("sale", 1), func(p *store.InsertInventoryMovementParams) { p.SaleLineID = ref(lineID) })},
		{"sale without line", movement("sale", -1)},
		{"receipt without order line", movement("receipt", 1)},
		{"return without return line", movement("return", 1)},
		{"opening with sale line", withRef(movement("opening", 1), func(p *store.InsertInventoryMovementParams) { p.SaleLineID = ref(lineID) })},
		{"negative opening", movement("opening", -1)},
		{"positive transfer out", movement("transfer_out", 1)},
		{"zero adjustment", movement("adjustment", 0)},
		{"unknown kind", movement("theft", -1)},
	}
	for _, r := range rejectedMovements {
		if _, err := q.InsertInventoryMovement(ctx, r.p); err == nil {
			t.Fatalf("InsertInventoryMovement(%s) succeeded, want a check violation", r.name)
		}
	}
	for _, m := range movements {
		if _, err := q.InsertInventoryMovement(ctx, m); err != nil {
			t.Fatalf("InsertInventoryMovement(%s): %v", m.Kind, err)
		}
	}
	if _, err := q.InsertInventoryMovement(ctx, movements[2]); err == nil {
		t.Fatal("second movement for one sale line succeeded, want a unique violation")
	}
	if got, err := q.GetStockLevel(ctx, store.GetStockLevelParams{BranchID: branchID, ItemID: itemID}); err != nil || got != 3 {
		t.Fatalf("GetStockLevel = %d, %v; want 3", got, err)
	}
	rejectedOrders := []struct {
		name string
		p    store.InsertPurchaseOrderParams
	}{
		{"received without date", store.InsertPurchaseOrderParams{Folio: "OC-X1", SupplierID: supplierID, BranchID: branchID,
			OrderedAt: day, Status: "received"}},
		{"open with date", store.InsertPurchaseOrderParams{Folio: "OC-X2", SupplierID: supplierID, BranchID: branchID,
			OrderedAt: day, ReceivedAt: sql.NullTime{Time: day.Add(time.Hour), Valid: true}, Status: "ordered"}},
		{"received before ordered", store.InsertPurchaseOrderParams{Folio: "OC-X3", SupplierID: supplierID, BranchID: branchID,
			OrderedAt: day, ReceivedAt: sql.NullTime{Time: day.Add(-time.Hour), Valid: true}, Status: "received"}},
	}
	for _, r := range rejectedOrders {
		if _, err := q.InsertPurchaseOrder(ctx, r.p); err == nil {
			t.Fatalf("InsertPurchaseOrder(%s) succeeded, want a check violation", r.name)
		}
	}

	rejectedCash := []struct {
		name string
		p    store.InsertCashMovementParams
	}{
		{"zero amount", store.InsertCashMovementParams{ShiftID: shiftID, MovedAt: opened.Add(time.Hour), Kind: "expense", Note: "Gasto menor"}},
		{"unknown kind", store.InsertCashMovementParams{ShiftID: shiftID, MovedAt: opened.Add(time.Hour), Kind: "deposit", AmountCents: 100, Note: "Fondo"}},
	}
	for _, r := range rejectedCash {
		if _, err := q.InsertCashMovement(ctx, r.p); err == nil {
			t.Fatalf("InsertCashMovement(%s) succeeded, want a check violation", r.name)
		}
	}
	if _, err := q.InsertCashMovement(ctx, store.InsertCashMovementParams{
		ShiftID: shiftID, MovedAt: opened.Add(8 * time.Hour), Kind: "withdrawal", AmountCents: 50000, Note: "Retiro a caja fuerte",
	}); err != nil {
		t.Fatal(err)
	}

	customerID, err := q.InsertCustomer(ctx, store.InsertCustomerParams{
		Code: "C0001", Rfc: "XAXX010101000", LegalName: "Público en general",
		TaxRegime: "616", PostalCode: "06000",
	})
	if err != nil {
		t.Fatal(err)
	}
	rejectedCustomers := []struct {
		name string
		p    store.InsertCustomerParams
	}{
		{"bad rfc", store.InsertCustomerParams{Code: "C0002", Rfc: "not-an-rfc", LegalName: "Ana López", TaxRegime: "612", PostalCode: "11560"}},
		{"bad regime", store.InsertCustomerParams{Code: "C0002", Rfc: "LOGL900315AB1", LegalName: "Ana López", TaxRegime: "999", PostalCode: "11560"}},
		{"bad postal code", store.InsertCustomerParams{Code: "C0002", Rfc: "LOGL900315AB1", LegalName: "Ana López", TaxRegime: "612", PostalCode: "1156"}},
		{"duplicate rfc", store.InsertCustomerParams{Code: "C0002", Rfc: "XAXX010101000", LegalName: "Otra", TaxRegime: "616", PostalCode: "06000"}},
	}
	for _, r := range rejectedCustomers {
		if _, err := q.InsertCustomer(ctx, r.p); err == nil {
			t.Fatalf("InsertCustomer(%s) succeeded, want a violation", r.name)
		}
	}
	if _, err := q.InsertInvoice(ctx, store.InsertInvoiceParams{
		SaleID: saleID, CustomerID: customerID, Uuid: "11111111-1111-4111-8111-111111111111",
		CfdiUse: "S01", PaymentForm: "04", PaymentMethod: "PUE", IssuedAt: day.Add(12 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	rejectedInvoices := []struct {
		name string
		p    store.InsertInvoiceParams
	}{
		{"bad use", store.InsertInvoiceParams{SaleID: saleID, CustomerID: customerID, Uuid: "22222222-2222-4222-8222-222222222222", CfdiUse: "D01", PaymentForm: "01", PaymentMethod: "PUE", IssuedAt: day}},
		{"bad form", store.InsertInvoiceParams{SaleID: saleID, CustomerID: customerID, Uuid: "22222222-2222-4222-8222-222222222222", CfdiUse: "G03", PaymentForm: "99", PaymentMethod: "PUE", IssuedAt: day}},
		{"deferred payment", store.InsertInvoiceParams{SaleID: saleID, CustomerID: customerID, Uuid: "22222222-2222-4222-8222-222222222222", CfdiUse: "G03", PaymentForm: "01", PaymentMethod: "PPD", IssuedAt: day}},
		{"bad uuid", store.InsertInvoiceParams{SaleID: saleID, CustomerID: customerID, Uuid: "not-a-uuid", CfdiUse: "G03", PaymentForm: "01", PaymentMethod: "PUE", IssuedAt: day}},
		{"second invoice for the sale", store.InsertInvoiceParams{SaleID: saleID, CustomerID: customerID, Uuid: "22222222-2222-4222-8222-222222222222", CfdiUse: "S01", PaymentForm: "01", PaymentMethod: "PUE", IssuedAt: day}},
	}
	for _, r := range rejectedInvoices {
		if _, err := q.InsertInvoice(ctx, r.p); err == nil {
			t.Fatalf("InsertInvoice(%s) succeeded, want a check violation", r.name)
		}
	}

	got, err := q.CountSeedRows(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := store.CountSeedRowsRow{
		Branches: 1, Categories: 1, Items: 1, Employees: 1, Sales: 1, SaleLines: 1,
		SalePayments: 2, SaleReturns: 1, SaleReturnLines: 1,
		Suppliers: 1, PurchaseOrders: 1, PurchaseOrderLines: 1, InventoryMovements: int64(len(movements)),
		CashRegisters: 1, RegisterShifts: 1, CashMovements: 1,
		Customers: 1, Invoices: 1,
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
