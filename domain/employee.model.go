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
	Id        string     `gorm:"id" json:"id"`
	StoreId   string     `gorm:"store_id" json:"store_id"`
	RoleId    string     `gorm:"role_id" json:"role_id"`
	FirstName string     `gorm:"first_name" json:"first_name"`
	LastName  string     `gorm:"last_name" json:"last_name"`
	Email     string     `gorm:"email" json:"email"`
	Phone     string     `gorm:"phone" json:"phone"`
	Password  string     `gorm:"password" json:"password"`
	Language  string     `gorm:"language" json:"language"`
	Photo     string     `gorm:"photo" json:"photo"`
	Store     *Store     `gorm:"foreignKey:StoreId" json:"store"`
	Role      *Role      `gorm:"foreignKey:RoleId" json:"role"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

type EmployeeRequest struct {
	Id        string `gorm:"id" json:"-"`
	RoleId    string `gorm:"role_id" json:"role_id"`
	StoreId   string `gorm:"store_id" json:"store_id"`
	FirstName string `gorm:"first_name" json:"first_name"`
	LastName  string `gorm:"last_name" json:"last_name"`
	Photo     string `gorm:"photo" json:"photo"`
	Email     string `gorm:"email" json:"email"`
	Phone     string `gorm:"phone" json:"phone"`
	Password  string `gorm:"password" json:"password"`
	Language  string `gorm:"language" json:"language"`
}
