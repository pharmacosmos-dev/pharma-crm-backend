package domain

type Role struct {
	Id   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
	Desc string `json:"desc" db:"desc"`
}
