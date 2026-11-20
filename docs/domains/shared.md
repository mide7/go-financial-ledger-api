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

[pagination.go](../../internal/pagination/pagination.go) defines generic
response envelopes for paginated lists:

- `OffsetPaginatedResult[T]`: `items`, `limit`, `page` (1-based),
  `total_count`, `has_more`.
- `CursorPaginatedResult[T]`: `items`, `limit`, `next_cursor_id`,
  `next_cursor_time`, `has_more`. `has_more` is true when the page is full, so
  a final page of exactly `limit` items is followed by one empty page.

Endpoints don't use these yet; they return plain arrays.

## `internal/config`

`config.Load()` builds a `Config` from the environment. See
[Getting started → Configuration](../getting-started.md#configuration).
Each command calls it once in `run()` and passes values down; there is no
package-level global.
