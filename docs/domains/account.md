# Account domain

Package: [`internal/domain/account`](../../internal/domain/account)
Adapter: [`repository/account_repo.go`](../../internal/platform/database/postgres/repository/account_repo.go)

An **account** is a single-currency bucket of value owned by someone: a user,
a merchant or the platform. Its balance is never stored; it is derived from the
[entries](transaction.md#entry) posted to it.

## Account

| Field | Type | Notes |
|---|---|---|
| `id` | UUIDv7 | |
| `owner_id` | string ≤ 64 | Opaque ID from the owning system; system accounts use `admin` |
| `type` | `Type` | A row in `account_types`, see below |
| `currency` | `currency.Code` | Fixed for the life of the account |
| `created_at`, `updated_at` | time | |

An owner can hold **one account per (type, currency)**, enforced by the
unique constraint `uq_owner_type_currency`. A second create returns
`ErrAccountExists` (409).

The database also stores a `status` column (see [Status](#status)); it isn't
in the `Account` struct yet.

## Account types

The chart of accounts lives in the `account_types` table, is seeded at setup,
and mirrors the `Type*` constants and `AllTypes` in
[model.go](../../internal/domain/account/model.go).

| Type | Nature | Normal balance | Purpose |
|---|---|---|---|
| `USER_WALLET` | LIABILITY | CREDIT | User's spendable balance per currency |
| `USER_LOCKED` | LIABILITY | CREDIT | User funds held pending withdrawal or dispute |
| `MERCHANT_WALLET` | LIABILITY | CREDIT | Merchant's available balance |
| `MERCHANT_PAYABLE` | LIABILITY | CREDIT | Settled merchant funds queued for payout |
| `MERCHANT_RESERVE` | LIABILITY | CREDIT | Rolling reserve held against chargebacks/refunds |
| `PLATFORM_ESCROW_LIABILITY` | LIABILITY | CREDIT | Funds held mid-transaction before release |
| `PLATFORM_FEE_REVENUE` | REVENUE | CREDIT | Fees the platform has earned |
| `PLATFORM_CASH` | ASSET | DEBIT | Real money at the bank or payment processor |
| `PLATFORM_FX_SUSPENSE` | LIABILITY | CREDIT | Clearing account between the two legs of a conversion |
| `PLATFORM_FX_GAIN_LOSS` | REVENUE | CREDIT | Realised FX spread or slippage |

**Normal balance** is the side that increases the account. It decides the sign
of every balance the API reports (see
[sign convention](../database.md#balance-sign-convention)). For example,
funding a wallet with ₦100 is *debit `PLATFORM_CASH` 10000 / credit
`USER_WALLET` 10000*. Both balances then read **+10000**.

Only `USER_WALLET`, `PLATFORM_ESCROW_LIABILITY`, `PLATFORM_FEE_REVENUE` and
`MERCHANT_PAYABLE` can be created through the API (the `oneof` rule in
[dto/account.go](../../internal/platform/transport/http/dto/account.go)). The
seeder creates one system account of every type.

When adding a type: add the constant, add it to `AllTypes` and `IsValid`, seed
the `account_types` row, and update `TestAllTypes`.

## Status

The `account_status` SQL domain allows:

| Status | Meaning |
|---|---|
| `ACTIVE` | Can receive and send funds (default) |
| `FULLY_FROZEN` | Can neither receive nor send |
| `DEBIT_RESTRICTED` | Can receive, cannot send |
| `CREDIT_RESTRICTED` | Can send, cannot receive |
| `CLOSED` | Permanently closed |

Every change is meant to be recorded in `account_status_history`, which is
immutable:

- Each row records who made the change: an actor type of `SYSTEM` or `ADMIN`,
  plus an actor ID.
- `previous_status` must differ from `new_status`.
- Only one row per account may have a `NULL` previous status: its initial status.

`ChangeAccountStatus` locks the account row (`SELECT … FOR UPDATE`) and
supports an optimistic `ExpectedStatus` check. **Status is not exposed through
the service or API yet, and statuses are not enforced when posting.** See
[Known issues](../known-issues.md).

## Service

`account.Service` ([service.go](../../internal/domain/account/service.go)):

| Method | Rules |
|---|---|
| `CreateAccount` | Type must be valid; currency must be active in the registry |
| `GetAccountDetails` | `ErrAccountNotFound` if missing |
| `ListAccounts` | Page < 1 becomes 1; limit < 1 becomes `DefaultPageLimit` (20); limit is capped at `MaxPageLimit` (100) |
| `GetAccountBalance` | Snapshot + delta, normal-balance positive, with an exact `balance_major` |
| `GetAccountStatementSummary` | `ErrInvalidDateRange` if start > end |
| `GetAccountStatementEntries` | Same date check; limit defaults and caps as above |
| `ReconcileAccount` | See below; records a discrepancy when found |
| `CreateAccountSnapshot` | Account must exist; logs `reason` |

## Balance snapshots

A snapshot (`balance_snapshots`) records an account's balance covering every
entry up to and including `last_entry_sequence`. Snapshots make balance reads
cheap: the balance is the latest snapshot plus the entries after it, not
a scan of the whole history.

- **Creating** sums all of the account's entries with the normal-balance sign and
  stores that sum with the highest entry sequence.
- `(account_id, last_entry_sequence)` is unique, so snapshotting twice with no
  new entries returns `ErrSnapshotExists` (409).
- Snapshots are always taken at the latest entry. A snapshot at an earlier
  sequence (`target_entry_sequence`) is not supported yet.

## Reconciliation

`POST /accounts/{id}/reconcile` runs the `ReconcileAccount` query inside a
transaction that locks the account row:

```
raw_ledger_sum       = Σ all entries (normal-balance signed)
snapshot_balance     = latest snapshot balance (0 if none)
delta_since_snapshot = Σ entries with sequence > last_snapshot_sequence
calculated_total     = snapshot_balance + delta_since_snapshot
discrepancy_amount   = raw_ledger_sum − calculated_total
status               = RECONCILED if discrepancy_amount = 0, else DISCREPANCY_DETECTED
```

Since entries are immutable, a mismatch means a snapshot is wrong, either from
a bug or from tampering. On `DISCREPANCY_DETECTED` the service:

1. inserts a `discrepancy_records` row with every figure above;
2. logs a warning;
3. still returns the result with **200**, so the caller can inspect it.

If the record can't be saved, the request fails rather than reporting the
result as normal.

The repository can also list unresolved discrepancies and resolve one with
notes (`resolved_at`, `resolution_notes`). These aren't exposed through the API yet.

## Statements

- **Summary:** opening balance (entries before `start_date`), unsigned period
  debit and credit totals, and closing balance. The period's end is inclusive.
- **Entries:** cursor-paginated lines, newest first, each with the
  `running_balance` after it. The end of the window is exclusive. The running
  balance starts from the nearest snapshot before the page, so a page costs about
  the same however long the history is.

## Errors

| Error | HTTP |
|---|---|
| `ErrAccountNotFound` | 404 |
| `ErrAccountExists` | 409 |
| `ErrInvalidType`, `ErrInvalidCurrency`, `ErrInvalidDateRange` | 422 |
| `ErrSnapshotExists` | 409 |
| `ErrSnapshotNotFound` | 404 |
| `ErrDiscrepancyFound` | 409 (reserved) |
| `ErrInvalidStatus`, `ErrInvalidActorType` | not mapped yet → 500 |
