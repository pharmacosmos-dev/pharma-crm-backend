package services

import (
	"github.com/pharma-crm-backend/domain"
	"time"
)

func (s *Services) UpdateDiscountCard(req *domain.UpdateDiscountCardRequest) error {
	var card = domain.UpdateDiscountCard{
		Percent:   req.Percent,
		UpdatedBy: req.UpdatedBy,
		UpdatedAt: time.Now(),
	}

	if req.CustomerID != nil {
		card.CustomerID = req.CustomerID
	}

	return s.db.
		Model(&domain.DiscountCard{}).
		Where("id = ? AND deleted_at IS NULL", req.ID).
		Updates(card).
		Error
}

func (s *Services) DeleteDiscountCard(id, deletedBy string) error {
	return s.db.
		Model(&domain.DiscountCard{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"deleted_by": deletedBy,
		}).
		Error
}
