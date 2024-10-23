package domain

import "time"

type Employee struct {
	Id        string     `json:"id" db:"id"`
	RoleId    string     `json:"role_id" db:"role_id"`
	FirstName string     `json:"first_name" db:"first_name"`
	LastName  string     `json:"last_name" db:"last_name"`
	Email     string     `json:"email" db:"email"`
	Phone     string     `json:"phone" db:"phone"`
	Password  string     `json:"password" db:"password"`
	CreatedAt *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
}
