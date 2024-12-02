package domain

import (
	"time"

	"github.com/pharma-crm-backend/pkg/utils"
)

type Customer struct {
	Id             string            `gorm:"id" json:"id"`
	StoreId        string            `gorm:"store_id" json:"store_id"`
	FirstName      string            `gorm:"first_name" json:"first_name"`
	LastName       string            `gorm:"last_name" json:"last_name"`
	MiddleName     string            `gorm:"middle_name" json:"middle_name"`
	Phone          utils.StringArray `gorm:"type:text[]" json:"phone"`
	Birthday       string            `gorm:"birthday" json:"birthday"`
	Gender         string            `gorm:"gender" json:"gender"`
	Email          string            `gorm:"email" json:"email"`
	Balance        float64           `gorm:"balance" json:"balance"`
	MaritalStatus  string            `gorm:"marital_status" json:"marital_status"`
	PrimaryLang    string            `gorm:"primary_lang" json:"primary_lang"`
	TgUsername     string            `gorm:"tg_username" json:"tg_username"`
	Facebook       string            `gorm:"facebook" json:"facebook"`
	Instagram      string            `gorm:"instagram" json:"instagram"`
	IsSmsNotify    bool              `gorm:"is_sms_notify" json:"is_sms_notify"`
	IsPhoneNotify  bool              `gorm:"is_phone_notify" json:"is_phone_notify"`
	IsSocialNotify bool              `gorm:"is_social_notify" json:"is_social_notify"`
	IsEmailNotify  bool              `gorm:"is_email_notify" json:"is_email_notify"`
	CreatedAt      *time.Time        `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time        `gorm:"updated_at" json:"updated_at"`
}

type CustomerRequest struct {
	Id             string            `gorm:"id" json:"-"`
	StoreId        string            `gorm:"store_id" json:"store_id"`
	FirstName      string            `gorm:"first_name" json:"first_name"`
	LastName       string            `gorm:"last_name" json:"last_name"`
	MiddleName     string            `gorm:"middle_name" json:"middle_name"`
	Phone          utils.StringArray `gorm:"type:text[]" json:"phone"`
	Birthday       string            `gorm:"birthday" json:"birthday"`
	Gender         string            `gorm:"gender" json:"gender"`
	Email          string            `gorm:"email" json:"email"`
	MaritalStatus  string            `gorm:"marital_status" json:"marital_status"`
	PrimaryLang    string            `gorm:"primary_lang" json:"primary_lang"`
	TgUsername     string            `gorm:"tg_username" json:"tg_username"`
	Facebook       string            `gorm:"facebook" json:"facebook"`
	Instagram      string            `gorm:"instagram" json:"instagram"`
	IsSmsNotify    bool              `gorm:"is_sms_notify" json:"is_sms_notify"`
	IsPhoneNotify  bool              `gorm:"is_phone_notify" json:"is_phone_notify"`
	IsSocialNotify bool              `gorm:"is_social_notify" json:"is_social_notify"`
	IsEmailNotify  bool              `gorm:"is_email_notify" json:"is_email_notify"`
}
