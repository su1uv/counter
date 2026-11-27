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
INSERT INTO sales (folio, branch_id, employee_id, sold_at, status, subtotal_cents, tax_cents, total_cents)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
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
    (SELECT count(*) FROM sale_return_lines) AS sale_return_lines;

-- name: ResetSeedData :exec
TRUNCATE sale_return_lines, sale_returns, sale_payments,
    sale_lines, sales, employees, items, categories, branches RESTART IDENTITY;
