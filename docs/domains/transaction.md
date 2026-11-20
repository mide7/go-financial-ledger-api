# Transaction domain

Package: [`internal/domain/transaction`](../../internal/domain/transaction)
Adapter: [`repository/transaction_repo.go`](../../internal/platform/database/postgres/repository/transaction_repo.go)

A **transaction** is one business event, such as a wallet top-up, a purchase or
a refund. It's recorded as two or more **entries** that move value between
accounts. Transactions and entries are **immutable**: a database trigger
rejects any `UPDATE` or `DELETE`. To undo one, post a new transaction that
links to it through `parent_transaction_id`. Reversal isn't implemented yet.

## Transaction

| Field | Notes |
|---|---|
| `id` | UUIDv7 |
| `reference` | Unique (≤ 128). The caller's idempotency key: retrying with the same reference returns `ErrDuplicateReference` and doesn't double-post |
| `type` | Free-form label, e.g. `WALLET_FUNDING`, `REFUND` (≤ 64) |
| `currency` | One currency per transaction; every entry's account must use it |
| `parent_transaction_id` | Optional link to a related transaction (e.g. the one being reversed) |
| `description` | Text |
| `metadata` | JSONB, defaults to `{}`; has a GIN index for querying |
| `transaction_date` | Business date supplied by the caller |
| `created_at` | When it was recorded |

## Entry

| Field | Notes |
|---|---|
| `account_id` | The account debited or credited |
| `type` | `DEBIT` or `CREDIT` (`domain.EntryType`) |
| `amount` | Minor units, `> 0` (DB `CHECK`) |
| `entry_sequence` | Global `BIGSERIAL`, the ordering used for snapshots and running balances |

The amount is always positive. Direction comes only from `type`, and the effect
on an account's balance depends on that account's
[normal balance](account.md#account-types).

## Posting rules

`CreateTransactionParams.Validate` ([params.go](../../internal/domain/transaction/params.go))
checks, in order:

1. At least two entries (`ErrAtLeastTwoEntries`).
2. Each entry has an account (`ErrInvalidAccountID`), a positive amount
   (`ErrInvalidEntryAmount`) and a valid type (`ErrInvalidEntryType`).
3. Debits and credits are totalled **separately**. If either side would exceed
   `math.MaxInt64` the result is `ErrAmountOverflow`. A single running net
   would wrap silently, which would let an unbalanced set of large entries pass.
4. Total debits must equal total credits (`ErrUnbalancedTransaction`).

The service then checks that the currency is active in the
[registry](currency.md#registry) (`ErrInvalidCurrency`), and passes the
transaction to the repository.

The repository, inside one database transaction:

1. inserts the transaction header (a unique violation on `reference` becomes
   `ErrDuplicateReference`);
2. bulk-inserts the entries with `COPY`;
3. commits.

The database then enforces the same invariants independently with the
statement-level trigger `trg_check_transaction_integrity`:

- every entry's account currency must equal the transaction currency;
- each transaction's entries must sum to zero.

A violation raises an exception and rolls the insert back. The API reports it
as a 500, because it means a bug got past the Go validation.

## Example: fund a wallet, then spend

| Transaction | Entry | Account | Amount |
|---|---|---|---|
| `fund-1` | DEBIT | `PLATFORM_CASH` (debit-normal) | 10000 |
| | CREDIT | `USER_WALLET` (credit-normal) | 10000 |
| `spend-1` | DEBIT | `USER_WALLET` | 2500 |
| | CREDIT | `PLATFORM_CASH` | 2500 |

Balances afterwards: wallet **7500**, cash **7500**. Both are positive in their
normal direction. This exact scenario is the integration test
[account_repo_integration_test.go](../../internal/platform/database/postgres/repository/account_repo_integration_test.go).

## Service

| Method | State |
|---|---|
| `CreateTransaction` | Implemented |
| `GetTransactionDetails` | Returns `domain.ErrNotImplemented` (501) |
| `ListTransactions` | Returns `domain.ErrNotImplemented` (501) |
| `ReverseTransaction` | Returns `domain.ErrNotImplemented` (501) |

The unimplemented methods need new SQL queries. A reversal should post a new
transaction that mirrors the original's entries with debit and credit swapped,
with `parent_transaction_id` pointing at the original.

## Errors

| Error | HTTP |
|---|---|
| `ErrDuplicateReference` | 422 |
| `ErrAtLeastTwoEntries`, `ErrUnbalancedTransaction`, `ErrInvalidEntryType`, `ErrInvalidEntryAmount`, `ErrInvalidAccountID`, `ErrAmountOverflow`, `ErrInvalidCurrency` | 422 |
| `ErrTransactionNotFound` | 404 (reserved) |
