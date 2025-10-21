package domain

import "time"

type Login struct {
	Phone    string `json:"phone" validate:"required,e164"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	Employee     Employee `json:"employee"`
}

type EmployeeClaims struct {
	UserId    string `json:"user_id"`
	CompanyId string `json:"company_id"`
	StoreId   string `json:"store_id"`
	Role      string `json:"role"`
}

type Employee struct {
	Id         string           `gorm:"id" json:"id"`
	CompanyId  string           `gorm:"company_id" json:"company_id"`
	StoreId    string           `gorm:"store_id" json:"store_id"`
	PublicId   int              `gorm:"public_id" json:"public_id"`
	Position   string           `gorm:"position" json:"position"`
	FirstName  string           `gorm:"first_name" json:"first_name"`
	LastName   string           `gorm:"last_name" json:"last_name"`
	FullName   string           `gorm:"full_name" json:"full_name"`
	Email      string           `gorm:"email" json:"email"`
	Phone      string           `gorm:"phone" json:"phone"`
	Password   string           `gorm:"password" json:"password"`
	Language   string           `gorm:"language" json:"language"`
	Gender     string           `gorm:"gender" json:"gender"`
	Status     string           `gorm:"status" json:"status"`
	Birthdate  string           `gorm:"birthdate" json:"birthdate"`
	Photo      string           `gorm:"photo" json:"photo"`
	RoleType   string           `gorm:"role_type" json:"role_type,omitempty"`
	CreatedAt  *time.Time       `gorm:"created_at" json:"created_at"`
	UpdatedAt  *time.Time       `gorm:"updated_at" json:"updated_at"`
	Store      *Store           `gorm:"foreignKey:StoreId" json:"store"`
	Company    *Company         `gorm:"foreignKey:CompanyId" json:"company"`
	Permission []Permission     `gorm:"-" json:"permissions"`
	Roles      []Role           `gorm:"many2many:employee_roles;" json:"roles"`
	Cashbox    *EmployeeCashbox `gorm:"-" json:"cashbox"`
}

type EmployeeRequest struct {
	Id        string   `gorm:"id" json:"-"`
	RoleIds   []string `gorm:"-" json:"role_ids"`
	CompanyId string   `gorm:"company_id" json:"company_id"`
	StoreId   *string  `gorm:"store_id" json:"store_id"`
	Position  string   `gorm:"position" json:"position"`
	FirstName string   `gorm:"first_name" json:"first_name"`
	LastName  string   `gorm:"last_name" json:"last_name"`
	FullName  string   `gorm:"full_name" json:"full_name"`
	Phone     string   `gorm:"phone" json:"phone" validate:"required,e164"`
	Gender    string   `gorm:"gender" json:"gender" validate:"required,oneof=male female"`
	Status    string   `gorm:"status" json:"-"`
	Password  *string  `gorm:"password" json:"password"`
	Language  string   `gorm:"language" json:"language" validate:"required,oneof=uz en ru"`
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
	Position  string `gorm:"position" json:"position"`
	Language  string `gorm:"language" json:"language" validate:"required,oneof=uz en ru"`
}

type EmployeeRole struct {
	Id         string       `gorm:"id" json:"id"`
	RoleId     string       `gorm:"role_id" json:"role_id"`
	EmployeeId string       `gorm:"employee_id" json:"employee_id"`
	CreatedAt  *time.Time   `gorm:"created_at" json:"created_at"`
	UpdatedAt  *time.Time   `gorm:"updated_at" json:"updated_at"`
	Permission []Permission `gorm:"many2many:role_permissions;" json:"permissions"`
}

// Employee bonus
type EmployeeBonus struct {
	Id                 string     `gorm:"id" json:"id"`
	EmployeeId         string     `gorm:"employee_id" json:"employee_id"`
	SaleId             string     `gorm:"sale_id" json:"sale_id"`
	ProductId          string     `gorm:"product_id" json:"product_id"`
	CashboxOperationId string     `gorm:"cashbox_operation_id" json:"cashbox_operation_id"`
	BonusAmount        float64    `gorm:"bonus_amount" json:"bonus_amount"`
	Quantity           int        `gorm:"quantity" json:"quantity"`
	UnitQuantity       int        `gorm:"unit_quantity" json:"unit_quantity"`
	CreatedAt          *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time `gorm:"updated_at" json:"updated_at"`
}

// Employee bonus create structure
type EmployeeBonusRequest struct {
	EmployeeId         string  `gorm:"employee_id" json:"employee_id"`
	SaleId             string  `gorm:"sale_id" json:"sale_id"`
	ProductId          string  `gorm:"product_id" json:"product_id"`
	CashboxOperationId string  `gorm:"cashbox_operation_id" json:"cashbox_operation_id"`
	BonusAmount        float64 `gorm:"bonus_amount" json:"bonus_amount"`
	Quantity           int     `gorm:"quantity" json:"quantity"`
	UnitQuantity       int     `gorm:"unit_quantity" json:"unit_quantity"`
}

type EmployeePreload struct {
	Id        string `gorm:"id" json:"id"`
	StoreId   string `gorm:"store_id" json:"store_id"`
	PublicId  int    `gorm:"public_id" json:"public_id"`
	FirstName string `gorm:"first_name" json:"first_name"`
	LastName  string `gorm:"last_name" json:"last_name"`
	FullName  string `gorm:"full_name" json:"full_name"`
}

type EmployeeForSale struct {
	Id        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
}
