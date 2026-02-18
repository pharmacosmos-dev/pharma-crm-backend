package domain

import (
	"time"

	"github.com/google/uuid"
)

type OAuthRequest struct {
	GrantType    string `form:"grant_type" json:"grant_type" binding:"required"`
	ClientId     string `form:"client_id" json:"client_id" binding:"required"`
	ClientSecret string `form:"client_secret" json:"client_secret" binding:"required"`
	Scope        string `form:"scope" json:"scope"`
}

type OAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}

// OAuthClient represents an OAuth2 client in the database
type OAuthClient struct {
	Id            uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ClientId      string    `json:"client_id" gorm:"type:text;unique;not null"`
	ClientSecret  string    `json:"client_secret" gorm:"type:text;not null"` // Never expose in JSON
	ClientName    string    `json:"client_name" gorm:"type:text"`
	AllowedScopes string    `json:"allowed_scopes" gorm:"type:text;default:'read write'"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at" gorm:"default:now()"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"default:now()"`
}

func (OAuthClient) TableName() string {
	return "oauth_clients"
}

// OAuthTokenClaims represents the JWT claims for OAuth tokens
type OAuthTokenClaims struct {
	ClientId  string   `json:"client_id,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	TokenType string   `json:"token_type"` // "client_credentials" or "user"
	UserId    any      `json:"user_id,omitempty"`
	CompanyId any      `json:"company_id,omitempty"`
	StoreId   any      `json:"store_id,omitempty"`
	Role      any      `json:"role,omitempty"`
}
