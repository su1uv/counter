-- +goose Up
CREATE TABLE schema_probe (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY
);

-- +goose Down
DROP TABLE schema_probe;
