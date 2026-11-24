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
| POST | `/api/v1/transactions` | Post a balanced transaction | 201 |
| GET | `/api/v1/transactions` | List transactions | **501** |
| GET | `/api/v1/transactions/{id}` | Transaction details | **501** |
| POST | `/api/v1/transactions/{id}/reverse` | Reverse a transaction | **501** |

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
- An empty list is currently returned as `"data": null`, not `[]`. See [Known issues](known-issues.md).

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
| 422 | `debits and credits must balance to zero` | Unbalanced transaction |
| 422 | `invalid entry type` / `entry amount must be greater than zero` / `entry account ID is required` | Bad leg |
| 422 | `total debits or credits exceed the maximum amount` | A side's total overflows int64 |
| 404 | `transaction not found` | Reserved for transaction lookup |
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
  `start_date ≤ created_at ≤ end_date`, with the end **inclusive**.
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

## Transactions

### `POST /api/v1/transactions`

```json
{
  "reference": "order-1001",
  "type": "WALLET_FUNDING",
  "currency": "NGN",
  "description": "Card top-up",
  "metadata": "e30=",
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
| `type` | required, 1–64 chars; free-form business label |
| `currency` | 3 chars, must be active; every entry's account must have this currency (enforced by a DB trigger) |
| `parent_transaction_id` | optional UUID |
| `description` | required, ≤ 256 chars |
| `metadata` | **base64-encoded JSON** (`"e30="` is `{}`). Effectively required for now; see [Known issues](known-issues.md) |
| `transaction_date` | required, RFC 3339; the business date, separate from `created_at` |
| `entries` | ≥ 2 legs; each has a UUID `account_id`, `type` `DEBIT`/`CREDIT`, and `amount` > 0. Debits must equal credits. |

Returns 201 `"transaction created"` with the transaction header. The response's
`entries` field is currently `null`; the posted legs aren't echoed back.

### `GET /api/v1/transactions`, `GET /api/v1/transactions/{id}`, `POST /api/v1/transactions/{id}/reverse`

These validate their input, then return **501** `not implemented yet`. Query
parameters accepted by the list endpoint: `page`, `limit` (1–100), `type`, and
`status` (`POSTED`, `REVERSED` or `FAILED`).
