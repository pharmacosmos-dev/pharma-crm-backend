package domain

import "time"

// Company structure
type Company struct {
	ID           string     `gorm:"id" json:"id"`
	Name         string     `gorm:"name" json:"name"`
	Email        string     `gorm:"email" json:"email"`
	Phone        string     `gorm:"phone" json:"phone"`
	Country      string     `gorm:"country" json:"country"`
	City         string     `gorm:"city" json:"city"`
	PostalCode   string     `gorm:"postal_code" json:"postal_code"`
	LegalName    string     `gorm:"legal_name" json:"legal_name"`
	LegalAddress string     `gorm:"legal_address" json:"legal_address"`
	CompanyInn   string     `gorm:"company_inn" json:"company_inn"`
	CompanyStir  string     `gorm:"company_stir" json:"company_stir"`
	CompanyMfo   string     `gorm:"company_mfo" json:"company_mfo"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
}

// company request structure
type CompanyRequest struct {
	Name         string `gorm:"name" json:"name"`
	Email        string `gorm:"email" json:"email"`
	Phone        string `gorm:"phone" json:"phone"`
	Country      string `gorm:"country" json:"country"`
	City         string `gorm:"city" json:"city"`
	PostalCode   string `gorm:"postal_code" json:"postal_code"`
	LegalName    string `gorm:"legal_name" json:"legal_name"`
	LegalAddress string `gorm:"legal_address" json:"legal_address"`
	CompanyInn   string `gorm:"company_inn" json:"company_inn"`
	CompanyStir  string `gorm:"company_stir" json:"company_stir"`
	CompanyMfo   string `gorm:"company_mfo" json:"company_mfo"`
}
