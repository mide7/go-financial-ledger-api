# Transaction domain

Package: [`internal/domain/transaction`](../../internal/domain/transaction)
Adapter: [`repository/transaction_repo.go`](../../internal/platform/database/postgres/repository/transaction_repo.go)

A **transaction** is one business event, such as a wallet top-up, a purchase or
a refund. It's recorded as two or more **entries** that move value between
accounts. Transactions and entries are **immutable**: a database trigger
rejects any `UPDATE` or `DELETE`. To undo one, post a new transaction that
links to it through `parent_transaction_id`; [reversal](#reversal) does this.

## Transaction

| Field | Notes |
|---|---|
| `id` | UUIDv7 |
| `reference` | Unique (≤ 128). The caller's idempotency key: retrying with the same reference returns `ErrDuplicateReference` and doesn't double-post |
| `type` | Free-form label, e.g. `WALLET_FUNDING`, `REFUND` (≤ 64) |
| `currency` | One currency per transaction; every entry's account must use it |
| `parent_transaction_id` | Optional link to a related transaction (e.g. the one being reversed) |
| `description` | Text |
| `metadata` | JSONB object (`json.RawMessage`); omitted or `null` becomes `{}`, any other non-object is `ErrInvalidMetadata`. Has a GIN index for querying |
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

1. share-locks the entry accounts (`LockAccountsForPosting`, `FOR SHARE` in
   id order) and checks each entry against its account's
   [status](account.md#status): `ErrEntryAccountNotFound` for an unknown
   account, `ErrAccountNotPostable` for an entry the status refuses;
2. inserts the transaction header (a unique violation on `reference` becomes
   `ErrDuplicateReference`);
3. bulk-inserts the entries with `COPY` and reads them back;
4. commits.

The database then enforces the same invariants independently with the
statement-level trigger `trg_check_transaction_integrity`:

- every entry's account currency must equal the transaction currency;
- each transaction's entries must sum to zero.

A violation raises an exception with a custom SQLSTATE and rolls the insert
back: `LD001` for a currency mismatch, `LD002` for an unbalanced transaction.
The repository maps them (`mapIntegrityError`) to `ErrCurrencyMismatch` and
`ErrUnbalancedTransaction`, both 422. The service doesn't check entry account
currencies itself, so the trigger is what reports a currency mismatch.

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
| `CreateTransaction` | Implemented. Rejects `type` `REVERSAL` with `ErrReservedType`. The repository enforces [account status](account.md#status) |
| `GetTransactionDetails(id uuid.UUID)` | Implemented. Returns the header and its entries in `entry_sequence` order; `ErrTransactionNotFound` for an unknown id |
| `ListTransactions(ListTransactionsParams)` | Implemented. Headers only, newest first. Filters: account (any entry on it), reference, type, currency, half-open `created_at` range (`ErrInvalidDateRange` unless start < end when both are set); keyset cursor on `id`. The limit is clamped to 1–100, default 20 (`DefaultPageLimit`, `MaxPageLimit`). Returns a `CursorPaginatedResult` |
| `ReverseTransaction(ReverseTransactionParams)` | Implemented; see [Reversal](#reversal) |
| `CreateConversion(CreateConversionParams)` | Scaffolded, 501 ([later-fx](../todo.md#later-scaffolded-routes)). Will post one transaction per currency through `PLATFORM_FX_SUSPENSE`, returning a `Conversion` |

The repository port matches: `GetByID(uuid.UUID)` returns the transaction with
its entries, `List` returns headers, and `Reverse(ReverseTransactionParams)`
returns the reversal. `Create` and `Reverse` share one insert helper, which
reads the posted entries back inside its
database transaction, so the created transaction carries their ids and
sequences.

## Reversal

`Reverse` runs in one database transaction:

1. Load the original; an unknown id is `ErrTransactionNotFound`.
2. Reject a `REVERSAL` with `ErrCannotReverseReversal`.
3. Reject an original that already has a reversal (`GetReversalOf`) with `ErrAlreadyReversed`.
4. Insert a `TypeReversal` header: the caller's `reference`, the original's
   currency, `parent_transaction_id` = the original, `description` = the
   reason, `metadata` `{}`, `transaction_date` now.
5. Lock the accounts and check them with `Status.AllowsReversal`: only a
   `CLOSED` account refuses (`ErrAccountNotPostable`).
6. Insert the original's entries with each `type` flipped
   (`domain.EntryType.Opposite`), same accounts and amounts.

Two concurrent reversals can both pass step 3. The partial unique index
`uq_transactions_one_reversal` (one `REVERSAL` per `parent_transaction_id`)
rejects the second insert, and the repository maps that violation, by index
name, to the same `ErrAlreadyReversed`. Any other unique violation is
`ErrDuplicateReference`.

Because `CreateTransaction` rejects `type` `REVERSAL`, a client can't post a
fake reversal that would block the real one.

## Errors

| Error | HTTP |
|---|---|
| `ErrDuplicateReference` | 422 |
| `ErrAtLeastTwoEntries`, `ErrUnbalancedTransaction`, `ErrInvalidEntryType`, `ErrInvalidEntryAmount`, `ErrInvalidAccountID`, `ErrAmountOverflow`, `ErrInvalidCurrency` | 422 |
| `ErrTransactionNotFound` | 404 |
| `ErrInvalidDateRange`, `ErrReservedType` | 422 |
| `ErrAlreadyReversed` | 409 |
| `ErrCannotReverseReversal` | 422 |
| `ErrAccountNotPostable`, `ErrEntryAccountNotFound` | 422 |
| `ErrCurrencyMismatch`, `ErrInvalidMetadata` | 422 |
