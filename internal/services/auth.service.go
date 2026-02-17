package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

func (s *Services) OAuthToken(ctx context.Context, req *domain.OAuthRequest) (*domain.OAuthResponse, error) {
	// Validate grant type
	if req.GrantType != "client_credentials" {
		return nil, errors.New("unsupported grant type, only 'client_credentials' is supported")
	}

	// Validate client credentials
	var client domain.OAuthClient
	err := s.db.WithContext(ctx).
		Where("client_id = ? AND is_active = ?", req.ClientId, true).
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid client credentials")
		}
		s.log.Errorf("error finding OAuth client: %v", err)
		return nil, errors.New("internal server error")
	}

	// Verify client secret
	err = utils.VerifyClientSecret(req.ClientSecret, client.ClientSecret)
	if err != nil {
		s.log.Warnf("failed client secret verification for client_id: %s", req.ClientId)
		return nil, errors.New("invalid client credentials")
	}

	// Validate scopes
	requestedScope := req.Scope
	if requestedScope == "" {
		requestedScope = client.AllowedScopes // Use default allowed scopes
	}

	if !utils.ValidateScope(requestedScope, client.AllowedScopes) {
		return nil, fmt.Errorf("invalid scope: requested scopes are not allowed for this client")
	}

	// Generate JWT token
	scopes := utils.ParseScopes(requestedScope)

	// Build claims map for JWT
	claims := map[string]any{
		"client_id":  client.ClientId,
		"scopes":     scopes,
		"token_type": "client_credentials",
	}

	// Get token expiry from config
	expiresIn := s.cfg.Secret.OAuthTokenExpiry
	if expiresIn <= 0 {
		expiresIn = 3600 // Default to 1 hour
	}

	// Generate access token using existing JWT handler
	accessToken, err := s.jwtHandler.GenerateToken(claims, time.Duration(expiresIn)*time.Second)
	if err != nil {
		s.log.Errorf("failed to generate OAuth access token: %v", err)
		return nil, errors.New("failed to generate access token")
	}

	// Build response
	result := &domain.OAuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       strings.Join(scopes, " "),
	}

	s.log.Infof("OAuth token generated for client: %s", client.ClientId)

	return result, nil
}
