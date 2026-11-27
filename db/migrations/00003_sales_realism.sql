-- +goose Up
ALTER TABLE branches ADD COLUMN tier text NOT NULL DEFAULT 'medium'
    CHECK (tier IN ('large', 'medium', 'small'));

ALTER TABLE employees ADD COLUMN terminated_on date
    CHECK (terminated_on >= hired_on);

ALTER TABLE sales ADD COLUMN status text NOT NULL DEFAULT 'completed'
    CHECK (status IN ('completed', 'cancelled'));

CREATE TABLE sale_payments (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sale_id bigint NOT NULL REFERENCES sales (id),
    method text NOT NULL CHECK (method IN ('card', 'cash', 'transfer')),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    card_brand text,
    last4 text CHECK (last4 ~ '^[0-9]{4}$'),
    CHECK (
        (method = 'card' AND card_brand IS NOT NULL AND last4 IS NOT NULL)
        OR (method <> 'card' AND card_brand IS NULL AND last4 IS NULL)
    )
);

CREATE INDEX sale_payments_sale_id_idx ON sale_payments (sale_id);

CREATE TABLE sale_returns (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    folio text NOT NULL UNIQUE,
    sale_id bigint NOT NULL REFERENCES sales (id),
    employee_id bigint NOT NULL REFERENCES employees (id),
    returned_at timestamptz NOT NULL,
    subtotal_cents bigint NOT NULL CHECK (subtotal_cents >= 0),
    tax_cents bigint NOT NULL CHECK (tax_cents >= 0),
    total_cents bigint NOT NULL CHECK (total_cents >= 0),
    CHECK (total_cents = subtotal_cents + tax_cents)
);

CREATE INDEX sale_returns_sale_id_idx ON sale_returns (sale_id);

CREATE TABLE sale_return_lines (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    return_id bigint NOT NULL REFERENCES sale_returns (id),
    line_no integer NOT NULL CHECK (line_no > 0),
    sale_line_id bigint NOT NULL REFERENCES sale_lines (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    refund_cents bigint NOT NULL CHECK (refund_cents >= 0),
    UNIQUE (return_id, line_no)
);

CREATE INDEX sale_return_lines_sale_line_id_idx ON sale_return_lines (sale_line_id);

-- +goose Down
DROP TABLE sale_return_lines;
DROP TABLE sale_returns;
DROP TABLE sale_payments;
ALTER TABLE sales DROP COLUMN status;
ALTER TABLE employees DROP COLUMN terminated_on;
ALTER TABLE branches DROP COLUMN tier;
