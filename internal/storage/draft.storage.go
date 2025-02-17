package storage

import (
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

func (s *Storage) UpdateDraftField(tx *gorm.DB, field string, value string, idField, idValue string) (*domain.Draft, error) {
	var res domain.Draft
	err := tx.Raw(`UPDATE drafts SET `+field+` = ? WHERE `+idField+` = ? RETURNING *`, value, idValue).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// get draft by id -> it calculate the total price of the draft
func (s *Storage) GetDraftByID(id string) (*domain.Draft, error) {
	var res domain.Draft
	err := s.db.
		Preload("Customer").
		Preload("Store").
		Preload("Employee").
		Select("drafts.*, SUM(ci.total_price) AS total_price").
		Joins("JOIN cart_item_drafts cid ON cid.draft_id = drafts.id").
		Joins("JOIN cart_items ci ON ci.id = cid.cart_item_id").
		Group(`drafts.id, drafts.product_id, drafts.cash_box_id, drafts.store_id, drafts.description,
			drafts.draft_time, drafts.created_at, drafts.updated_at, drafts.created_by,
			drafts.is_active, drafts.sale_id,
			drafts.customer_id, drafts.draft_number, drafts.status`).
		Debug().
		First(&res, "drafts.id = ?", id).Error

	if err != nil {
		return nil, err
	}
	return &res, nil
}
