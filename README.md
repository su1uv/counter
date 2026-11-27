# counter

Counter is a Go service. A store manager asks a question in Spanish or English, and an agent answers by calling typed SQL tools over sales, inventory and cash-register data, and by retrieving passages from operating manuals and SAT CFDI guides, with citations. This repository currently holds only the foundation: config, HTTP server, Postgres, migrations and typed queries.

## Prerequisites

- Go 1.27.4
- Docker with Compose (local Postgres 18)
- goose v3.27.3 CLI (only to create migration files)
- sqlc v1.31.1 CLI (only to regenerate `internal/store`)

## Environment variables

| Variable | Used by | Notes |
| --- | --- | --- |
| `DATABASE_URL` | service, `migrate`, `seed` | required |
| `ADDR` | service | default `:8080` |
| `TEST_DATABASE_URL` | integration tests | tests skip when unset |

See `.env.example`. There is no `.env` loader: `export` the variables. If port 5432 is taken, start Compose with `POSTGRES_PORT=5433` and adjust the URLs.

## Quickstart

```bash
export DATABASE_URL='postgres://counter:counter@localhost:5432/counter?sslmode=disable'
make db-up
make migrate-up
make run
curl localhost:8080/healthz   # liveness, no database
curl localhost:8080/readyz    # readiness, pings the database
```

## Seed data

`make seed` fills the database with synthetic data for "Joyería Aurora", a fictional jewelry chain: 15 branches, 8 categories, 300 items, employees, sales with lines, payments and returns, and the inventory behind them (suppliers, purchase orders and a stock ledger) from 2026-05-01 to 2026-10-31. Money is MXN in integer centavos, and shelf prices include 16% IVA. Data values are in Spanish.

```bash
make migrate-up                  # seed never migrates
make seed                        # refuses if the seed tables already have rows
make seed SEED_FLAGS=-reset      # truncates the seed tables, then seeds
make seed SEED=7 SEED_FLAGS=-reset
```

Seeding is deterministic: the same `SEED` (default 1), code, and Go version produce the same rows and the same ids. The window is fixed, not relative to today. With `SEED=1` a full run inserts 78 employees, 82,013 sales, 112,878 sale lines, 85,237 payments, 2,023 returns, 9,491 purchase orders and 160,017 inventory movements in about 27 seconds on a laptop against the Compose Postgres. The `seed` command is separate from the service; the service binary does not import it.

The sales follow patterns an agent can find in SQL:

- **Calendar.** Saturday is the busiest day and Monday the quietest. Stores open 11:00–21:00, busiest around 18:00. Paydays (the 15th and the last day of the month) sell 30% more, and Mother's Day (May 10) is the busiest day of the window.
- **Branch tiers.** `branches.tier` is `large` (3 branches, including Centro), `medium` (7) or `small` (5). Large branches have more traffic, more staff, and sell more gold and watches, so their average ticket is higher.
- **Staff.** Each branch has one manager and 2–5 sellers. Some sellers sell more than others, each has a weekly day off, and a few leave during the window (`employees.terminated_on`, the last day worked) and are replaced.
- **Baskets.** Most sales have one line and one unit. About 10% of lines get 5–20% off, recorded in `sale_lines.discount_cents`.
- **Payments.** `sale_payments` is about 60% card, 30% cash and 10% bank transfer. About 5% of sales are split across two methods. Payments of a completed sale add up to its total.
- **Cancellations and returns.** About 1% of sales have `sales.status = 'cancelled'`; they keep their lines but have no payments. About 2% of lines come back within 30 days in `sale_returns` and `sale_return_lines`, which point to the original sale line. Filter on `status = 'completed'` for revenue.

Inventory follows the sales. The simulation runs day by day and branch by branch:

- **Ledger.** `inventory_movements` is the only record of stock. Each row has a signed `quantity`, the item's `unit_cost_cents`, and a `reference` to the source document (sale or return folio, purchase order, transfer or count). `kind` is `opening`, `receipt`, `sale`, `return`, `transfer_in`, `transfer_out` or `adjustment`. Sale, return and receipt rows point to their source line through `sale_line_id`, `sale_return_line_id` or `purchase_order_line_id`. The `stock_levels` view sums the ledger per branch and item.
- **Stock policy.** Each branch stocks every item. Opening stock covers about 30 days of that branch's expected demand, with at least 2 units. The reorder point is about 12 days of demand, with at least 1 unit.
- **Sales.** Every completed sale line takes stock at `sold_at`, and returned units go back on the shelf. Cancelled sales take no stock. When an item is short, the line is swapped for another in-stock item of the same category. If none is left, the line is lost, and a sale with no lines left never happens. Both are rare. Stock never goes negative.
- **Purchasing.** Suppliers deliver directly to branches: `PR01` silver, `PR02` gold 10k and 14k, `PR03` gold 18k, and `PR04` watches. After closing, each branch orders every item at or below its reorder point, up to its maximum, with one `purchase_orders` row per supplier. Orders arrive in full at 10:00, after the supplier's `lead_time_days` plus 0–2 days. Orders that would arrive after the window stay `ordered`.
- **Transfers.** When a branch sells out of an item, a branch in the same city with spare stock above its own reorder point sends some that night. The units arrive the next morning. Both rows share a `TR-` reference.
- **Counts.** On the last day of each month, each branch counts its stock. A few items come up one unit short and are recorded as `adjustment` rows of −1 with a `CONTEO-` reference. That is shrinkage, about 140 units chain-wide.

## Deploy order

The service never migrates and never embeds migration SQL. Run the separate `migrate` binary once per deploy against the `db/migrations/` files on disk, and start service instances only after it succeeds:

```bash
make build
DATABASE_URL=... bin/migrate -dir db/migrations up   # non-zero exit aborts the deploy
DATABASE_URL=... bin/counter                         # then start N instances
```

## Tests

```bash
make test               # unit tests, no database
make test-integration   # needs `make db-up`; uses the counter_test database
make cover              # coverage profile
```

## Generated code

`internal/store` is generated by sqlc from `db/migrations` and `db/queries`. Never edit it by hand: change the SQL, run `make sqlc`, and commit the result. `make sqlc-check` fails if it is stale. New migrations: `make migrate-new name=<name>`.
