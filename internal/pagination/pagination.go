// Package pagination provides the page envelopes every listing endpoint
// returns, for offset- and cursor-paginated results.
package pagination

import (
	"time"

	"github.com/google/uuid"
)

// OffsetPaginatedResult is one page of an offset-paginated listing.
type OffsetPaginatedResult[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalCount int64 `json:"total_count"`
	HasMore    bool  `json:"has_more"`
}

// NewOffsetPaginatedResult builds page (1-based) of a listing of totalCount
// items, limit per page. A nil items becomes an empty slice, so it encodes as [].
func NewOffsetPaginatedResult[T any](items []T, page, limit int, totalCount int64) OffsetPaginatedResult[T] {
	if items == nil {
		items = []T{}
	}

	seen := int64(max(page-1, 0)*limit) + int64(len(items))

	return OffsetPaginatedResult[T]{
		Items:      items,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		HasMore:    seen < totalCount,
	}
}

// Cursor locates the last item of a page. CreatedAt is set only for listings
// whose keyset includes a timestamp.
type Cursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

// CursorPaginatedResult is one page of a cursor-paginated listing. To fetch the
// next page, pass NextCursor (and NextCursorCreatedAt, when present) back as
// the request's cursor parameters. Both are null on the last page.
type CursorPaginatedResult[T any] struct {
	Items               []T        `json:"items"`
	Limit               int32      `json:"limit"`
	HasMore             bool       `json:"has_more"`
	NextCursor          *uuid.UUID `json:"next_cursor"`
	NextCursorCreatedAt *time.Time `json:"next_cursor_created_at,omitempty"`
}

// FetchLimit is how many rows to request for a page of limit items: one more,
// so NewCursorPaginatedResult can tell whether another page exists.
func FetchLimit(limit int32) int32 {
	return limit + 1
}

// NewCursorPaginatedResult builds a page from rows fetched with FetchLimit(limit).
// Rows beyond limit are dropped and set HasMore; cursorOf gives the cursor of
// the page's last item. A nil rows becomes an empty slice, so it encodes as [].
func NewCursorPaginatedResult[T any](rows []T, limit int32, cursorOf func(T) Cursor) CursorPaginatedResult[T] {
	page := CursorPaginatedResult[T]{Items: rows, Limit: limit}

	if int64(len(rows)) > int64(limit) {
		page.Items = rows[:limit]
		page.HasMore = true

		next := cursorOf(page.Items[len(page.Items)-1])
		page.NextCursor = &next.ID

		if !next.CreatedAt.IsZero() {
			page.NextCursorCreatedAt = &next.CreatedAt
		}
	}

	if page.Items == nil {
		page.Items = []T{}
	}

	return page
}
