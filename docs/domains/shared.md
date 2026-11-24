# Shared kernel

Small packages used by several domains.

## `internal/domain`

[domain.go](../../internal/domain/domain.go)

- **`EntryType`**: `DEBIT` or `CREDIT`, the side of an entry. Use
  `ParseEntryType` (case-sensitive) and `IsValid`. It matches the Postgres enum
  `entry_type`, which is also used for `account_types.normal_balance`.
- **`ErrNotImplemented`**: returned by operations that exist in the API but
  have no implementation yet. The HTTP layer maps it to **501**. It exists so
  that a stub can never look like success by returning `nil, nil`.

## `internal/pagination`

[pagination.go](../../internal/pagination/pagination.go) defines the page
every list endpoint returns ([API → Pagination](../api.md#pagination)):

- `OffsetPaginatedResult[T]`: `items`, `page` (1-based), `limit`,
  `total_count`, `has_more`. Built by `NewOffsetPaginatedResult(items, page,
  limit, total)`; used by `ListAccounts`, whose repository has a matching `Count`.
- `CursorPaginatedResult[T]`: `items`, `limit`, `has_more`, `next_cursor`, and
  `next_cursor_created_at` for keysets that include a timestamp.

Cursor listings follow one pattern in the service: clamp the limit, ask the
repository for `FetchLimit(limit)` rows (one extra), then
`NewCursorPaginatedResult(rows, limit, cursorOf)` drops the extra row, sets
`has_more` and takes the cursor from the last item. Repositories just return
"up to Limit rows". Both constructors turn a nil slice into `[]`.

## `internal/config`

`config.Load()` builds a `Config` from the environment. See
[Getting started → Configuration](../getting-started.md#configuration).
Each command calls it once in `run()` and passes values down; there is no
package-level global.
