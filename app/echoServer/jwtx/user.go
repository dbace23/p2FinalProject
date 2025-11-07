// util/jwt/jwtx.go
package jwtx

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func UserIDFromContext(c echo.Context) (int64, error) {
	tok, ok := c.Get("user").(*jwt.Token)
	if !ok || tok == nil {
		return 0, errors.New("no jwt token in context")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid jwt claims")
	}

	if f, ok := claims["sub"].(float64); ok {
		return int64(f), nil
	}
	return 0, errors.New("sub missing in claims")
}

func EmailFromContext(c echo.Context) (string, error) {
	tok, ok := c.Get("user").(*jwt.Token)
	if !ok || tok == nil {
		return "", errors.New("no jwt token in context")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid jwt claims")
	}
	if s, ok := claims["email"].(string); ok && s != "" {
		return s, nil
	}
	return "", errors.New("email missing in claims")
}
