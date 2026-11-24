# Testing

```sh
make test                 # unit tests, -race, coverage
go test -race -shuffle=on ./...
make lint                 # also lints integration-tagged files
```

## Layout

Each source file has a matching test file (`foo.go` → `foo_test.go`), and test
functions follow the order of the code they test. Tests are table-driven with
named cases and call `t.Parallel()`. The exceptions are tests that use
`t.Setenv` or `t.Chdir`.

| Kind | Where | Needs |
|---|---|---|
| Domain unit tests | `internal/domain/**/_test.go` | Nothing. Services use fake repositories that embed the port interface, so an unexpected call panics. |
| HTTP tests | `transport/http/router_test.go`, `httputil/`, `middleware/`, `handler/`, `dto/`, `validation/` | Nothing. They use `httptest`, the real router and middleware, and fake services. |
| Repository unit tests | `repository/errors_test.go`, `currency_registry_test.go` | Nothing |
| Integration tests | `*_integration_test.go`, build tag `integration` | PostgreSQL |
| Fuzz tests | `FuzzCreateTransactionParams_Validate`, `FuzzFormatMinor` | Nothing |

## Integration tests

They're excluded from `go test ./...` by the `//go:build integration` tag, and
they skip when `TEST_DATABASE_URL` is unset.

```sh
make docker-compose-up
TEST_DATABASE_URL='postgresql://postgres:postgres@localhost:5432/financial-ledger-api?sslmode=disable' \
  make test-integration
```

| Test | Checks | Database requirements |
|---|---|---|
| `TestBalanceSignConvention` ([account_repo_integration_test.go](../internal/platform/database/postgres/repository/account_repo_integration_test.go)) | Balance, snapshot, reconcile and statements agree on the sign rule for a credit-normal and a debit-normal account | Migrated and seeded. Leaves rows behind (entries are immutable) under a fresh `it-<uuid>` owner. |
| `TestRun_UpDownUp` ([main_integration_test.go](../cmd/migrate/main_integration_test.go)) | Every migration applies, fully reverts, and re-applies | User may `CREATE DATABASE`. Uses a throwaway database that it drops afterwards. |

## Fuzzing

```sh
go test ./internal/domain/transaction -run '^$' -fuzz FuzzCreateTransactionParams_Validate -fuzztime 30s
go test ./internal/domain/currency    -run '^$' -fuzz FuzzFormatMinor -fuzztime 30s
```

The first compares `Validate` with exact `big.Int` arithmetic. This found the
int64 overflow that let an unbalanced transaction pass. The second checks that
`FormatMinor` output parses back to the original value. The seed corpus runs as
part of ordinary `go test`.

## Writing tests

- Fake the domain ports (`account.Repository`, `currency.Registry`,
  `account.Service`, …), not concrete types. Embed the interface in the fake and
  override only the methods the test uses.
- Assert observable behavior: returned values, errors with `errors.Is`, HTTP
  status codes and bodies, and the parameters sent to a port when that's the
  contract, such as pagination defaults.
- Use `httptest.NewRequestWithContext(t.Context(), …)`; the linter rejects
  `NewRequest`.
- Anything that needs a real database goes behind the `integration` tag.
