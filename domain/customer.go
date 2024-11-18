package domain

type Customer struct {
	Id    string `gorm:"id" json:"id"`
	Name  string `gorm:"name" json:"name"`
	Email string `gorm:"email" json:"email"`
	Phone string `gorm:"phone" json:"phone"`
}
