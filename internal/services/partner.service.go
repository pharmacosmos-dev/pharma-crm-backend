package services

import (
	"context"
	"errors"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// SeedOAuthClients ensures the default OAuth clients exist in the database
// This should be called during application startup
func (s *Services) CreatePartnerOAuthClient(ctx context.Context, req *domain.OAuthClient) error {

	// Check if Partner client already exists
	var count int64
	err := s.db.WithContext(ctx).
		Model(&domain.OAuthClient{}).
		Where("client_id = ?", req.ClientId).
		Count(&count).Error

	if err != nil {
		s.log.Errorf("error checking OAuth client existence: %v", err)
		return err
	}

	if count > 0 {
		s.log.Info("Partner OAuth client already exists, skipping seed")
		return nil
	}

	// Hash the client secret
	hashedSecret, err := utils.HashClientSecret(s.cfg.Secret.UzumClientSecret)
	if err != nil {
		s.log.Errorf("failed to hash client secret: %v", err)
		return err
	}

	// Create the Partner client
	uzumClient := domain.OAuthClient{
		ClientId:      req.ClientId,
		ClientSecret:  hashedSecret,
		ClientName:    req.ClientName,
		AllowedScopes: req.AllowedScopes,
		IsActive:      true,
	}

	err = s.db.WithContext(ctx).Create(&uzumClient).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			s.log.Info("Partner OAuth client already exists (duplicate key), skipping")
			return nil
		}
		s.log.Errorf("failed to create OAuth client: %v", err)
		return err
	}

	s.log.Infof("Successfully created OAuth client for Partner (client_id: %s)", uzumClient.ClientId)
	return nil
}
