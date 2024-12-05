package domain

type Supplier struct {
	Id        string `gorm:"id" json:"id" db:"id"`
	FirstName string `gorm:"first_name" json:"first_name" db:"first_name"`
	LastName  string `gorm:"last_name" json:"last_name" db:"last_name"`
}
