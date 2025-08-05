package services

import (
	"errors"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// get customer list data
func (s *Services) ListCustomer(param *domain.QueryParam) ([]domain.Customer, int64, error) {
	var (
		res        []domain.Customer
		totalCount int64
	)

	// Start building the query
	query := s.db.
		Model(&domain.Customer{}).
		Preload("Store").
		Preload("Tag").
		Select(`
		customers.*,
		COALESCE(dc.barcode, '') AS discount_card,
		(SELECT created_at
		FROM sales
		WHERE sales.customer_id = customers.id
		ORDER BY sales.created_at DESC LIMIT 1)
		AS sale_date,
		COALESCE(SUM(sales.total_amount), 0) AS sale_amount`).
		Joins("LEFT JOIN sales ON sales.customer_id = customers.id").
		Joins("LEFT JOIN discount_cards dc ON dc.customer_id = customers.id").
		Where("customers.is_active = ?", true)

	if param.Search != "" {
		query = query.Where("dc.barcode = ?", param.Search)

	}
	if param.StoreID != "" {
		query = query.Where("customers.store_id = ?", param.StoreID)
	}
	err := query.
		Group("customers.id, dc.barcode").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Order("customers.created_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}

	return res, totalCount, nil
}

func (s *Services) ListDiscountCards(param *domain.QueryParam) ([]domain.DiscountCardWithCustomer, int64, error) {
	var (
		res        []domain.DiscountCardWithCustomer
		totalCount int64
	)

	query := s.db.
		Table("discount_cards").
		Select(`
			discount_cards.*,
			customers.full_name,
			customers.phone,
			customers.balance,
			customers.store_id,
			customers.tag_id,
			stores.name AS store_name,
			tags.name AS tag_name
		`).
		Joins("LEFT JOIN customers ON customers.id = discount_cards.customer_id").
		Joins("LEFT JOIN stores ON stores.id = customers.store_id").
		Joins("LEFT JOIN tags ON tags.id = customers.tag_id").
		Where("customers.deleted_at IS NULL AND discount_cards.deleted_at IS NULL")

	if param.Search != "" {
		query = query.Where(`
			customers.full_name ILIKE ? OR 
			customers.phone ILIKE ? OR
			discount_cards.barcode ILIKE ?`,
			"%"+param.Search+"%",
			"%"+param.Search+"%",
			"%"+param.Search+"%",
		)
	}

	if param.StoreID != "" {
		query = query.Where("customers.store_id = ?", param.StoreID)
	}

	err := query.
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Order("discount_cards.created_at DESC").
		Scan(&res).Error

	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}

	return res, totalCount, nil
}

// get or create customer by existing phone
func (s *Services) GetOrCreateCustomerByPhone(req *domain.NoorClientInfo) (*domain.Customer, error) {
	var customer *domain.Customer
	req.Phone = utils.NormalizePhoneNumber(req.Phone)

	err := s.db.First(&customer, "phone = ?", req.Phone).Error

	// If record found, return it
	if err == nil {
		return customer, nil
	}

	// If record not found, create new customer
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.CreateCustomerWithPhone(req)
	}

	// Any other database error
	s.log.Warn("ERROR on getting customer info: %v", err)
	return nil, err
}

// create new customer with phone and name
func (s *Services) CreateCustomerWithPhone(req *domain.NoorClientInfo) (*domain.Customer, error) {
	var res domain.Customer
	query := `
	INSERT INTO customers(
		first_name,
		full_name,
		phone
	)
	VALUES (?, ?, ?)
	RETURNING *
	`
	err := s.db.Raw(query, req.Name, req.Name, req.Phone).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return &res, err
	}
	return &res, nil
}
