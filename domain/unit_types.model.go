package domain

import "time"

type UnitType struct {
	Id        string     `gorm:"id" json:"id"`
	UnitName  string     `gorm:"unit_name" json:"unit_name"`
	Codename  string     `gorm:"codename" json:"codename"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

type UnitTypeRequest struct {
	Id       string `gorm:"id" json:"-"`
	UnitName string `gorm:"unit_name" json:"unit_name"`
	Codename string `gorm:"codename" json:"codename"`
}

