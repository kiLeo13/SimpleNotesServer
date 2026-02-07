package utils

import (
	"errors"
	"fmt"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"strings"
)

var jwks keyfunc.Keyfunc

func InitJWKS(region, poolID string) error {
	// URL where Cognito publishes its public keys
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, poolID)

	var err error
	jwks, err = keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return fmt.Errorf("failed to create JWKS from resource at %s: %w", jwksURL, err)
	}

	log.Infof("JWKS initialized. Keys loaded from %s", jwksURL)
	return nil
}

type TokenData struct {
	Sub   string
	Email string
	Exp   int64
}

// ValidateToken parses AND validates the signature locally.
// It returns the data if the token is authentic and unexpired.
func ValidateToken(tokenString string) (*TokenData, error) {
	if jwks == nil {
		return nil, errors.New("JWKS not initialized")
	}

	clean := sanitizeToken(tokenString)
	token, err := jwt.Parse(clean, jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims format")
	}

	return &TokenData{
		Sub:   getValue(claims, "sub"),
		Email: getValue(claims, "email"),
		Exp:   getInt64(claims, "exp"),
	}, nil
}

func ParseTokenDataCtx(ctx echo.Context) (*TokenData, error) {
	token := ctx.Request().Header.Get(echo.HeaderAuthorization)
	return ValidateToken(token)
}

func sanitizeToken(token string) string {
	return strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
}

func getValue(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key].(string); ok {
		return val
	}
	return ""
}

func getInt64(claims jwt.MapClaims, key string) int64 {
	val, ok := claims[key]
	if !ok {
		return 0
	}
	if f, ok := val.(float64); ok {
		return int64(f)
	}
	if i, ok := val.(int64); ok {
		return i
	}
	return 0
}
