package domain

import "time"

type Store struct {
	Id        string     `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	Location  string     `json:"location" db:"location"`
	CreatedAt *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
}
