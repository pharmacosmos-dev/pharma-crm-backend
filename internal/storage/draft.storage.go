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
