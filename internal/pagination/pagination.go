package pagination

type OffsetPaginatedResult[T any] struct {
	Items      []T   `json:"items"`
	Limit      int64 `json:"limit"`
	Page       int64 `json:"page"`
	TotalCount int64 `json:"total_count"`
	HasMore    bool  `json:"has_more"`
}

func NewOffsetPaginatedResult[T any](items []T, limit, page, totalCount int64) OffsetPaginatedResult[T] {
	offset := (page - 1) * limit

	return OffsetPaginatedResult[T]{
		Items:      items,
		Limit:      limit,
		Page:       page,
		TotalCount: totalCount,
		HasMore:    offset+int64(len(items)) < totalCount,
	}
}

type CursorPaginatedResult[T any] struct {
	Items          []T     `json:"items"`
	Limit          int64   `json:"limit"`
	NextCursorID   *string `json:"next_cursor_id"`
	NextCursorTime *string `json:"next_cursor_time"`
	HasMore        bool    `json:"has_more"`
}

func NewCursorPaginatedResult[T any](items []T, limit int64, nextCursorID, nextCursorTime *string) CursorPaginatedResult[T] {
	return CursorPaginatedResult[T]{
		Items:          items,
		Limit:          limit,
		NextCursorID:   nextCursorID,
		NextCursorTime: nextCursorTime,
		HasMore:        len(items) == int(limit),
	}
}
