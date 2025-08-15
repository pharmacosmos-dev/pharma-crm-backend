package services

import (
	"github.com/pharma-crm-backend/domain"
)

func (s *Services) CreateRejectedProduct(req *domain.RejectedProductRequest) error {

	// insert new rejected product
	if err := s.db.Table("rejected_products").Create(req).Error; err != nil {
		s.log.Error("failed to create rejected product:", err)
		return err
	}

	return nil
}
