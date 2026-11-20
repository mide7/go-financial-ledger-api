package pagination

import "testing"

func TestNewOffsetPaginatedResult_HasMore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		items      int
		limit      int64
		page       int64
		totalCount int64
		want       bool
	}{
		{name: "first page of many", items: 10, limit: 10, page: 1, totalCount: 25, want: true},
		{name: "last partial page", items: 5, limit: 10, page: 3, totalCount: 25, want: false},
		{name: "exact final page", items: 10, limit: 10, page: 2, totalCount: 20, want: false},
		{name: "empty result", items: 0, limit: 10, page: 1, totalCount: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewOffsetPaginatedResult(make([]int, tt.items), tt.limit, tt.page, tt.totalCount)
			if got.HasMore != tt.want {
				t.Errorf("HasMore = %v, want %v", got.HasMore, tt.want)
			}
		})
	}
}

func TestNewCursorPaginatedResult_HasMore(t *testing.T) {
	t.Parallel()

	if got := NewCursorPaginatedResult(make([]int, 10), 10, nil, nil); !got.HasMore {
		t.Error("full page: HasMore = false, want true")
	}

	if got := NewCursorPaginatedResult(make([]int, 3), 10, nil, nil); got.HasMore {
		t.Error("partial page: HasMore = true, want false")
	}
}
