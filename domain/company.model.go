package domain

import "time"

// Company structure
type Company struct {
	ID           string     `gorm:"id" json:"id"`
	Name         string     `gorm:"name" json:"name"`
	Email        string     `gorm:"email" json:"email"`
	Phone        string     `gorm:"phone" json:"phone"`
	IsFranchise  bool       `gorm:"is_franchise" json:"is_franchise"`
	Country      string     `gorm:"country" json:"country"`
	City         string     `gorm:"city" json:"city"`
	PostalCode   string     `gorm:"postal_code" json:"postal_code"`
	LegalName    string     `gorm:"legal_name" json:"legal_name"`
	LegalAddress string     `gorm:"legal_address" json:"legal_address"`
	CompanyInn   string     `gorm:"company_inn" json:"company_inn"`
	CompanyMfo   string     `gorm:"company_mfo" json:"company_mfo"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
}

// company request structure
type CompanyRequest struct {
	Name         string `gorm:"name" json:"name"`
	Email        string `gorm:"email" json:"email"`
	Phone        string `gorm:"phone" json:"phone"`
	IsFranchise  bool   `gorm:"is_franchise" json:"is_franchise"`
	Country      string `gorm:"country" json:"country"`
	City         string `gorm:"city" json:"city"`
	PostalCode   string `gorm:"postal_code" json:"postal_code"`
	LegalName    string `gorm:"legal_name" json:"legal_name"`
	LegalAddress string `gorm:"legal_address" json:"legal_address"`
	CompanyInn   string `gorm:"company_inn" json:"company_inn"`
	CompanyMfo   string `gorm:"company_mfo" json:"company_mfo"`
}

type CompanyStore struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsFranchise bool   `json:"is_franchise"`
}

type CompanyWithStores struct {
	ID          string         `json:"id"`
	Company     string         `json:"company"`
	IsFranchise bool           `json:"is_franchise"`
	Stores      []CompanyStore `json:"stores"`
}

type CompanyWithStoresResponse struct {
	PharmaCosmos CompanyWithStores   `json:"pharma_cosmos"`
	Franchises   []CompanyWithStores `json:"franchises"`
}
