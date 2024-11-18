package domain

import "time"

type Login struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token       string                 `json:"token"`
	Employee    Employee               `json:"employee"`
	Permissions map[string]interface{} `json:"permissions"`
}

type Employee struct {
	Id        string     `gorm:"id" json:"id" db:"id"`
	RoleId    string     `gorm:"role_id" json:"role_id" db:"role_id"`
	FirstName string     `gorm:"first_name" json:"first_name" db:"first_name"`
	LastName  string     `gorm:"last_name" json:"last_name" db:"last_name"`
	Email     string     `gorm:"email" json:"email" db:"email"`
	Phone     string     `gorm:"phone" json:"phone" db:"phone"`
	Password  string     `gorm:"password" json:"password" db:"password"`
	Language  string     `gorm:"language" json:"language" db:"language"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at" db:"updated_at"`
}
