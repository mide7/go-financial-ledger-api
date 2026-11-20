# Getting started

## Prerequisites

The project manages its tools with [devbox](https://www.jetify.com/devbox)
(`devbox.json`): Go 1.27.1, golangci-lint, air (live reload), gopls and sqlc 1.27.0.

```sh
devbox shell          # enter an environment with all tools on PATH
```

Without devbox, install those tools yourself. You also need Docker for the
database. Postgres **18** is required, because the schema uses `uuidv7()`.

> `devbox.json` and `devbox.lock` are currently excluded by `.gitignore`
> (`*devbox*`), so a fresh clone won't have them. See [Known issues](known-issues.md).

## Configuration

Settings come from environment variables. On startup, `config.Load` reads a
`.env` file from the working directory if one exists. Variables already set in
the environment take precedence. Copy the example to start:

```sh
cp .env.example .env
```

| Variable | Required | Default | Used for |
|---|---|---|---|
| `DATABASE_URL` | **yes** | — | Postgres connection string. Startup pings it and fails if it's empty or unreachable. |
| `APP_ENV` | no | `development` | `development` or `production`; picks the logging defaults below |
| `LOG_LEVEL` | no | `debug` (dev) / `info` (prod) | `debug`, `info`, `warn` or `error` |
| `LOG_FORMAT` | no | `text` (dev) / `json` (prod) | Default format for outputs without an `@format` suffix |
| `LOG_OUTPUTS` | no | `stderr` (dev) / `stdout` (prod) | Where logs go; see [Logging](#logging) |
| `PORT` | no | `:8080` | HTTP listen address; a bare number like `8080` also works |
| `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB` | no | `localhost`, `6379`, empty, `0` | Reserved; Redis is not used yet. `REDIS_DB` must be an integer. |
| `ADMIN_NAME`, `ADMIN_EMAIL`, `ADMIN_PASSWORD` | no | empty | Reserved; not used yet |
| `FRONTEND_URL` | no | empty | Reserved; not used yet |

For the development compose database:

```
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/financial-ledger-api?sslmode=disable
```

Inside the compose network, use host `postgres` instead of `localhost`.

## Logging

All commands log through one configured `*slog.Logger` (see
[Architecture → Conventions](architecture.md#conventions)). `LOG_OUTPUTS` is a
comma-separated list, and **every output receives every record**:

| Output | Meaning |
|---|---|
| `stdout` | Standard output |
| `stderr` | Standard error |
| `file:<path>` | Append to a file, creating it if needed. It is not rotated, so use logrotate or the platform's log handling. |

Any output can end in `@text` or `@json` to override `LOG_FORMAT`.

```sh
# Development (the default): readable text on stderr, debug level, with source file:line
APP_ENV=development

# Production: JSON to stdout for the platform, a JSON file, and readable text on stderr
APP_ENV=production
LOG_OUTPUTS=stdout@json,file:/var/log/ledger/app.log@json,stderr@text
```

Source locations are included only in development. Every record has a
`service` attribute (`ledger-api`, `migrate` or `seed`). Each request produces one
`http request` record: 5xx logs at error, 4xx at warn, everything else at info.
Unexpected 500 errors are also logged with their cause as `request failed`. An
invalid `APP_ENV` or logging value stops startup with a clear error.

## Run it

### Everything in Docker (recommended)

```sh
make docker-compose-up     # postgres → migrate up → seed → app (live reload via air)
curl localhost:8080/health/
make docker-compose-down   # stop and DELETE the database volume (-v)
```

The `migration` and `seeder` services run once, in order, before `app` starts.

### App on the host, database in Docker

```sh
docker compose -f docker/development/docker-compose.yaml up -d postgres
make migrate-up            # must run from the repository root
make seed
make run                   # builds bin/ledger-service and starts it
```

## Make targets

Run `make help` for the full list.

| Target | Does |
|---|---|
| `build` / `run` | Build `bin/ledger-service` / build and run it |
| `test` | All unit tests with `-race` and coverage |
| `test-short` | Unit tests with `-short` |
| `test-integration` | Database integration tests; needs `TEST_DATABASE_URL` (see [Testing](testing.md)) |
| `lint` / `fmt` / `tidy` | golangci-lint / `go fmt` + `go vet` / `go mod tidy` + verify |
| `migration <name>` | Create a new up/down migration pair, e.g. `make migration add_withdrawals` |
| `migrate-up` / `migrate-down` | Apply all / revert **all** migrations |
| `sqlc` | Regenerate `internal/platform/database/postgres/sqlc` from `query/*.sql` |
| `seed` | Insert reference data (safe to re-run) |
| `docker-build`, `docker-run` | Production image (multi-stage build, Alpine runtime, non-root user; contains only the server binary) |
| `docker-build-dev`, `docker-run-dev` | Development image with air |
| `docker-compose-up` / `-down` | Development stack |
| `docker-compose-prod-up` / `-down` | Production compose stack |

## Seed data

`make seed` (or the compose `seeder` service) runs
[seeder.go](../internal/platform/database/postgres/seeder.go) in a single
transaction. Every insert is `ON CONFLICT DO NOTHING`, so re-running is safe.

- **Currencies:** USD, EUR and NGN, each with exponent 2 and active.
- **Account types:** the 10 types listed in [Account domain](domains/account.md#account-types).
- **System accounts:** one NGN account per account type, owned by `admin`.

## Migrations

Migrations live in `internal/platform/database/postgres/migrations` and are run
by [cmd/migrate](../cmd/migrate/main.go) with golang-migrate:

```sh
go run ./cmd/migrate up      # or: make migrate-up
go run ./cmd/migrate down    # reverts every migration
```

The migrations path is relative, so run the command from the repository root.
While the project is in development, schema changes are made by **editing the
existing migration files** rather than adding `ALTER` migrations. After editing,
recreate the dev database with `make docker-compose-down && make docker-compose-up`.
See [Database](database.md#migrations-and-sqlc).
