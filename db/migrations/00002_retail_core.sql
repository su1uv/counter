-- +goose Up
CREATE TABLE branches (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code text NOT NULL UNIQUE,
    name text NOT NULL,
    city text NOT NULL,
    state text NOT NULL,
    timezone text NOT NULL,
    opened_on date NOT NULL
);

CREATE TABLE categories (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code text NOT NULL UNIQUE,
    name text NOT NULL
);

CREATE TABLE items (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sku text NOT NULL UNIQUE,
    category_id bigint NOT NULL REFERENCES categories (id),
    name text NOT NULL,
    material text NOT NULL,
    unit_cost_cents bigint NOT NULL CHECK (unit_cost_cents >= 0),
    list_price_cents bigint NOT NULL CHECK (list_price_cents >= 0),
    active boolean NOT NULL DEFAULT true
);

CREATE TABLE employees (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code text NOT NULL UNIQUE,
    branch_id bigint NOT NULL REFERENCES branches (id),
    full_name text NOT NULL,
    role text NOT NULL CHECK (role IN ('seller', 'manager')),
    hired_on date NOT NULL
);

CREATE TABLE sales (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    folio text NOT NULL UNIQUE,
    branch_id bigint NOT NULL REFERENCES branches (id),
    employee_id bigint NOT NULL REFERENCES employees (id),
    sold_at timestamptz NOT NULL,
    subtotal_cents bigint NOT NULL CHECK (subtotal_cents >= 0),
    tax_cents bigint NOT NULL CHECK (tax_cents >= 0),
    total_cents bigint NOT NULL CHECK (total_cents >= 0),
    CHECK (total_cents = subtotal_cents + tax_cents)
);

CREATE INDEX sales_branch_id_sold_at_idx ON sales (branch_id, sold_at);

CREATE TABLE sale_lines (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sale_id bigint NOT NULL REFERENCES sales (id),
    line_no integer NOT NULL CHECK (line_no > 0),
    item_id bigint NOT NULL REFERENCES items (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    unit_price_cents bigint NOT NULL CHECK (unit_price_cents >= 0),
    unit_cost_cents bigint NOT NULL CHECK (unit_cost_cents >= 0),
    discount_cents bigint NOT NULL CHECK (discount_cents >= 0),
    total_cents bigint NOT NULL CHECK (total_cents >= 0),
    UNIQUE (sale_id, line_no)
);

CREATE INDEX sale_lines_item_id_idx ON sale_lines (item_id);

-- +goose Down
DROP TABLE sale_lines;
DROP TABLE sales;
DROP TABLE employees;
DROP TABLE items;
DROP TABLE categories;
DROP TABLE branches;
