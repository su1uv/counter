-- name: Ping :one
SELECT 1::int AS ok;

-- name: CountSchemaProbe :one
SELECT count(*) FROM schema_probe;
