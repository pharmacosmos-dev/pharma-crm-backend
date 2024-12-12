package token

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/pkg/logger"
)

type JWTHandler struct {
	Cfg *config.Config
	Log *logger.Logger
}

// GenerateTokens generates access and refresh tokens.
func (j *JWTHandler) GenerateTokens(accessClaims map[string]interface{}, refreshClaims map[string]interface{}) (accessToken string, refreshToken string, err error) {
	// Generate access token
	accessToken, err = j.generateToken(accessClaims, config.AccessTokenExpiresInTime)
	if err != nil {
		j.Log.Error("Failed to generate access token:", err)
		return "", "", err
	}

	// Generate refresh token
	refreshToken, err = j.generateToken(refreshClaims, config.RefreshTokenExpiresInTime)
	if err != nil {
		j.Log.Error("Failed to generate refresh token:", err)
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// generateToken generates a JWT with the provided claims and expiration duration.
func (j *JWTHandler) generateToken(claimsMap map[string]interface{}, expiresIn time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)

	for key, value := range claimsMap {
		claims[key] = value
	}

	claims["iat"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(expiresIn).Unix()

	tokenString, err := token.SignedString([]byte(j.Cfg.Secret.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ExtractClaims extracts claims from given token
func (j *JWTHandler) ExtractClaims(tokenString string, tokenSecretKey string) (jwt.MapClaims, error) {
	var (
		token *jwt.Token
		err   error
	)

	token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// check token signing method etc
		return []byte(tokenSecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ExtractToken checks and returns token part of input string
func (j *JWTHandler) ExtractToken(bearer string) (token string, err error) {
	strArr := strings.Split(bearer, " ")
	if len(strArr) == 2 {
		return strArr[1], nil
	}
	return token, errors.New("wrong token format")
}

func (j *JWTHandler) VerifyToken(tokenString string, cfg config.Config) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.Secret.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	cliams, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return nil, err
	}

	return cliams, err
}
