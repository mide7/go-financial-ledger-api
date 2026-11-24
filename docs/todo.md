# TODO

The work plan for closing the gaps found in the 2026-11-24 review. Every item
below is **scaffolded**: its route, DTO, service and repository methods, SQL
queries (sqlc-generated) and error mappings exist, and it answers 501 `not
implemented yet` until it's built. Items are done one route at a time.

Code markers use the item's tag, so `grep -rn "TODO(gap-1)" .` finds every
place an item touches.

**Definition of done for every item:**
- The stub is replaced by an implementation; its `TODO(<tag>)` markers are gone.
- Unit tests for the service and handler, plus an integration test when SQL is involved.
- Its case is removed from `TestScaffoldedRoutes_NotImplemented` in
  [router_test.go](../internal/platform/transport/http/router_test.go).
- The docs are updated as described in [Contributing](contributing.md#documentation-rule),
  and the item is ticked here.

## Gaps (do in this order)

### gap-1: Read transactions back

| Route | State |
|---|---|
| `GET /transactions/{id}` | **Built** (2026-11-24) |
| `GET /transactions` | **Built** (2026-11-24) |

- **Done:**
  - IDs are `uuid.UUID` end to end.
  - `ListTransactionsParams` has filters: `account_id`, `reference`, `type`,
    `currency`, `start_date` (inclusive) and `end_date` (exclusive), plus
    `cursor` and `limit`. It replaces the old `page` and `status` parameters,
    because transactions have no status column.
  - Queries: `GetTransactionByID`, `ListTransactions` (keyset on `id DESC`),
    `ListEntriesByTransactionID`.
- **Checklist:**
  - [x] `transactionRepository.GetByID`: fetch the header and its entries; `pgx.ErrNoRows` returns `ErrTransactionNotFound`.
  - [x] `transactionRepository.List`: map the filters to `pgtype` nulls.
  - [x] Service: clamp the limit as the account listings do; reject a date range unless `start_date` is before `end_date` (`ErrInvalidDateRange`, 422).
  - [x] The list endpoint returns headers only; details include entries.
  - [x] Return the posted entries from `POST /transactions` too, fixing `"entries": null`.
  - [x] Unit tests (service, handler) and the `TestTransactionReadBack` integration test. **Not yet run against a database.**

### gap-2: Reverse a transaction

| Route | State |
|---|---|
| `POST /transactions/{id}/reverse` | **Built** (2026-11-24). Body: `reference`, `reason`; answers 201 with the reversal |

- [x] Repository, in one database transaction: load the original, reject a
  `REVERSAL`, check `GetReversalOf`, insert a header with
  `parent_transaction_id` set to the original and the same currency, insert
  the original's entries with DEBIT and CREDIT swapped.
- [x] Concurrent reversals: the partial unique index `uq_transactions_one_reversal`
  on `transactions (parent_transaction_id) WHERE type = 'REVERSAL'`, edited
  into the performance-indexes migration; its violation maps to
  `ErrAlreadyReversed`. This replaces locking the original.
- [x] `reason` is stored as the reversal's `description`.
- [x] `POST /transactions` rejects `type` `REVERSAL` (`ErrReservedType`, 422), so a fake reversal can't block the real one.
- [x] Unit tests and the `TestTransactionReversal` / `TestTransactionReversal_Concurrent` integration tests. **Not yet run against a database.**
- Status checks on reversal: done in gap-3.

### gap-3: Enforce account status when posting

**Built** (2026-11-24). No route of its own; applies to `POST /transactions`
and reversals now, and conversions and withdrawals when they're built (call
`lockPostableAccounts` in their repository code).

- [x] Lock the entry accounts (`LockAccountsForPosting`, `FOR SHARE`, id order)
  before inserting, and reject entries the status forbids (`ErrAccountNotPostable`, 422).
  An unknown entry account is now `ErrEntryAccountNotFound` (422) instead of a foreign-key 500.
- [x] Restricted statuses are in ledger terms: `DEBIT_RESTRICTED` blocks DEBIT
  entries, `CREDIT_RESTRICTED` blocks CREDIT entries; `FULLY_FROZEN` and `CLOSED` block both.
- [x] Status changes: a change to the current status is `ErrStatusUnchanged` (409);
  `CLOSED` is final (`ErrAccountClosed`, 409).
