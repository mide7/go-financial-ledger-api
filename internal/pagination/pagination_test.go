package pagination

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewOffsetPaginatedResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		items       []int
		page, limit int
		totalCount  int64
		wantHasMore bool
	}{
		{name: "first page of many", items: make([]int, 10), page: 1, limit: 10, totalCount: 25, wantHasMore: true},
		{name: "last partial page", items: make([]int, 5), page: 3, limit: 10, totalCount: 25},
		{name: "exact final page", items: make([]int, 10), page: 2, limit: 10, totalCount: 20},
		{name: "past the end", items: nil, page: 9, limit: 10, totalCount: 20},
		{name: "empty result", items: nil, page: 1, limit: 10, totalCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewOffsetPaginatedResult(tt.items, tt.page, tt.limit, tt.totalCount)
			if got.HasMore != tt.wantHasMore {
				t.Errorf("HasMore = %v, want %v", got.HasMore, tt.wantHasMore)
			}

			if got.Items == nil {
				t.Error("Items is nil; want an empty slice so it encodes as []")
			}
		})
	}
}

func TestNewCursorPaginatedResult(t *testing.T) {
	t.Parallel()

	ids := make([]uuid.UUID, 4)
	for i := range ids {
		ids[i] = uuid.New()
	}

	at := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	byID := func(id uuid.UUID) Cursor { return Cursor{ID: id} }
	withTime := func(id uuid.UUID) Cursor { return Cursor{ID: id, CreatedAt: at} }

	tests := []struct {
		name      string
		rows      []uuid.UUID
		limit     int32
		cursorOf  func(uuid.UUID) Cursor
		wantItems int
		wantNext  *uuid.UUID
		wantTime  bool
	}{
		{name: "extra row means more", rows: ids, limit: 3, cursorOf: byID, wantItems: 3, wantNext: &ids[2]},
		{name: "exactly limit is the last page", rows: ids[:3], limit: 3, cursorOf: byID, wantItems: 3},
		{name: "short page is the last page", rows: ids[:1], limit: 3, cursorOf: byID, wantItems: 1},
		{name: "empty", rows: nil, limit: 3, cursorOf: byID, wantItems: 0},
		{name: "timestamp cursor", rows: ids, limit: 2, cursorOf: withTime, wantItems: 2, wantNext: &ids[1], wantTime: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewCursorPaginatedResult(tt.rows, tt.limit, tt.cursorOf)

			if len(got.Items) != tt.wantItems || got.Items == nil {
				t.Fatalf("Items = %v, want %d (non-nil)", got.Items, tt.wantItems)
			}

			if got.HasMore != (tt.wantNext != nil) {
				t.Errorf("HasMore = %v, want %v", got.HasMore, tt.wantNext != nil)
			}

			switch {
			case tt.wantNext == nil && got.NextCursor != nil:
				t.Errorf("NextCursor = %v, want nil", *got.NextCursor)
			case tt.wantNext != nil && (got.NextCursor == nil || *got.NextCursor != *tt.wantNext):
				t.Errorf("NextCursor = %v, want %v", got.NextCursor, *tt.wantNext)
			}

			if (got.NextCursorCreatedAt != nil) != tt.wantTime {
				t.Errorf("NextCursorCreatedAt = %v, want set: %v", got.NextCursorCreatedAt, tt.wantTime)
			}
		})
	}
}

func TestFetchLimit(t *testing.T) {
	t.Parallel()

	if got := FetchLimit(20); got != 21 {
		t.Errorf("FetchLimit(20) = %d, want 21", got)
	}
}
