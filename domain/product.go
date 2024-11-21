package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type StringSlice []string

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return errors.New("unsupported type")
	}
}

func (s StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Product
type Product struct {
	Id                 string      `gorm:"id" json:"id" db:"id"`
	StoreId            string      `gorm:"-" json:"store_id" db:"store_id"`
	CategoryId         string      `gorm:"category_id" json:"category_id" db:"category_id"`
	BrandId            string      `gorm:"-" json:"brand_id" db:"brand_id"`
	SupplierId         string      `gorm:"-" json:"supplier_id" db:"supplier_id"`
	UnitId             string      `gorm:"-" json:"unit_id" db:"unit_id"`
	ProductType        string      `gorm:"product_type" json:"product_type" db:"product_type"`
	ProductVariability string      `gorm:"product_variability" json:"product_variability" db:"product_variability"`
	Name               string      `gorm:"name" json:"name" db:"name"`
	Sku                string      `gorm:"sku" json:"sku" db:"sku"`
	Barcode            string      `gorm:"barcode" json:"barcode" db:"barcode"`
	MainPhoto          string      `gorm:"main_photo" json:"main_photo" db:"main_photo"`
	Photos             StringSlice `gorm:"-" json:"photos" db:"photos"`
	SupplyPrice        float64     `gorm:"supply_price" json:"supply_price" db:"supply_price"`
	Markup             int         `gorm:"markup" json:"markup" db:"markup"`
	RetailPrice        float64     `gorm:"retail_price" json:"retail_price" db:"retail_price"`
	Quantity           int         `gorm:"quantity" json:"quantity" db:"quantity"`
	Sum                float64     `gorm:"sum" json:"sum" db:"sum"`
	Description        string      `gorm:"description" json:"description" db:"description"`
	Status             string      `gorm:"status" json:"status" db:"status"`
	Manufacturer       string      `gorm:"manufacturer" json:"manufacturer" db:"manufacturer"`
	ExpireDate         string      `gorm:"expire_date" json:"expire_date" db:"expire_date"`
	Category           Category    `gorm:"foreignKey:CategoryId" json:"category" db:"category"`
	CreatedAt          *time.Time  `gorm:"created_at" json:"created_at" db:"created_at"`
	UpdatedAt          *time.Time  `gorm:"updated_at" json:"updated_at" db:"updated_at"`
}
