package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ReservedProduct struct {
	Id           string     `json:"id" gorm:"column:id;primaryKey"`
	SortIndex    int        `json:"sort_index" gorm:"column:sort_index"`
	Name         string     `json:"name" gorm:"column:name"`
	MaterialCode string     `json:"material_code" gorm:"column:material_code"`
	IsActive     bool       `json:"is_active" gorm:"column:is_active"`
	CreatedAt    *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    *time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (ReservedProduct) TableName() string {
	return "reserved_products"
}

// MaterialCode — 1C material_code'ni ham satr ("ABC123"), ham son (11001) ko'rinishida
// yuborishi mumkin: loyihadagi 1C endpointlarining bir qismi int (product1c), boshqasi
// string (uzumtezkor) ishlatadi. Har kuni 05:00 da keladigan import formatning shu mayda
// farqi tufayli 400 bilan yiqilmasligi uchun ikkalasi ham qabul qilinadi va satr sifatida
// saqlanadi.
type MaterialCode string

func (m MaterialCode) String() string {
	return string(m)
}

func (m *MaterialCode) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "null" {
		*m = ""
		return nil
	}

	if strings.HasPrefix(raw, `"`) {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		*m = MaterialCode(strings.TrimSpace(str))
		return nil
	}

	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("invalid material_code: %s", raw)
	}
	*m = MaterialCode(num.String())

	return nil
}

// ReservedProductImportItem — 1C yuboradigan bitta qator.
type ReservedProductImportItem struct {
	Index        int          `json:"index" example:"1"`
	ProductName  string       `json:"product_name" example:"Product A"`
	MaterialCode MaterialCode `json:"material_code" swaggertype:"string" example:"MAT001"`
}


type ReservedProductImportRequest struct {
	Products []ReservedProductImportItem `json:"products" binding:"required,min=1"`
}


type ReservedProductImportResult struct {
	TotalReceived    int `json:"total_received" example:"9000"`
	Inserted         int `json:"inserted" example:"100"`
	Updated          int `json:"updated" example:"8900"`
	Deactivated      int `json:"deactivated" example:"2000"`
	SkippedDuplicate int `json:"skipped_duplicate" example:"0"`
}

type ReservedProductQueryParams struct {
	Search   string `form:"search"`    // name yoki material_code bo'yicha
	IsActive *bool  `form:"is_active"` // berilmasa — hammasi
	Limit    int    `form:"limit"`
	Offset   int    `form:"offset"`
}
