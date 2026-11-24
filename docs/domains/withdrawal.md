# Withdrawal domain

Package: [`internal/domain/withdrawal`](../../internal/domain/withdrawal)
Adapter: none yet

**Scaffolded only** ([later-withdrawals](../todo.md#later-scaffolded-routes)).
Every service method returns `domain.ErrNotImplemented` (501), and there's no
repository adapter, because the `withdrawals` table is still drafted, commented
out, in the core migration.

A withdrawal pays a user's funds out through an external processor (for
example Paystack). It moves through the ledger in two steps:

1. **Lock.** `CreateWithdrawal` posts a transaction moving the amount from the
   user's `USER_WALLET` to `USER_LOCKED`, and stores it as `lock_transaction_id`.
   Status `PROCESSING`.
2. **Resolve.** When the processor reports back:
   - `SettleWithdrawal` posts `USER_LOCKED` → `PLATFORM_CASH` (the money left),
     and stores the processor's reference. Status `SETTLED`.
   - `FailWithdrawal` posts `USER_LOCKED` → `USER_WALLET` (the funds are
     released). Status `FAILED`.

   Either way, the posted transaction is stored as `resolution_transaction_id`.

Only a `PROCESSING` withdrawal can be resolved; otherwise the result is
`ErrNotProcessing`. The exact postings above are the plan, to be confirmed when
the item is built.

## Withdrawal

| Field | Notes |
|---|---|
| `id` | UUIDv7 |
| `account_id`, `amount`, `currency` | The account paid out from, in minor units |
| `lock_transaction_id` | The transaction that locked the funds |
| `resolution_transaction_id` | The settling or releasing transaction, once resolved |
| `processor`, `processor_reference` | E.g. `PAYSTACK`, and its payout ID once known; unique together |
| `status` | `PROCESSING`, `SETTLED` or `FAILED` |
| `created_at`, `updated_at`, `resolved_at` | |

## Service

| Method | State |
|---|---|
| `CreateWithdrawal(CreateWithdrawalParams)` | 501 |
| `GetWithdrawal(id)` | 501 |
| `ListWithdrawals(ListWithdrawalsParams)` | 501; filters by account and status, keyset cursor; returns a `CursorPaginatedResult` |
| `SettleWithdrawal(SettleWithdrawalParams)` | 501 |
| `FailWithdrawal(FailWithdrawalParams)` | 501 |

## Errors

| Error | HTTP |
|---|---|
| `ErrWithdrawalNotFound` | 404 (reserved) |
| `ErrNotProcessing` | 409 (reserved) |
