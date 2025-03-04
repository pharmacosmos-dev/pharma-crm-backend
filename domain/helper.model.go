package domain

import "time"

// Product Measurenment
type ProductMeasurement struct {
	ID         *string    `gorm:"id" json:"id"`
	MxikCode   string     `gorm:"mxik_code" json:"mxik_code"`
	MxikNameUz string     `gorm:"mxik_name_uz" json:"mxik_name_uz"`
	MxikNameRu string     `gorm:"mxik_name_ru" json:"mxik_name_ru"`
	UnitName   string     `gorm:"unit_name" json:"unit_name"`
	UnitCode   string     `gorm:"unit_code" json:"unit_code"`
	CreatedAt  *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"updated_at" json:"updated_at"`
}

type SoliqResponse struct {
	Success string              `json:"success"`
	Code    int                 `json:"code"`
	Reason  string              `json:"reason"`
	Data    []SoliqIKPUResponse `json:"data"`
}

type SoliqIKPUResponse struct {
	MxikCode string `json:"mxikCode"`
	Name     string `json:"name"`
	Units    string `json:"units"`
}
