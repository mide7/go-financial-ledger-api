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

**Scaffolded routes:** `TestScaffoldedRoutes_NotImplemented` in
[router_test.go](../internal/platform/transport/http/router_test.go) sends a
valid request to every route whose service is still a stub, through the real
stub services, and expects 501. It proves each route is registered and its
validation accepts a correct request. Remove a route's case when its
[todo.md](todo.md) item is built.

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
| `TestTransactionReadBack` ([transaction_repo_integration_test.go](../internal/platform/database/postgres/repository/transaction_repo_integration_test.go)) | `Create` returns persisted entries; `GetByID` returns entries in sequence order and `ErrTransactionNotFound` for an unknown id; `List` pages newest first by account and honours the reference, currency and half-open date filters | Migrated and seeded. Leaves rows behind under a fresh `it-<uuid>` owner. |
| `TestTransactionReversal`, `TestTransactionReversal_Concurrent` (same file) | A reversal mirrors the entries and zeroes both balances; a second reversal, reversing a reversal, an unknown id and a reused reference are rejected; of two racing reversals exactly one succeeds | Migrated and seeded. Leaves rows behind under a fresh `it-<uuid>` owner. |
| `TestPostingRespectsAccountStatus` (same file) | For each non-active wallet status: credit, debit and reversal are allowed or refused per the status rules; an unknown entry account is `ErrEntryAccountNotFound` | Migrated and seeded. Leaves rows behind. |
| `TestChangeAccountStatus_Transitions` ([account_repo_integration_test.go](../internal/platform/database/postgres/repository/account_repo_integration_test.go)) | A change to the current status is `ErrStatusUnchanged`; nothing leaves `CLOSED` (`ErrAccountClosed`); neither writes history | Migrated and seeded. |
| `TestListAccounts_OwnerFilter` ([account_repo_integration_test.go](../internal/platform/database/postgres/repository/account_repo_integration_test.go)) | Filtering by `owner_id` returns exactly that owner's accounts, alone and with `type`; an unknown owner returns none | Migrated and seeded. Leaves rows behind under fresh `it-<uuid>` owners. |
| `TestIdempotencyStore_*` ([idempotency_store_integration_test.go](../internal/platform/database/postgres/repository/idempotency_store_integration_test.go)) | Claim, in-progress, complete and replay; FAILED keys reclaimable only by the same request, expired keys by any; `DeleteExpired`; exactly one of 8 concurrent claims wins | Migrated. Leaves `it-<uuid>` keys behind until they expire. |
| `TestIntegrityTriggerErrors`, `TestStatementSummary_EndIsExclusive` ([transaction_repo_integration_test.go](../internal/platform/database/postgres/repository/transaction_repo_integration_test.go)) | The trigger's `LD001`/`LD002` map to `ErrCurrencyMismatch`/`ErrUnbalancedTransaction`; the statement summary excludes an entry at exactly `end_date` | Migrated and seeded (needs active USD). Leaves rows behind. |
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
