# Known issues and gaps

Last reviewed 2026-11-24. Remove an entry once it's fixed. [TODO](todo.md)
holds the plan for each; the tags in brackets name its item.

## Bugs

None known as of 2026-11-24.

## Not enforced yet

- **Authentication:** there is none. Status changes record a self-reported
  `actor_id`, and the status and discrepancy endpoints are open to any caller.
  [gap-6, deferred]
- **Idempotency keys are global:** without authentication, one caller's
  `Idempotency-Key` can collide with another's. [gap-6, deferred]

## Not implemented

- Currency conversion
  [later-fx], reconciling every account [later-reconcile-all] and withdrawals
  [later-withdrawals]. All are scaffolded and return 501.
- Point-in-time reconciliation (`as_of_timestamp`) and snapshots at an earlier
  sequence (`target_entry_sequence`); both are rejected with 422.
- Redis health check (always `true`) and the reserved config (`REDIS_*`,
  `ADMIN_*`, `FRONTEND_URL`).
- Withdrawals: the table is still drafted, commented out, in the core migration.

## API rough edges

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
