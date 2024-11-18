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

type JwtHandler struct {
	cfg config.Config
	log logger.Logger
}

// GenerateJWT ...
func (h *JwtHandler) GenerateJWT(m map[string]interface{}, tokenExpireTime time.Duration, tokenSecretKey string) (tokenString string, err error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)

	for key, value := range m {
		claims[key] = value
	}
	
	claims["iat"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(tokenExpireTime).Unix()

	tokenString, err = token.SignedString([]byte(tokenSecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ExtractClaims extracts claims from given token
func (j *JwtHandler) ExtractClaims(tokenString string, tokenSecretKey string) (jwt.MapClaims, error) {
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
func (j *JwtHandler) ExtractToken(bearer string) (token string, err error) {
	strArr := strings.Split(bearer, " ")
	if len(strArr) == 2 {
		return strArr[1], nil
	}
	return token, errors.New("wrong token format")
}

func (j *JwtHandler) VerifyToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.cfg.Secret.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	cliams, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		j.log.Error("invalid jwt token")
		return nil, err
	}

	return cliams, err
}
