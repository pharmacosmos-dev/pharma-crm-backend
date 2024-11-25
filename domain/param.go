package domain

type Params struct {
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
	Order  map[string]interface{} `json:"order"`
}


