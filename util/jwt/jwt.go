package jwt

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func Issue(secret string, userID int64, role string, ttlHours int) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(time.Duration(ttlHours) * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func ParseAuth(authHeader string, secret string) (map[string]any, error) {
	tokenStr := strings.TrimSpace(authHeader)
	if tokenStr == "" {
		return nil, errors.New("missing authorization")
	}

	if strings.HasPrefix(strings.ToLower(tokenStr), "bearer ") {
		tokenStr = strings.TrimSpace(tokenStr[7:])
	}
	if tokenStr == "" {
		return nil, errors.New("missing token")
	}

	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {

		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("invalid token")
	}

	mc, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	if exp, ok := mc["exp"]; ok {
		switch v := exp.(type) {
		case float64:
			if time.Now().Unix() > int64(v) {
				return nil, errors.New("token expired")
			}
		case jsonNumber:

			if i, err := strconv.ParseInt(string(v), 10, 64); err == nil {
				if time.Now().Unix() > i {
					return nil, errors.New("token expired")
				}
			}
		}
	}

	out := make(map[string]any, len(mc))
	for k, v := range mc {
		out[k] = v
	}
	return out, nil
}

type jsonNumber string
