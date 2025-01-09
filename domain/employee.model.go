package domain

import "time"

type Login struct {
	Phone    string `json:"phone" example:"+998944444444"`
	Password string `json:"password" example:"12345678"`
}

type LoginResponse struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	Employee     Employee `json:"employee"`
}

type Employee struct {
	Id        string     `gorm:"id" json:"id"`
	StoreId   string     `gorm:"store_id" json:"store_id"`
	PublicId  int        `gorm:"public_id" json:"public_id"`
	FirstName string     `gorm:"first_name" json:"first_name"`
	LastName  string     `gorm:"last_name" json:"last_name"`
	Email     string     `gorm:"email" json:"email"`
	Phone     string     `gorm:"phone" json:"phone"`
	Password  string     `gorm:"password" json:"password"`
	Language  string     `gorm:"language" json:"language"`
	Gender    string     `gorm:"gender" json:"gender"`
	Status    string     `gorm:"status" json:"status"`
	Birthdate string     `gorm:"birthdate" json:"birthdate"`
	Photo     string     `gorm:"photo" json:"photo"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
	Store     *Store     `gorm:"foreignKey:StoreId" json:"store"`
	Roles     []Role     `gorm:"many2many:employee_roles;" json:"roles"` // Many-to-Many relationship
}

type EmployeeRequest struct {
	Id        string   `gorm:"id" json:"-"`
	RoleIds   []string `gorm:"-" json:"role_ids"`
	StoreId   string   `gorm:"store_id" json:"store_id"`
	PublicId  int      `gorm:"public_id" json:"-"`
	FirstName string   `gorm:"first_name" json:"first_name"`
	LastName  string   `gorm:"last_name" json:"last_name"`
	Phone     string   `gorm:"phone" json:"phone"`
	Gender    string   `gorm:"gender" json:"gender"`
	Status    string   `gorm:"status" json:"-"`
	Password  *string  `gorm:"password" json:"password"`
	Language  string   `gorm:"language" json:"language"`
	Birthdate string   `gorm:"birthdate" json:"birthdate"`
}

// Reset password request
type ResetPasswordRequest struct {
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// Employee update info request
type EmployeeUpdateInfoRequest struct {
	FirstName string `gorm:"first_name" json:"first_name"`
	LastName  string `gorm:"last_name" json:"last_name"`
	Photo     string `gorm:"photo" json:"photo"`
	Language  string `gorm:"language" json:"language"`
}

type EmployeeRole struct {
	Id         string     `gorm:"id" json:"id"`
	RoleId     string     `gorm:"role_id" json:"role_id"`
	EmployeeId string     `gorm:"employee_id" json:"employee_id"`
	CreatedAt  *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"updated_at" json:"updated_at"`
}
