# HTTP API

Base path: `/api/v1`. All bodies are JSON. All money amounts are **integers in
minor units** (e.g. `10000` = ₦100.00 for NGN, exponent 2).

## Endpoints

| Method | Path | Purpose | Success |
|---|---|---|---|
| GET | `/health/` | Dependency health | 200 / 503 |
| POST | `/api/v1/accounts` | Open an account | 201 |
| GET | `/api/v1/accounts` | List accounts | 200 |
| GET | `/api/v1/accounts/{id}` | Account details | 200 |
| GET | `/api/v1/accounts/{id}/balance` | Current balance | 200 |
| GET | `/api/v1/accounts/{id}/statement/summary` | Opening/closing balance and totals for a period | 200 |
| GET | `/api/v1/accounts/{id}/statement/entries` | Statement lines with running balance | 200 |
| POST | `/api/v1/accounts/{id}/reconcile` | Reconcile ledger against latest snapshot | 200 |
| POST | `/api/v1/accounts/{id}/snapshot` | Checkpoint the current balance | 201 |
| GET | `/api/v1/accounts/{id}/snapshot` | Latest balance snapshot | 200 |
| PATCH | `/api/v1/accounts/{id}/status` | Change the account's status | 200 |
| GET | `/api/v1/accounts/{id}/status/history` | Status changes, newest first | 200 |
| GET | `/api/v1/discrepancies` | Unresolved reconciliation discrepancies | 200 |
| POST | `/api/v1/discrepancies/{id}/resolve` | Mark a discrepancy resolved | 200 |
| GET | `/api/v1/currencies` | Active currencies | 200 |
| POST | `/api/v1/transactions` | Post a balanced transaction | 201 |
| GET | `/api/v1/transactions` | List transactions | 200 |
| GET | `/api/v1/transactions/{id}` | Transaction details with entries | 200 |
| POST | `/api/v1/transactions/{id}/reverse` | Reverse a transaction | 201 |
| POST | `/api/v1/conversions` | Convert between currencies | **501** ([later-fx](todo.md#later-scaffolded-routes)) |
| POST | `/api/v1/reconciliations` | Reconcile every account | **501** ([later-reconcile-all](todo.md#later-scaffolded-routes)) |
| POST | `/api/v1/withdrawals` | Lock funds and open a withdrawal | **501** ([later-withdrawals](todo.md#later-scaffolded-routes)) |
| GET | `/api/v1/withdrawals` | List withdrawals | **501** |
| GET | `/api/v1/withdrawals/{id}` | Withdrawal details | **501** |
| POST | `/api/v1/withdrawals/{id}/settle` | Record a paid-out withdrawal | **501** |
| POST | `/api/v1/withdrawals/{id}/fail` | Record a failed withdrawal, releasing the funds | **501** |

Routes marked **501** are scaffolded: they decode and validate their input as
documented below, then answer 501 `not implemented yet`. [todo.md](todo.md) tracks them.

`/health` without the trailing slash is redirected (301) to `/health/`.

## Request formats

- **Path IDs** must be UUIDs. A malformed ID returns 422 `invalid path parameters`.
- **Dates** in query strings and bodies use RFC 3339, e.g. `2026-01-31T00:00:00Z`.
  URL-encode `+` offsets (`%2B01:00`).
- **Unsupported fields** that the API defines but can't honor yet
  (`as_of_timestamp`, `target_entry_sequence`) return 422 if sent with a value.

## Response envelope

The `JSONResponse` middleware wraps every response body:

```json
{
  "status": 201,
  "message": "account created",
  "timestamp": "2026-11-20T10:15:00Z",
  "data": { "...": "handler payload" },
  "errors": null
}
```

- `message` is the handler's custom message, or the HTTP status text.
- For status < 400 the payload is in `data` and `errors` is `null`. For status ≥ 400 it's the reverse.
- An empty list is returned as `"data": []`.

## Errors

On error, `errors` holds either a message string or, for validation failures, a
list of field errors:

```json
{
  "status": 422,
  "message": "Unprocessable Entity",
  "errors": [
    { "field": "owner_id", "message": "required" },
    { "field": "currency", "message": "must be exactly 3 characters" }
  ]
}
```

`field` is the name the client sent: the JSON key or query parameter.

| Status | `errors` message | Cause |
|---|---|---|
| 422 | *field list* | Request failed validation |
| 422 | `invalid JSON body` / `invalid query parameters` / `invalid path parameters` | Could not decode that part of the request |
| 404 | `account not found` | Unknown account ID |
| 404 | `no balance snapshot exists for this account` | No snapshot yet |
| 409 | `account already exists` | Same owner + type + currency already exists |
| 409 | `no new entries since the last balance snapshot` | Snapshot would duplicate the previous one |
| 409 | `account balance discrepancy detected` | Reserved; reconcile currently returns the result with 200 |
| 422 | `invalid currency code` | Currency unknown or inactive |
| 422 | `invalid account type` | Type not in the chart of accounts |
| 422 | `start_date must not be after end_date` | Bad statement window |
| 422 | `duplicate reference` | Transaction reference already used |
| 422 | `transaction must contain at least two entries` | Fewer than two legs |
| 422 | `debits and credits must balance to zero` | Unbalanced transaction (checked in Go, and again by the database trigger) |
| 422 | `invalid entry type` / `entry amount must be greater than zero` / `entry account ID is required` | Bad leg |
| 422 | `total debits or credits exceed the maximum amount` | A side's total overflows int64 |
| 404 | `transaction not found` | Unknown transaction id (lookup, reversal) |
| 422 | `start_date must be before end_date` | Bad transaction-list window |
| 422 | `transaction type REVERSAL is reserved` | `POST /transactions` with `type: REVERSAL` |
| 409 | `transaction has already been reversed` | The transaction already has a reversal |
| 422 | `a reversal cannot itself be reversed` | Reversing a `REVERSAL` |
| 422 | `account status does not allow this entry` | An entry's account status refuses it ([rules](domains/account.md#status)) |
| 422 | `entry account not found` | An entry names an account that doesn't exist |
| 422 | `entry account currency does not match the transaction currency` | An entry's account has another currency (reported by the database trigger) |
| 422 | `metadata must be a JSON object` | `metadata` is an array, string or number |
| 422 | `invalid account status` / `invalid actor type` | Bad status change |
| 409 | `account is not in the expected status` | `expected_status` didn't match |
| 409 | `account is already in that status` | Status change to the current status |
| 409 | `account is closed; its status can't change` | Status change on a `CLOSED` account |
| 404 | `discrepancy not found or already resolved` | Unknown or resolved discrepancy |
| 404 | `withdrawal not found` / 409 `withdrawal is no longer processing` | Reserved for withdrawals |
| 400 | `Idempotency-Key must be 1–256 characters` | Empty or over-long [`Idempotency-Key`](#idempotency-key-header) |
| 413 | `request body too large` | Body over 1 MiB on a route that takes `Idempotency-Key`, when the header is sent |
| 422 | `idempotency key reused with a different request` | Same key, different method, path or body |
| 409 | `a request with this idempotency key is in progress` | Retry while the first request with the key is still running |
| 501 | `not implemented yet` | Endpoint has no implementation yet |
| 500 | `an unexpected error occurred` | Anything else (details are logged, not returned) |

The authoritative mapping is `errorMappings` in
[httputil/response.go](../internal/platform/transport/http/httputil/response.go).

---

## Health

### `GET /health/`

Pings the database. Redis is not checked yet and always reports `true`.

```json
// 200 "services healthy"   (503 "services unhealthy" when db is false)
{ "db": true, "redis": true }
```

## Accounts

### `POST /api/v1/accounts`

```json
{ "owner_id": "user-42", "type": "USER_WALLET", "currency": "NGN" }
```

| Field | Rules |
|---|---|
| `owner_id` | required, 1–64 chars |
| `type` | one of `USER_WALLET`, `PLATFORM_ESCROW_LIABILITY`, `PLATFORM_FEE_REVENUE`, `MERCHANT_PAYABLE` |
| `currency` | 3 uppercase letters, must be an active currency |

Returns 201 `"account created"` with the [account](domains/account.md#account).
Returns 409 if the owner already has an account of that type and currency.

### `GET /api/v1/accounts`

| Query | Rules |
|---|---|
| `page` | ≥ 1, default 1 |
| `limit` | 1–100, default 20 |
| `currency` | optional filter, 3 uppercase letters |
| `type` | optional filter, same values as create |
| `owner_id` | optional filter, ≤ 64 chars; finds an owner's accounts |

Results are sorted newest first (`created_at DESC, id DESC`).

### `GET /api/v1/accounts/{id}`

Returns the account, or 404.

### `GET /api/v1/accounts/{id}/balance`

```json
{ "currency": "NGN", "exponent": 2, "balance_minor": 750000, "balance_major": "7500.00" }
```

The balance is positive in the account's normal-balance direction
([why](database.md#balance-sign-convention)). It's computed as the latest
snapshot plus the entries after it. `balance_major` is an exact decimal string.

### `GET /api/v1/accounts/{id}/statement/summary?start_date=…&end_date=…`

Both dates are required, and `start_date` must not be after `end_date`.

```json
{
  "account_id": "0190…", "currency": "NGN", "exponent": 2,
  "start_date": "2026-01-01T00:00:00Z", "end_date": "2026-02-01T00:00:00Z",
  "opening_balance": 0, "closing_balance": 750000,
  "total_debits": 250000, "total_credits": 1000000
}
```

- The opening balance covers entries before `start_date`. The period covers
  `start_date ≤ created_at < end_date`: the end is **exclusive**, as for
  statement entries and the transaction list.
- Totals are unsigned. The closing balance is the opening balance plus the
  period's movement in the account's normal direction.

### `GET /api/v1/accounts/{id}/statement/entries?start_date=…&end_date=…[&limit=…][&cursor=…]`

Returns statement lines **newest first**:

```json
[
  { "id": "0190…", "transaction_id": "0190…", "type": "DEBIT", "amount": 250000,
    "running_balance": 750000, "currency": "NGN", "exponent": 2,
    "description": "", "entry_sequence": 42, "created_at": "…" }
]
```

| Query | Rules |
|---|---|
| `start_date`, `end_date` | required. The window is `start_date ≤ created_at < end_date` (end **exclusive**) |
| `limit` | 1–100, default 20 |
| `cursor` | the `id` of the last entry on the previous page |

`running_balance` is the account balance immediately after that entry.

### `POST /api/v1/accounts/{id}/reconcile`

The body is optional. `as_of_timestamp` is rejected; reconciliation always runs
against the current ledger. Returns 200 `"account reconciled"`:

```json
{
  "account_id": "0190…", "status": "RECONCILED", "currency": "NGN", "exponent": 2,
  "raw_ledger_sum": 750000, "snapshot_balance": 1000000, "delta_since_snapshot": -250000,
  "calculated_total": 750000, "discrepancy_amount": 0,
  "last_snapshot_sequence": 40, "current_max_sequence": 42,
  "reconciled_at": "…"
}
```

`status` is `RECONCILED` or `DISCREPANCY_DETECTED`. A discrepancy is still a 200
response, and it is also saved as a discrepancy record. See
[Reconciliation](domains/account.md#reconciliation).

### `POST /api/v1/accounts/{id}/snapshot`

```json
{ "reason": "end of day" }
```

`reason` is required (1–128 chars). It is logged but not stored, because there's
no column for it. `target_entry_sequence` is rejected. Returns 201
`"account snapshot created"`:

```json
{ "id": "0190…", "account_id": "0190…", "balance": 750000, "last_entry_sequence": 42, "created_at": "…" }
```

Returns 409 if nothing has been posted since the last snapshot.

### `GET /api/v1/accounts/{id}/snapshot`

Returns the account's most recent snapshot, in the same shape as the create
response. Returns 404 `no balance snapshot exists for this account` if none has been taken.

### `PATCH /api/v1/accounts/{id}/status`

```json
{ "status": "FULLY_FROZEN", "expected_status": "ACTIVE", "reason": "KYC review", "actor_id": "admin-7" }
```

| Field | Rules |
|---|---|
| `status` | required; one of the [account statuses](domains/account.md#status) |
| `expected_status` | optional; the change applies only if the account is still in this status, else 409 |
| `reason` | required, 1–256 chars |
| `actor_id` | required, 1–64 chars. **Self-reported** until the API has authentication |

API callers are always recorded with actor type `ADMIN`. Returns 200
`"account status changed"` with the new history row:

```json
{ "id": "0190…", "account_id": "0190…", "previous_status": "ACTIVE", "new_status": "FULLY_FROZEN",
  "reason": "KYC review", "actor_type": "ADMIN", "actor_id": "admin-7", "created_at": "…" }
```

Changing to the current status is 409 `account is already in that status`, and
a `CLOSED` account can't change status at all (409 `account is closed; its
status can't change`). `expected_status` is checked first.

### `GET /api/v1/accounts/{id}/status/history[?limit=…][&cursor=…&cursor_created_at=…]`

Returns history rows (same shape as above) **newest first**.

| Query | Rules |
|---|---|
| `limit` | 1–100, default 20 |
| `cursor`, `cursor_created_at` | the `id` and `created_at` of the last row on the previous page; send both or neither |

## Discrepancies

A discrepancy is recorded when [reconciliation](#post-apiv1accountsidreconcile)
finds a mismatch.

### `GET /api/v1/discrepancies`

Returns every unresolved discrepancy, newest first. The list is not paginated.

```json
[
  { "id": "0190…", "account_id": "0190…", "raw_ledger_sum": 750000, "snapshot_balance": 1000000,
    "calculated_total": 750100, "discrepancy_amount": 100,
    "last_snapshot_sequence": 40, "current_max_sequence": 42, "detected_at": "…" }
]
```

### `POST /api/v1/discrepancies/{id}/resolve`

```json
{ "notes": "fee leg posted twice; corrected by order-1002" }
```

`notes` is required (1–1024 chars). Returns 200 `"discrepancy resolved"` with
no data. Returns 404 if the discrepancy doesn't exist or is already resolved.

## Currencies

### `GET /api/v1/currencies`

Returns the active currencies from the in-memory registry, sorted by code:

```json
[ { "code": "EUR", "name": "Euro", "exponent": 2, "is_active": true }, … ]
```

## Transactions

### `POST /api/v1/transactions`

```json
{
  "reference": "order-1001",
  "type": "WALLET_FUNDING",
  "currency": "NGN",
  "description": "Card top-up",
  "metadata": { "channel": "card" },
  "transaction_date": "2026-11-20T10:00:00Z",
  "entries": [
    { "account_id": "<platform cash id>", "type": "DEBIT",  "amount": 1000000 },
    { "account_id": "<user wallet id>",   "type": "CREDIT", "amount": 1000000 }
  ]
}
```

| Field | Rules |
|---|---|
| `reference` | required, 1–128 chars, **unique** (acts as an idempotency key; reuse returns 422 `duplicate reference`) |
| `type` | required, 1–64 chars; free-form business label. `REVERSAL` is reserved (422); use the reverse endpoint |
| `currency` | 3 chars, must be active; every entry's account must have this currency (enforced by a DB trigger) |
| `parent_transaction_id` | optional UUID |
| `description` | required, ≤ 256 chars |
| `metadata` | optional JSON **object**, stored as JSONB; omitted or `null` is stored as `{}`. Any other JSON type is 422 `metadata must be a JSON object` |
| `transaction_date` | required, RFC 3339; the business date, separate from `created_at` |
| `entries` | ≥ 2 legs; each has a UUID `account_id`, `type` `DEBIT`/`CREDIT`, and `amount` > 0. Debits must equal credits. Each account must exist (422 `entry account not found`) and its [status](domains/account.md#status) must allow the entry's type (422 `account status does not allow this entry`) |

Returns 201 `"transaction created"` with the transaction and its posted
`entries` (the same shape as `GET /transactions/{id}`).

### `GET /api/v1/transactions/{id}`

Returns the transaction with its `entries` in `entry_sequence` order, or 404
`transaction not found`. A malformed id is 400.

```json
{
  "id": "019a…",
  "reference": "order-1001",
  "type": "WALLET_FUNDING",
  "currency": "NGN",
  "parent_transaction_id": null,
  "description": "Card top-up",
  "metadata": { "channel": "card" },
  "transaction_date": "2026-11-20T10:00:00Z",
  "created_at": "2026-11-20T10:00:01Z",
  "entries": [
    {
      "id": "019a…", "transaction_id": "019a…", "account_id": "<platform cash id>",
      "type": "DEBIT", "amount": 1000000, "entry_sequence": 41, "created_at": "2026-11-20T10:00:01Z"
    },
    {
      "id": "019a…", "transaction_id": "019a…", "account_id": "<user wallet id>",
      "type": "CREDIT", "amount": 1000000, "entry_sequence": 42, "created_at": "2026-11-20T10:00:01Z"
    }
  ]
}
```

### `GET /api/v1/transactions`

Returns transaction headers **newest first**, without entries (the `entries`
field is omitted); fetch `GET /transactions/{id}` for the legs. To get the next
page, pass the last item's `id` as `cursor`; a page shorter than `limit` is the
last one.

| Query | Rules |
|---|---|
| `account_id` | optional UUID; transactions with an entry on this account |
| `reference` | optional, ≤ 128 chars |
| `type` | optional, ≤ 64 chars |
| `currency` | optional, 3 uppercase letters |
| `start_date`, `end_date` | optional RFC 3339; `start_date ≤ created_at < end_date`. When both are given, `start_date` must be before `end_date`, else 422 `start_date must be before end_date` |
| `cursor` | the `id` of the last transaction on the previous page |
| `limit` | 1–100, default 20 |

### `POST /api/v1/transactions/{id}/reverse`

```json
{ "reference": "order-1001-reversal", "reason": "duplicate charge" }
```

`reference` (1–128 chars, unique) and `reason` (1–256 chars) are required.
Posts a `REVERSAL` transaction that mirrors the original's entries with debit
and credit swapped, in the original's currency, so every affected balance
returns to what it was before the original. The reversal's
`parent_transaction_id` is the original, its `description` is the `reason`,
`metadata` is `{}` and `transaction_date` is the time of the reversal.

Returns 201 `"transaction reversed"` with the reversal and its entries (the
shape of `GET /transactions/{id}`).

| Status | When |
|---|---|
| 404 `transaction not found` | No transaction with that id |
| 409 `transaction has already been reversed` | It already has a reversal, including when two reversals race |
| 422 `a reversal cannot itself be reversed` | The id is a `REVERSAL` |
| 422 `duplicate reference` | `reference` is already used by any transaction |

Reversal entries ignore restrictions and freezes, but a `CLOSED` account
refuses them: 422 `account status does not allow this entry`.

### `POST /api/v1/conversions` (501)

```json
{ "reference": "fx-1", "source_account_id": "<NGN wallet>", "target_account_id": "<USD wallet>",
  "source_amount": 150000, "rate": "0.00065" }
```

`rate` is target units per source unit, as a decimal string. The conversion
will be posted as one transaction per currency through the
`PLATFORM_FX_SUSPENSE` accounts, and will answer 201 `"conversion created"`
with `{reference, rate, source, target}`.

## Reconciliation jobs

### `POST /api/v1/reconciliations` (501)

Will reconcile every account (see [Reconciliation](domains/account.md#reconciliation))
and answer 200 `"reconciliation run finished"` with
`{accounts_checked, discrepancies_found, started_at, finished_at}`.

## Withdrawals (501)

Scaffolded routes; see [Withdrawal domain](domains/withdrawal.md).

| Route | Body or query |
|---|---|
| `POST /api/v1/withdrawals` | `{ "account_id", "amount" (> 0), "processor" (1–64), "reference" (1–128) }` → 201 |
| `GET /api/v1/withdrawals` | `account_id`, `status` (`PROCESSING`, `SETTLED`, `FAILED`), `cursor`, `limit` (1–100) |
| `GET /api/v1/withdrawals/{id}` | none |
| `POST /api/v1/withdrawals/{id}/settle` | `{ "processor_reference" (1–128) }` |
| `POST /api/v1/withdrawals/{id}/fail` | `{ "reason" (1–256) }` |

## `Idempotency-Key` header

These routes accept an optional `Idempotency-Key` header (1–256 characters),
which makes a retry safe after a timeout or dropped connection:

- `POST /api/v1/transactions`
- `POST /api/v1/transactions/{id}/reverse`
- `POST /api/v1/conversions`
- `POST /api/v1/withdrawals`

Other routes ignore the header. A request without it behaves as before.

The request is fingerprinted by SHA-256 of its method, path and body, and the
key is kept for `IDEMPOTENCY_TTL` (default 24 h; see
[Getting started](getting-started.md#configuration)).

| Situation | Response |
|---|---|
| First request with the key | Runs normally. A response below 500 is saved |
| Retry, same key and request, first one finished | The saved status, message and `data`/`errors`, with header `Idempotent-Replayed: true`. The handler doesn't run again. This includes saved 4xx responses |
| Retry while the first is still running | 409 `a request with this idempotency key is in progress` |
| Same key, different method, path or body | 422 `idempotency key reused with a different request` |
| The first request got a 5xx or crashed | The key is marked failed; a retry of the same request runs again |
| The key has expired | Treated as a new key |

Replayed JSON is equal in value to the original, but object keys may come back
in a different order (the response is stored as JSONB). The envelope's
`timestamp` is the time of the replay.

Keys are global, not per caller, because the API has no authentication yet
([gap-6](todo.md#gap-6-authentication-deferred)); use unguessable keys such as
UUIDs. The transaction `reference` remains the ledger's own duplicate check:
a new key with a reused reference still gets 422 `duplicate reference`.
