-- +goose Up
CREATE TABLE suppliers (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code text NOT NULL UNIQUE,
    name text NOT NULL,
    city text NOT NULL,
    lead_time_days integer NOT NULL CHECK (lead_time_days > 0)
);

-- Suppliers deliver straight to branches. An order is received in full or
-- still open.
CREATE TABLE purchase_orders (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    folio text NOT NULL UNIQUE,
    supplier_id bigint NOT NULL REFERENCES suppliers (id),
    branch_id bigint NOT NULL REFERENCES branches (id),
    ordered_at timestamptz NOT NULL,
    received_at timestamptz CHECK (received_at > ordered_at),
    status text NOT NULL CHECK (status IN ('ordered', 'received')),
    CHECK ((status = 'received') = (received_at IS NOT NULL))
);

CREATE INDEX purchase_orders_branch_id_ordered_at_idx ON purchase_orders (branch_id, ordered_at);

CREATE TABLE purchase_order_lines (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    purchase_order_id bigint NOT NULL REFERENCES purchase_orders (id),
    line_no integer NOT NULL CHECK (line_no > 0),
    item_id bigint NOT NULL REFERENCES items (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    unit_cost_cents bigint NOT NULL CHECK (unit_cost_cents >= 0),
    UNIQUE (purchase_order_id, line_no)
);

CREATE INDEX purchase_order_lines_item_id_idx ON purchase_order_lines (item_id);

-- Stock is the sum of this ledger. Quantity is signed: sales and transfers
-- out take stock, adjustments go either way, every other kind adds. Sale,
-- return and receipt rows point to exactly one source line; reference is
-- the business code of the source document.
CREATE TABLE inventory_movements (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    branch_id bigint NOT NULL REFERENCES branches (id),
    item_id bigint NOT NULL REFERENCES items (id),
    moved_at timestamptz NOT NULL,
    kind text NOT NULL CHECK (kind IN (
        'opening', 'receipt', 'sale', 'return', 'transfer_in', 'transfer_out', 'adjustment')),
    quantity integer NOT NULL,
    unit_cost_cents bigint NOT NULL CHECK (unit_cost_cents >= 0),
    reference text NOT NULL,
    sale_line_id bigint UNIQUE REFERENCES sale_lines (id),
    sale_return_line_id bigint UNIQUE REFERENCES sale_return_lines (id),
    purchase_order_line_id bigint UNIQUE REFERENCES purchase_order_lines (id),
    CHECK (CASE
        WHEN kind IN ('sale', 'transfer_out') THEN quantity < 0
        WHEN kind = 'adjustment' THEN quantity <> 0
        ELSE quantity > 0
    END),
    CHECK ((kind = 'sale') = (sale_line_id IS NOT NULL)),
    CHECK ((kind = 'return') = (sale_return_line_id IS NOT NULL)),
    CHECK ((kind = 'receipt') = (purchase_order_line_id IS NOT NULL))
);

CREATE INDEX inventory_movements_branch_id_item_id_moved_at_idx
    ON inventory_movements (branch_id, item_id, moved_at);

CREATE VIEW stock_levels AS
SELECT branch_id, item_id, sum(quantity)::bigint AS quantity
FROM inventory_movements
GROUP BY branch_id, item_id;

-- +goose Down
DROP VIEW stock_levels;
DROP TABLE inventory_movements;
DROP TABLE purchase_order_lines;
DROP TABLE purchase_orders;
DROP TABLE suppliers;
