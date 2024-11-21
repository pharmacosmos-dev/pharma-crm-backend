package domain

type Params struct {
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
	Order  map[string]interface{} `json:"order"`
}

type Meta struct {
	TotalCount  int `json:"total_count"`
	PageCount   int `json:"page_count"`
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
}
