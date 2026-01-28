package domain

// Main response struct for single store
type OsonAptekaRequest struct {
	Store       string `json:"store"`
	Code        string `json:"code"`
	RemainCount int    `json:"remain_count"`
	Programm    int    `json:"programm"`
	Drugs       []Drug `json:"drugs"`
}

// Drug struct
type Drug struct {
	ID           string  `json:"id"`
	DrugID       string  `json:"drug_id"`
	Barcode      string  `json:"barcode"`
	Name         string  `json:"name"`
	Manufacturer string  `json:"manufacturer"`
	Price        float64 `json:"price"`
	Qty          float64 `json:"qty"`
	ExpiryDate   string  `json:"expiry_date"`
}

type OsonAptekaRemainingQuantityResponse struct {
	Data struct {
		Store  string `json:"store"`
		Status int    `json:"status"`
		Error  string `json:"error"`
	} `json:"data"`
	Messages  []string `json:"messages"`
	Succeeded bool     `json:"succeeded"`
}
