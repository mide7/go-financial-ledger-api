# Known issues and gaps

Recorded as of 2026-11-20. Remove an entry once it's fixed.

## Bugs

| Area | Issue | Fix direction |
|---|---|---|
| Account status | Changing an account to the status it already has violates the history table's `previous_status <> new_status` CHECK and returns 500 | Reject it in the repository after the row lock, as a 409 or 422 |
| Create transaction | Omitting `metadata` sends `NULL` into the `NOT NULL` JSONB column, so the request fails with 500. `metadata` is also a `[]byte`, so clients must base64-encode JSON. | Make it `json.RawMessage` and default to `{}` |
| Database errors | Trigger violations (currency mismatch, unbalanced) surface as 500 | Map the exceptions to 422 in the repository |
| Statements | The summary's end date is inclusive (`<=`) but entries' end date is exclusive (`<`) | Pick one, probably half-open |

## Not enforced yet

- **Account status:** nothing prevents posting to `FULLY_FROZEN`, `CLOSED` or
  restricted accounts, or reopening a `CLOSED` account.
- **Authentication:** there is none. Status changes record a self-reported
  `actor_id`, and the status and discrepancy endpoints are open to any caller.
- **Idempotency:** the `idempotency_records` table exists but isn't used. A
  transaction's unique `reference` is the only duplicate protection.

## Not implemented

- Transaction get, list and reverse (return 501).
- Point-in-time reconciliation (`as_of_timestamp`) and snapshots at an earlier
  sequence (`target_entry_sequence`); both are rejected with 422.
- Redis health check (always `true`) and the reserved config (`REDIS_*`,
  `ADMIN_*`, `FRONTEND_URL`).
- Withdrawals: the table is drafted, commented out, in the core migration.

## API rough edges

- Empty lists are returned as `"data": null` instead of `[]`, because the
  repository mappers return `nil`.
- `POST /transactions` returns `"entries": null` instead of the posted legs.
- Only 4 of the 10 account types can be created through the API.
- `JSONResponse`'s `SkipJSONResponse` flag is read from the request context
  before the handler runs, so only an outer middleware can set it. A handler
  can't opt out.
- `GET /discrepancies` is not paginated.

## Tooling and operations

- `.gitignore` excludes `*devbox*`, `.envrc` and `.claude`, so `devbox.json`,
  `devbox.lock` (including sqlc 1.27.0) and the project skills aren't committed.
- The production image contains only the server binary. Migrations have to be
  run separately (`cmd/migrate` from the source tree, as the dev compose does).
- `cmd/migrate` resolves migrations relative to the working directory, so it
  must run from the repository root.
- There's no goroutine-leak check for the currency registry's listener.
- File log outputs are not rotated; rely on logrotate or the platform.
