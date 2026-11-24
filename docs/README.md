# Financial Ledger API — Documentation

A double-entry ledger service written in Go. It stores money movements as
immutable, balanced transactions on accounts, derives balances from those
entries, and can checkpoint and reconcile balances to detect corruption.

| Document | What it covers |
|---|---|
| [Getting started](getting-started.md) | Local setup, configuration, Make targets, Docker |
| [Architecture](architecture.md) | Layers, package layout, request flow, conventions |
| [HTTP API](api.md) | Every endpoint, request/response formats, error codes |
| [Database](database.md) | Schema, triggers, indexes, sign convention, migrations, sqlc |
| [Testing](testing.md) | Unit, integration and fuzz tests; how to run them |
| [Known issues](known-issues.md) | Gaps and bugs not yet fixed |
| [TODO](todo.md) | Planned work, one item per route or fix, with what's scaffolded |
| [Contributing](contributing.md) | The documentation rule and pre-commit checks |

## Domains

| Domain | Package | Responsibility |
|---|---|---|
| [Account](domains/account.md) | `internal/domain/account` | Accounts, balances, statements, snapshots, reconciliation, status |
| [Transaction](domains/transaction.md) | `internal/domain/transaction` | Posting balanced double-entry transactions |
| [Currency](domains/currency.md) | `internal/domain/currency` | Currency codes, minor units, the live currency registry |
| [Withdrawal](domains/withdrawal.md) | `internal/domain/withdrawal` | Payouts to external processors (scaffolded only) |
| [Shared kernel](domains/shared.md) | `internal/domain`, `internal/pagination` | Entry types, `ErrNotImplemented`, pagination envelopes |

## Core ideas in one minute

- **Double entry.** Every transaction has two or more entries (legs). Each entry
  debits or credits one account by a positive amount in minor units. The debits
  and credits in a transaction must be equal, which Go and a database trigger both check.
- **Immutability.** Transactions and entries cannot be updated or deleted; a
  database trigger rejects it. Corrections are new transactions.
- **Derived balances.** There is no stored "balance" column. A balance is the
  sum of an account's entries, optionally starting from a **balance snapshot**.
- **Normal-balance sign convention.** Every balance is positive in the account's
  natural direction (`account_types.normal_balance`). A funded user wallet
  (credit-normal) and a funded cash account (debit-normal) both read positive.
  See [Database → Sign convention](database.md#balance-sign-convention).
- **Reconciliation.** Recomputing an account's balance from all entries and
  comparing it to *snapshot + entries since* detects tampering or bugs; a
  mismatch is stored as a **discrepancy record**.

## Status

The project is in active development (as of 2026-11-24).

- **Working end to end:**
  - accounts, including lookup by owner;
  - balances, statements and snapshots;
  - per-account reconciliation;
  - account status changes and history, with statuses enforced when posting;
  - discrepancy listing and resolution;
  - the currency list;
  - transaction creation, lookup, listing and reversal.
  - `Idempotency-Key` retries on the money-moving POST routes.
- **Scaffolded, returning `501 Not Implemented`:** currency conversion; reconciling every account; withdrawals.
  [TODO](todo.md) tracks each of these.

[Known issues](known-issues.md) lists current bugs and limitations.