- [x] Reversals skip restrictions and freezes (they're corrections); only `CLOSED` refuses them.
  This is broader than the earlier recommendation, which exempted only `FULLY_FROZEN`.
- [x] `Status.AllowsEntry`, `AllowsReversal` and `CheckTransition`, with table tests, plus the
  `TestPostingRespectsAccountStatus` and `TestChangeAccountStatus_Transitions` integration tests.
  **Not yet run against a database.**

### gap-4: Find an owner's accounts

| Route | State |
|---|---|
| `GET /accounts?owner_id=…` | **Built** (2026-11-24). The DTO passes `owner_id` to the existing repository filter |

- [x] DTO field and handler test.
- [x] Integration test `TestListAccounts_OwnerFilter`: two owners, filtering returns only one owner's accounts (also combined with `type`). **Not yet run against a database.**

### gap-5: Idempotent retries (`Idempotency-Key`)

**Built** (2026-11-24). See [API → Idempotency-Key](api.md#idempotency-key-header).

- [x] Store adapter `repository.NewIdempotencyStore`: insert-or-nothing, then
  `ReclaimIdempotencyRecord` (new query) for an expired key or a FAILED one
  with the same fingerprint, then get. `Complete`, `Fail` (new) and `DeleteExpired`.
- [x] Middleware `middleware.Idempotency(store, ttl, log)`:
  SHA-256 of method, path and body; claim; replay a completed record's status,
  message and body (`Idempotent-Replayed: true`); 409 in progress; 422 when the
  request differs; 400 for a bad key; 413 over 1 MiB; saves the response.
- [x] Wired for `POST /transactions`, reversals, conversions and withdrawals;
  TTL from `IDEMPOTENCY_TTL` (default 24 h).
- [x] Expired records are purged hourly by a ticker in `cmd/server`.
- [x] A 5xx response or a panic marks the record FAILED, and a retry of the same request runs again. 4xx responses are saved and replayed.
- [x] Unit tests (middleware with an in-memory store, router wrapping, config) and
  `TestIdempotencyStore_*` integration tests. **Not yet run against a database.**
- The `idempotency.ErrKeyReused` and `ErrInProgress` responses are written by the
  middleware itself, not `httputil.Error` (which imports the middleware package).

### gap-6: Authentication (deferred)

Deliberately skipped until gaps 1–5 and 7 are done. Afterwards: authenticate
callers, scope `Idempotency-Key`s per caller, record the real actor on status changes (replacing the self-reported
`actor_id`), and restrict the admin routes (status, discrepancies, reconciliations, withdrawals settle/fail).

### gap-7: Correctness rough edges

**Built** (2026-11-24).

- [x] **Trigger violations returned 500.** The integrity trigger now raises
  custom SQLSTATEs (`LD001` currency mismatch, `LD002` unbalanced; edited into
  the trigger migration), and the repository maps them to `ErrCurrencyMismatch`
  and `ErrUnbalancedTransaction` (both 422).
- [x] **Statement end dates differed.** The summary is now half-open
  (`start ≤ t < end`), matching statement entries and the transaction list.
- [x] **Empty lists were `null`.** `mapToAccountDomainList` was the only mapper
  returning `nil`; it now returns `[]`. Transaction-list items omit `entries`.
- [x] **`metadata`** is `json.RawMessage`: clients send a plain JSON object;
  omitted or `null` is stored as `{}`; other JSON types are `ErrInvalidMetadata` (422).
- [x] **Changing to the same status returned 500:** fixed in gap-3.
- [x] Unit tests plus `TestIntegrityTriggerErrors` and
  `TestStatementSummary_EndIsExclusive`. **Not yet run against a database.**

## Later (scaffolded routes)

| Tag | Route(s) | Scaffolded | Still needed |
|---|---|---|---|
| `later-fx` | `POST /conversions` | `CreateConversionParams`, `Conversion`, DTO, handler | Rate source and precision rules, the two legs through `PLATFORM_FX_SUSPENSE`, gain/loss into `PLATFORM_FX_GAIN_LOSS`, a check that source and target accounts differ |
| `later-withdrawals` | `POST /withdrawals`, `GET /withdrawals`, `GET /withdrawals/{id}`, `POST /withdrawals/{id}/settle`, `POST /withdrawals/{id}/fail` | The new `withdrawal` domain (model, params, `Repository` port, stub `Service`), DTOs, handlers | Uncomment the `withdrawals` table in the core migration, write its queries and adapter, and lock/settle/release the funds through `USER_LOCKED` |
| `later-reconcile-all` | `POST /reconciliations` | `ReconcileAllAccounts`, `ReconciliationRun`, the `ListAccountIDs` port and the `ListAccountIDsAfter` query | Page through accounts and reconcile each, plus a schedule (cron or ticker) and a snapshot pass |

## Later (no new route)

- [ ] Point-in-time reconciliation (`as_of_timestamp` on `POST /accounts/{id}/reconcile`). It's rejected with 422 today.
- [ ] Snapshots at an earlier sequence (`target_entry_sequence` on `POST /accounts/{id}/snapshot`). It's rejected with 422 today.
- [ ] Paging for `GET /discrepancies` (a cursor on `detected_at, id`).
- [ ] A real Redis health check, or drop `redis` from `/health/` until Redis is used.
