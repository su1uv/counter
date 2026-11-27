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
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM sale_lines t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM sale_payments t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM sale_returns t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM sale_return_lines t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM suppliers t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM purchase_orders t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM purchase_order_lines t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM inventory_movements t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM cash_registers t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM register_shifts t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM cash_movements t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM customers t),
		(SELECT md5(string_agg(t::text, '|' ORDER BY id)) FROM invoices t))`).Scan(&fp)
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

func countPayments(ds Dataset) int64 {
	var n int64
	for _, s := range ds.Sales {
		n += int64(len(s.Payments))
	}
	return n
}

func countReturnLines(ds Dataset) int64 {
	var n int64
	for _, r := range ds.Returns {
		n += int64(len(r.Lines))
	}
	return n
}

func countOrderLines(ds Dataset) int64 {
	var n int64
	for _, po := range ds.PurchaseOrders {
		n += int64(len(po.Lines))
	}
	return n
}

// zeroViolations runs queries that count rows breaking a rule.
func zeroViolations(t *testing.T, db *sql.DB, rules []struct{ name, query string }) {
	t.Helper()
	for _, rule := range rules {
		var bad int
		if err := db.QueryRow(rule.query).Scan(&bad); err != nil || bad != 0 {
			t.Fatalf("%s: %d violations, %v", rule.name, bad, err)
		}
	}
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
		Branches:        int64(len(ds.Branches)),
		Categories:      int64(len(ds.Categories)),
		Items:           int64(len(ds.Items)),
		Employees:       int64(len(ds.Employees)),
		Sales:           int64(len(ds.Sales)),
		SaleLines:       countLines(ds),
		SalePayments:    countPayments(ds),
		SaleReturns:     int64(len(ds.Returns)),
		SaleReturnLines: countReturnLines(ds),

		Suppliers:          int64(len(ds.Suppliers)),
		PurchaseOrders:     int64(len(ds.PurchaseOrders)),
		PurchaseOrderLines: countOrderLines(ds),
		InventoryMovements: int64(len(ds.Movements)),
		CashRegisters:      int64(len(ds.Registers)),
		RegisterShifts:     int64(len(ds.Shifts)),
		CashMovements:      int64(len(ds.CashMovements)),
		Customers:          int64(len(ds.Customers)),
		Invoices:           int64(len(ds.Invoices)),
	}
	if want.SaleReturns == 0 || want.RegisterShifts == 0 || want.Customers == 0 || want.Invoices == 0 {
		t.Fatal("small dataset has no returns, shifts, customers or invoices to write")
	}
	if got != want {
		t.Fatalf("CountSeedRows = %+v, want %+v", got, want)
	}

	zeroViolations(t, db, []struct{ name, query string }{
		{"sale total equals its lines", `SELECT count(*) FROM sales s
			WHERE s.total_cents <> (SELECT sum(l.total_cents) FROM sale_lines l WHERE l.sale_id = s.id)`},
		{"seller works at the sale's branch", `SELECT count(*) FROM sales s
			JOIN employees e ON e.id = s.employee_id WHERE e.branch_id <> s.branch_id`},
		{"line cost matches item cost", `SELECT count(*) FROM sale_lines l
			JOIN items i ON i.id = l.item_id WHERE l.unit_cost_cents <> i.unit_cost_cents`},
		{"payments add up to each completed sale total", `SELECT count(*) FROM sales s
			WHERE s.status = 'completed'
			AND s.total_cents <> (SELECT coalesce(sum(p.amount_cents), 0) FROM sale_payments p WHERE p.sale_id = s.id)`},
		{"cancelled sales have no payments or returns", `SELECT count(*) FROM sales s
			WHERE s.status = 'cancelled' AND (EXISTS (SELECT 1 FROM sale_payments p WHERE p.sale_id = s.id)
			OR EXISTS (SELECT 1 FROM sale_returns r WHERE r.sale_id = s.id))`},
		{"returned quantity at most sold quantity", `SELECT count(*) FROM sale_lines l
			WHERE l.quantity < (SELECT coalesce(sum(rl.quantity), 0) FROM sale_return_lines rl WHERE rl.sale_line_id = l.id)`},
		{"return line belongs to the returned sale", `SELECT count(*) FROM sale_return_lines rl
			JOIN sale_returns r ON r.id = rl.return_id JOIN sale_lines l ON l.id = rl.sale_line_id
			WHERE l.sale_id <> r.sale_id`},
		{"return total equals its refunds", `SELECT count(*) FROM sale_returns r
			WHERE r.total_cents <> (SELECT sum(rl.refund_cents) FROM sale_return_lines rl WHERE rl.return_id = r.id)`},
		{"return within 30 days, at the sale's branch", `SELECT count(*) FROM sale_returns r
			JOIN sales s ON s.id = r.sale_id JOIN employees e ON e.id = r.employee_id
			WHERE r.returned_at <= s.sold_at OR r.returned_at > s.sold_at + interval '31 days' OR e.branch_id <> s.branch_id`},
		{"first sale keeps its folio", `SELECT count(*) FROM sales WHERE id = 1 AND folio <> '` + ds.Sales[0].Folio + `'`},
		{"every branch stocks every item", `SELECT count(*) FROM branches b CROSS JOIN items i
			WHERE NOT EXISTS (SELECT 1 FROM stock_levels s WHERE s.branch_id = b.id AND s.item_id = i.id)`},
		{"movement cost matches item cost", `SELECT count(*) FROM inventory_movements m
			JOIN items i ON i.id = m.item_id WHERE m.unit_cost_cents <> i.unit_cost_cents`},
		{"one sale movement per completed sale line, none for cancelled", `SELECT count(*) FROM sale_lines l
			JOIN sales s ON s.id = l.sale_id LEFT JOIN inventory_movements m ON m.sale_line_id = l.id
			WHERE (s.status = 'completed') <> (m.id IS NOT NULL)
			OR m.quantity <> -l.quantity OR m.item_id <> l.item_id OR m.branch_id <> s.branch_id OR m.moved_at <> s.sold_at`},
		{"ledger sale quantity equals completed sale lines", `SELECT count(*) FROM (SELECT
			(SELECT -sum(quantity) FROM inventory_movements WHERE kind = 'sale') AS ledger,
			(SELECT sum(l.quantity) FROM sale_lines l JOIN sales s ON s.id = l.sale_id WHERE s.status = 'completed') AS sold) x
			WHERE ledger <> sold`},
		{"one return movement per return line", `SELECT count(*) FROM sale_return_lines rl
			JOIN sale_returns r ON r.id = rl.return_id JOIN sale_lines l ON l.id = rl.sale_line_id
			JOIN sales s ON s.id = r.sale_id LEFT JOIN inventory_movements m ON m.sale_return_line_id = rl.id
			WHERE m.id IS NULL OR m.quantity <> rl.quantity OR m.item_id <> l.item_id
			OR m.branch_id <> s.branch_id OR m.moved_at <> r.returned_at`},
		{"no stock level negative", `SELECT count(*) FROM stock_levels WHERE quantity < 0`},
		{"stock never negative", `SELECT count(*) FROM (
			SELECT sum(quantity) OVER (PARTITION BY branch_id, item_id ORDER BY moved_at, id) AS balance
			FROM inventory_movements) x WHERE balance < 0`},
		{"one receipt per received order line, none for open orders", `SELECT count(*) FROM purchase_order_lines pl
			JOIN purchase_orders po ON po.id = pl.purchase_order_id
			LEFT JOIN inventory_movements m ON m.purchase_order_line_id = pl.id
			WHERE (po.status = 'received') <> (m.id IS NOT NULL)
			OR m.quantity <> pl.quantity OR m.item_id <> pl.item_id OR m.branch_id <> po.branch_id OR m.moved_at <> po.received_at`},
		{"every receipt matches a purchase order line", `SELECT count(*) FROM inventory_movements
			WHERE kind = 'receipt' AND purchase_order_line_id IS NULL`},
		{"transfers move one item between two branches of a city", `SELECT count(*) FROM (
			SELECT m.reference FROM inventory_movements m JOIN branches b ON b.id = m.branch_id
			WHERE m.kind IN ('transfer_out', 'transfer_in') GROUP BY m.reference
			HAVING count(*) <> 2 OR sum(m.quantity) <> 0 OR count(DISTINCT m.branch_id) <> 2
			OR count(DISTINCT m.item_id) <> 1 OR count(DISTINCT b.city) <> 1) x`},
		{"count adjustments are shrinkage", `SELECT count(*) FROM inventory_movements
			WHERE kind = 'adjustment' AND quantity >= 0`},
		{"every sale has a shift", `SELECT count(*) FROM sales WHERE shift_id IS NULL`},
		{"sale is inside its shift at the same branch", `SELECT count(*) FROM sales s
			JOIN register_shifts sh ON sh.id = s.shift_id
			JOIN cash_registers r ON r.id = sh.register_id
			WHERE s.sold_at < sh.opened_at OR s.sold_at >= sh.closed_at OR s.branch_id <> r.branch_id`},
		{"one register per branch", `SELECT count(*) FROM branches b
			WHERE (SELECT count(*) FROM cash_registers r WHERE r.branch_id = b.id) <> 1`},
		{"shift opener works at the register's branch", `SELECT count(*) FROM register_shifts sh
			JOIN cash_registers r ON r.id = sh.register_id
			JOIN employees e ON e.id = sh.opened_by
			WHERE e.branch_id <> r.branch_id`},
		{"expected cash equals the shift's cash flows", `SELECT count(*) FROM register_shifts sh
			JOIN cash_registers reg ON reg.id = sh.register_id
			WHERE sh.expected_cash_cents <> sh.opening_float_cents
			+ coalesce((SELECT sum(p.amount_cents) FROM sale_payments p
				JOIN sales s ON s.id = p.sale_id
				WHERE s.shift_id = sh.id AND s.status = 'completed' AND p.method = 'cash'), 0)
			- coalesce((SELECT sum((2 * ret.total_cents * cash.cash_cents + s.total_cents) / (2 * s.total_cents))
				FROM sale_returns ret
				JOIN sales s ON s.id = ret.sale_id
				JOIN LATERAL (SELECT coalesce(sum(p.amount_cents), 0) AS cash_cents
					FROM sale_payments p WHERE p.sale_id = s.id AND p.method = 'cash') cash ON true
				WHERE s.branch_id = reg.branch_id
				AND ret.returned_at >= sh.opened_at AND ret.returned_at < sh.closed_at
				AND cash.cash_cents > 0 AND s.total_cents > 0), 0)
			- coalesce((SELECT sum(m.amount_cents) FROM cash_movements m WHERE m.shift_id = sh.id), 0)`},
		{"counted cash is within 200 pesos of expected", `SELECT count(*) FROM register_shifts
			WHERE abs(counted_cash_cents - expected_cash_cents) > 20000`},
		{"most shifts count exact", `SELECT count(*) FROM (
			SELECT count(*) FILTER (WHERE counted_cash_cents = expected_cash_cents) AS exact, count(*) AS n
			FROM register_shifts) x WHERE exact * 2 <= n`},
		{"cash movement is inside its shift", `SELECT count(*) FROM cash_movements m
			JOIN register_shifts sh ON sh.id = m.shift_id
			WHERE m.moved_at < sh.opened_at OR m.moved_at >= sh.closed_at`},
		{"público en general is the generic receptor", `SELECT count(*) FROM customers
			WHERE id = 1 AND (rfc <> 'XAXX010101000' OR tax_regime <> '616')`},
		{"customer regime matches the rfc length", `SELECT count(*) FROM customers WHERE
			(tax_regime = '616' AND rfc <> 'XAXX010101000')
			OR (tax_regime = '612' AND length(rfc) <> 13)
			OR (tax_regime = '601' AND length(rfc) <> 12)`},
		{"invoice matches its completed sale", `SELECT count(*) FROM invoices i
			JOIN sales s ON s.id = i.sale_id
			WHERE s.status <> 'completed' OR i.issued_at <> s.sold_at OR i.payment_method <> 'PUE'`},
		{"cancelled sales have no invoice", `SELECT count(*) FROM sales s
			WHERE s.status = 'cancelled' AND EXISTS (SELECT 1 FROM invoices i WHERE i.sale_id = s.id)`},
		{"cfdi use follows the receptor regime", `SELECT count(*) FROM invoices i
			JOIN customers c ON c.id = i.customer_id
			WHERE (c.tax_regime = '616' AND i.cfdi_use <> 'S01')
			OR (c.tax_regime = '601' AND i.cfdi_use <> 'G01')
			OR (c.tax_regime = '612' AND i.cfdi_use <> 'G03')`},
		{"payment form follows the largest tender", `SELECT count(*) FROM invoices i
			JOIN LATERAL (
				SELECT method FROM sale_payments p WHERE p.sale_id = i.sale_id
				ORDER BY amount_cents DESC, id LIMIT 1
			) pay ON true
			WHERE i.payment_form <> CASE pay.method WHEN 'cash' THEN '01' WHEN 'transfer' THEN '03' ELSE '04' END`},
		{"invoices go to general and named receptors", `SELECT count(*) FROM (SELECT 1) x
			WHERE NOT EXISTS (SELECT 1 FROM invoices i JOIN customers c ON c.id = i.customer_id WHERE c.rfc = 'XAXX010101000')
			OR NOT EXISTS (SELECT 1 FROM invoices i JOIN customers c ON c.id = i.customer_id WHERE c.rfc <> 'XAXX010101000')`},
	})

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

	// Each edit breaks a unique key or a check so that one insert kind fails.
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
		{"payment", func(ds *Dataset) { ds.Sales[0].Payments[0].Method = "cheque" }},
		{"return", func(ds *Dataset) { ds.Returns[0].TotalCents++ }},
		{"return line", func(ds *Dataset) { ds.Returns[0].Lines = append(ds.Returns[0].Lines, ds.Returns[0].Lines[0]) }},
		{"supplier", func(ds *Dataset) { ds.Suppliers[1].Code = ds.Suppliers[0].Code }},
		{"purchase order", func(ds *Dataset) { ds.PurchaseOrders[1].Folio = ds.PurchaseOrders[0].Folio }},
		{"purchase order line", func(ds *Dataset) {
			ds.PurchaseOrders[0].Lines = append(ds.PurchaseOrders[0].Lines, ds.PurchaseOrders[0].Lines[0])
		}},
		{"opening movement", func(ds *Dataset) { ds.Movements[0].Quantity = -1 }},
		{"register", func(ds *Dataset) { ds.Registers[1].Code = ds.Registers[0].Code }},
		{"shift", func(ds *Dataset) { ds.Shifts[0].ClosedAt = ds.Shifts[0].OpenedAt }},
		{"cash movement", func(ds *Dataset) { ds.CashMovements[0].Kind = "theft" }},
		{"customer", func(ds *Dataset) { ds.Customers[1].Code = ds.Customers[0].Code }},
		{"invoice", func(ds *Dataset) { ds.Invoices[0].PaymentMethod = "PPD" }},
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

// TestWriteSalesPatterns checks the Phase 2 shape in SQL over two full
// months, enough data for the comparisons to be stable.
func TestWriteSalesPatterns(t *testing.T) {
	db := openMigratedDB(t)
	cfg := Default()
	cfg.Months = 2
	if err := Write(context.Background(), db, mustGenerate(t, cfg)); err != nil {
		t.Fatal(err)
	}
	zeroViolations(t, db, []struct{ name, query string }{
		{"Saturday revenue above Monday revenue", `SELECT count(*) FROM (
			SELECT sum(total_cents) FILTER (WHERE extract(isodow FROM sold_at AT TIME ZONE 'America/Mexico_City') = 6) AS sat,
			       sum(total_cents) FILTER (WHERE extract(isodow FROM sold_at AT TIME ZONE 'America/Mexico_City') = 1) AS mon
			FROM sales WHERE status = 'completed') x WHERE sat <= mon`},
		{"large tier daily revenue above small tier", `SELECT count(*) FROM (
			SELECT avg(revenue) FILTER (WHERE tier = 'large') AS large, avg(revenue) FILTER (WHERE tier = 'small') AS small
			FROM (SELECT b.tier, s.branch_id, (s.sold_at AT TIME ZONE b.timezone)::date, sum(s.total_cents) AS revenue
				FROM sales s JOIN branches b ON b.id = s.branch_id WHERE s.status = 'completed'
				GROUP BY 1, 2, 3) d) x WHERE large <= small`},
		{"some sales cancelled", `SELECT count(*) FROM (SELECT 1) x
			WHERE NOT EXISTS (SELECT 1 FROM sales WHERE status = 'cancelled')`},
		{"some payments split", `SELECT count(*) FROM (SELECT 1) x
			WHERE NOT EXISTS (SELECT 1 FROM sale_payments GROUP BY sale_id HAVING count(*) > 1)`},
		{"some employees left", `SELECT count(*) FROM (SELECT 1) x
			WHERE NOT EXISTS (SELECT 1 FROM employees WHERE terminated_on IS NOT NULL)`},
		{"some transfers and shrinkage", `SELECT count(*) FROM (SELECT 1) x
			WHERE NOT EXISTS (SELECT 1 FROM inventory_movements WHERE kind = 'transfer_in')
			OR NOT EXISTS (SELECT 1 FROM inventory_movements WHERE kind = 'adjustment')`}})
}
