package domain

import (
	"time"

	"github.com/pharma-crm-backend/pkg/utils"
)

type Customer struct {
	Id        string            `gorm:"id" json:"id"`
	FirstName string            `gorm:"first_name" json:"first_name"`
	LastName  string            `gorm:"last_name" json:"last_name"`
	Phone     utils.StringArray `gorm:"type:text[]" json:"phone"`
	Birthday  string            `gorm:"birthday" json:"birthday" example:"2006-01-02"`
	Gender    string            `gorm:"gender" json:"gender" example:"male/female"`
	Balance   float64           `gorm:"balance" json:"balance"`
	TagId     string            `gorm:"-" json:"tag_id"`
	// Email          string     `gorm:"email" json:"email"`
	// MaritalStatus  string     `gorm:"marital_status" json:"marital_status"`
	// PrimaryLang    string     `gorm:"primary_lang" json:"primary_lang"`
	// TgUsername     string     `gorm:"tg_username" json:"tg_username"`
	// Facebook       string     `gorm:"facebook" json:"facebook"`
	// Instagram      string     `gorm:"instagram" json:"instagram"`
	// IsSmsNotify    bool       `gorm:"is_sms_notify" json:"is_sms_notify"`
	// IsPhoneNotify  bool       `gorm:"is_phone_notify" json:"is_phone_notify"`
	// IsSocialNotify bool       `gorm:"is_social_notify" json:"is_social_notify"`
	// IsEmailNotify  bool       `gorm:"is_email_notify" json:"is_email_notify"`
	Tag       *Tag       `gorm:"foreignKey:TagId" json:"tag"`
	CreatedBy string     `gorm:"-" json:"created_by"`
	UpdatedBy string     `gorm:"-" json:"updated_by"`
	DeletedBy string     `gorm:"-" json:"deleted_by"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

type CustomerRequest struct {
	Id        string            `gorm:"id" json:"-"`
	FirstName string            `gorm:"first_name" json:"first_name"`
	LastName  string            `gorm:"last_name" json:"last_name"`
	Phone     utils.StringArray `gorm:"type:text[]" json:"phone"`
	Birthday  string            `gorm:"birthday" json:"birthday" example:"2006-01-02"`
	Gender    string            `gorm:"gender" json:"gender" example:"male/female"`
	TagId     string            `gorm:"-" json:"tag_id"`
	CreatedBy string            `gorm:"-" json:"created_by"`
	// Email          string            `gorm:"email" json:"email"`
	// MaritalStatus  string            `gorm:"marital_status" json:"marital_status"`
	// PrimaryLang    string            `gorm:"primary_lang" json:"primary_lang"`
	// TgUsername     string            `gorm:"tg_username" json:"tg_username"`
	// Facebook       string            `gorm:"facebook" json:"facebook"`
	// Instagram      string            `gorm:"instagram" json:"instagram"`
	// IsSmsNotify    bool              `gorm:"is_sms_notify" json:"is_sms_notify"`
	// IsPhoneNotify  bool              `gorm:"is_phone_notify" json:"is_phone_notify"`
	// IsSocialNotify bool              `gorm:"is_social_notify" json:"is_social_notify"`
	// IsEmailNotify  bool              `gorm:"is_email_notify" json:"is_email_notify"`
}

type Tag struct {
	Id   string `gorm:"id" json:"id"`
	Name string `gorm:"name" json:"name"`
}
