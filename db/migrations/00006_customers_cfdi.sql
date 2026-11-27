-- +goose Up
-- Customers are CFDI receptors for the chain. The RFC is a fictional Mexican
-- tax id. Público en general is the SAT generic receptor XAXX010101000.
-- tax_regime is a c_RegimenFiscal code: 601 general de ley (persona moral),
-- 612 actividades empresariales (persona física), 616 sin obligaciones
-- fiscales (público en general).
CREATE TABLE customers (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code text NOT NULL UNIQUE,
    rfc text NOT NULL UNIQUE CHECK (rfc ~ '^[A-ZÑ&]{3,4}[0-9]{6}[A-Z0-9]{3}$'),
    legal_name text NOT NULL CHECK (legal_name <> ''),
    tax_regime text NOT NULL CHECK (tax_regime IN ('601', '612', '616')),
    postal_code text NOT NULL CHECK (postal_code ~ '^[0-9]{5}$')
);

-- One row per invoiced sale. This is metadata only: nothing is stamped and
-- nothing is sent to the SAT. payment_form is a c_FormaPago code (01 cash,
-- 03 transfer, 04 credit card). payment_method stays PUE because every
-- seeded sale is paid in full at the counter.
CREATE TABLE invoices (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sale_id bigint NOT NULL UNIQUE REFERENCES sales (id),
    customer_id bigint NOT NULL REFERENCES customers (id),
    uuid text NOT NULL UNIQUE CHECK (uuid ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    cfdi_use text NOT NULL CHECK (cfdi_use IN ('G01', 'G03', 'S01')),
    payment_form text NOT NULL CHECK (payment_form IN ('01', '03', '04')),
    payment_method text NOT NULL CHECK (payment_method = 'PUE'),
    issued_at timestamptz NOT NULL
);

CREATE INDEX invoices_customer_id_idx ON invoices (customer_id);

-- +goose Down
DROP TABLE invoices;
DROP TABLE customers;
