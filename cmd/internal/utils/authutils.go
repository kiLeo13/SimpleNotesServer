package utils

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"strings"
)

var parser = new(jwt.Parser)

type TokenData struct {
	// Sub describes the user's ID on Cognito.
	// This value will never be empty.
	Sub string

	// Email the user's Email.
	// This value will be empty if the provided token is an Access Token, for instance.
	Email string
}

func ParseTokenData(token string) (*TokenData, error) {
	if token == "" {
		return nil, errors.New("token is empty")
	}

	clean := sanitizeToken(token)
	claims, err := GetUnsafeClaims(clean)
	if err != nil {
		return nil, err
	}

	return &TokenData{
		Sub:   getValue(claims, "sub"),
		Email: getValue(claims, "email"),
	}, nil
}

func ParseTokenDataCtx(ctx echo.Context) (*TokenData, error) {
	token := ctx.Request().Header.Get("Authorization")
	clean := sanitizeToken(token)
	return ParseTokenData(clean)
}

// GetUnsafeClaims DOES NOT check if the claims are valid.
// However, we are safe to use it, as all requests go through API Gateway first.
func GetUnsafeClaims(tokenString string) (jwt.MapClaims, error) {
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid claims format")
}

func sanitizeToken(token string) string {
	var clean string
	if strings.HasPrefix(token, "Bearer") {
		clean = strings.TrimPrefix(token, "Bearer")
	} else {
		clean = token
	}
	return strings.TrimSpace(clean)
}

func getValue(claims jwt.MapClaims, key string) string {
	claim, ok := claims[key].(string)
	if !ok {
		return ""
	}
	return claim
}
