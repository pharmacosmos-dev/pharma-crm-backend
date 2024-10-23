package domain

import "time"

type Store struct {
	ID        string     `json:"id" db:"id"`                 // UUID of the store
	Name      string     `json:"name" db:"name"`             // Name of the store
	Location  string     `json:"location" db:"location"`     // Location of the store
	CreatedAt *time.Time `json:"created_at" db:"created_at"` // Creation timestamp
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"` // Update timestamp
}
