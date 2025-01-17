package utils

type Meta struct {
	TotalCount  int64 `json:"total_count"`
	PageCount   int   `json:"page_count"`
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
}

// ListResponse constructs a paginated response with metadata.
func ListResponse[T any](items []T, totalCount int64, limit, offset int) map[string]interface{} {
	return map[string]interface{}{
		"_meta": Meta{
			TotalCount:  totalCount,
			PerPage:     limit,
			CurrentPage: (offset / limit) + 1,
			PageCount:   int((totalCount + int64(limit) - 1) / int64(limit)),
		},
		"data": items,
	}
}
