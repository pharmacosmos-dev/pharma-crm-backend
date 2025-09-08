package services

import (
	"time"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

func (s *Services) SaveAsilBelgiToken(db *gorm.DB, req *domain.AsilBelgiTokenRequest) error {
	// deactivate old tokens
	if err := db.Model(&domain.AsilBelgiToken{}).
		Where("is_active = true").
		Update("is_active", false).Error; err != nil {
		return err
	}

	// parse expires_at if given
	var expiresAt time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			expiresAt = t
		}
	} else {
		expiresAt = time.Now().Add(time.Hour * 10)
	}

	// insert new token
	token := domain.AsilBelgiToken{
		Token:     req.Token,
		ExpiresAt: expiresAt,
		IsActive:  true,
	}
	return db.Create(&token).Error
}
