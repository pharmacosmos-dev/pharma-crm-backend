package services

import (
	"errors"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

func (s *Services) CreateOrUpdateRejectedProduct(req *domain.RejectedProductRequest) error {
	var existingID string

	if req.ProductID != nil {
		err := s.db.Table("rejected_products").
			Select("id").
			Where("product_id = ? AND store_id = ?", *req.ProductID, req.StoreID).
			Scan(&existingID).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	if existingID == "" && req.ProductID == nil && req.ProductName != nil {
		err := s.db.Table("rejected_products").
			Select("id").
			Where("product_name = ? AND store_id = ?", *req.ProductName, req.StoreID).
			Scan(&existingID).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	// CREATE
	if existingID == "" {
		req.RejectedTimes = 1
		if err := s.db.Table("rejected_products").Create(req).Error; err != nil {
			return err
		}
		return nil
	}

	// UPDATE: rejected_times + 1, updated_by set
	if err := s.db.Table("rejected_products").
		Where("id = ?", existingID).
		UpdateColumns(map[string]interface{}{
			"rejected_times": gorm.Expr("rejected_times + ?", 1),
			"updated_by":     req.CreatedBy,
			"updated_at":     gorm.Expr("NOW()"),
		}).Error; err != nil {
		return err
	}

	return nil
}
