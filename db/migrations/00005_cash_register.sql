-- +goose Up
-- One register per branch. Shifts are one opening of that register.
CREATE TABLE cash_registers (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    branch_id bigint NOT NULL REFERENCES branches (id),
    code text NOT NULL UNIQUE,
    name text NOT NULL
);

CREATE INDEX cash_registers_branch_id_idx ON cash_registers (branch_id);

-- Expected cash is the drawer balance the count should match: the opening
-- float, plus cash taken in, minus cash refunds and cash moved out. It can
-- be negative when refunds exceed the drawer. Counted cash is that balance
-- plus the over or short.
CREATE TABLE register_shifts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    register_id bigint NOT NULL REFERENCES cash_registers (id),
    opened_by bigint NOT NULL REFERENCES employees (id),
    opened_at timestamptz NOT NULL,
    closed_at timestamptz NOT NULL CHECK (closed_at > opened_at),
    opening_float_cents bigint NOT NULL CHECK (opening_float_cents >= 0),
    expected_cash_cents bigint NOT NULL,
    counted_cash_cents bigint NOT NULL
);

CREATE INDEX register_shifts_register_id_opened_at_idx ON register_shifts (register_id, opened_at);

-- Withdrawals to the safe and petty expenses leave the drawer during a shift.
CREATE TABLE cash_movements (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    shift_id bigint NOT NULL REFERENCES register_shifts (id),
    moved_at timestamptz NOT NULL,
    kind text NOT NULL CHECK (kind IN ('withdrawal', 'expense')),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    note text NOT NULL
);

CREATE INDEX cash_movements_shift_id_idx ON cash_movements (shift_id);

-- The shift that was open when the sale was rung up. The seed sets it on
-- every sale.
ALTER TABLE sales ADD COLUMN shift_id bigint REFERENCES register_shifts (id);

CREATE INDEX sales_shift_id_idx ON sales (shift_id);

-- +goose Down
ALTER TABLE sales DROP COLUMN shift_id;
DROP TABLE cash_movements;
DROP TABLE register_shifts;
DROP TABLE cash_registers;
