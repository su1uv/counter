-- name: InsertBranch :one
INSERT INTO branches (code, name, city, state, timezone, opened_on, tier)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: InsertCategory :one
INSERT INTO categories (code, name)
VALUES ($1, $2)
RETURNING id;

-- name: InsertItem :one
INSERT INTO items (sku, category_id, name, material, unit_cost_cents, list_price_cents, active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: InsertEmployee :one
INSERT INTO employees (code, branch_id, full_name, role, hired_on, terminated_on)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: InsertSale :one
INSERT INTO sales (folio, branch_id, employee_id, sold_at, status, subtotal_cents, tax_cents, total_cents, shift_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- name: InsertSaleLine :one
INSERT INTO sale_lines (
    sale_id, line_no, item_id, quantity,
    unit_price_cents, unit_cost_cents, discount_cents, total_cents
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: InsertSalePayment :one
INSERT INTO sale_payments (sale_id, method, amount_cents, card_brand, last4)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: InsertSaleReturn :one
INSERT INTO sale_returns (folio, sale_id, employee_id, returned_at, subtotal_cents, tax_cents, total_cents)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: InsertSaleReturnLine :one
INSERT INTO sale_return_lines (return_id, line_no, sale_line_id, quantity, refund_cents)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: InsertSupplier :one
INSERT INTO suppliers (code, name, city, lead_time_days)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: InsertPurchaseOrder :one
INSERT INTO purchase_orders (folio, supplier_id, branch_id, ordered_at, received_at, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: InsertPurchaseOrderLine :one
INSERT INTO purchase_order_lines (purchase_order_id, line_no, item_id, quantity, unit_cost_cents)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: InsertCashRegister :one
INSERT INTO cash_registers (branch_id, code, name)
VALUES ($1, $2, $3)
RETURNING id;

-- name: InsertRegisterShift :one
INSERT INTO register_shifts (
    register_id, opened_by, opened_at, closed_at,
    opening_float_cents, expected_cash_cents, counted_cash_cents
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: InsertCashMovement :one
INSERT INTO cash_movements (shift_id, moved_at, kind, amount_cents, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: InsertInventoryMovement :one
INSERT INTO inventory_movements (
    branch_id, item_id, moved_at, kind, quantity, unit_cost_cents, reference,
    sale_line_id, sale_return_line_id, purchase_order_line_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: GetStockLevel :one
SELECT quantity FROM stock_levels
WHERE branch_id = $1 AND item_id = $2;

-- name: CountSeedRows :one
SELECT
    (SELECT count(*) FROM branches) AS branches,
    (SELECT count(*) FROM categories) AS categories,
    (SELECT count(*) FROM items) AS items,
    (SELECT count(*) FROM employees) AS employees,
    (SELECT count(*) FROM sales) AS sales,
    (SELECT count(*) FROM sale_lines) AS sale_lines,
    (SELECT count(*) FROM sale_payments) AS sale_payments,
    (SELECT count(*) FROM sale_returns) AS sale_returns,
    (SELECT count(*) FROM sale_return_lines) AS sale_return_lines,
    (SELECT count(*) FROM suppliers) AS suppliers,
    (SELECT count(*) FROM purchase_orders) AS purchase_orders,
    (SELECT count(*) FROM purchase_order_lines) AS purchase_order_lines,
    (SELECT count(*) FROM inventory_movements) AS inventory_movements,
    (SELECT count(*) FROM cash_registers) AS cash_registers,
    (SELECT count(*) FROM register_shifts) AS register_shifts,
    (SELECT count(*) FROM cash_movements) AS cash_movements;

-- name: ResetSeedData :exec
TRUNCATE cash_movements, register_shifts, cash_registers,
    inventory_movements, purchase_order_lines, purchase_orders, suppliers,
    sale_return_lines, sale_returns, sale_payments,
    sale_lines, sales, employees, items, categories, branches RESTART IDENTITY;
