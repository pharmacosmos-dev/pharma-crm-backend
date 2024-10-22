package storage

type Storage struct {
	ProductRepo
	CustomerRepo
	EmployeeRepo
}

func NewStorage() *Storage {
	return &Storage{}
}
