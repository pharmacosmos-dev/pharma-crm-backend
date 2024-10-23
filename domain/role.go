package domain

import "time"

type Role struct {
	Id        string     `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	Desc      string     `json:"description" db:"description"`
	CreatedAt *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
}
