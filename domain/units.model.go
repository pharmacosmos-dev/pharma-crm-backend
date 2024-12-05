package domain

type Unit struct {
	Id           string `gorm:"id" json:"id" db:"id"`
	Unit         string `gorm:"unit" json:"unit" db:"unit"`
	Abbreviation string `gorm:"abbreviation" json:"abbreviation" db:"abbreviation"`
	Accuracy     string `gorm:"accuracy" json:"accuracy" db:"accuracy"`
}
