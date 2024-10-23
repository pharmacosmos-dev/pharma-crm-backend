package domain

type Unit struct {
	Id           string `json:"id" db:"id"`
	Unit         string `json:"unit" db:"unit"`
	Abbreviation string `json:"abbreviation" db:"abbreviation"`
	Accuracy     string `json:"accuracy" db:"accuracy"`
}
