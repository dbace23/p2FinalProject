package jwtx

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

var ErrNoToken = errors.New("missing jwt token")
var ErrNoSub = errors.New("missing sub claim")
var ErrBadSub = errors.New("invalid sub claim")

func UserIDFromContext(c echo.Context) (int64, error) {
	tok, ok := c.Get("user").(*jwt.Token)
	if !ok || tok == nil {
		return 0, ErrNoToken
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return 0, ErrNoToken
	}

	for _, k := range []string{"sub", "user_id", "uid"} {
		if v, ok := claims[k]; ok {
			switch vv := v.(type) {
			case float64:
				return int64(vv), nil
			case int64:
				return vv, nil
			case json.Number:
				if n, err := vv.Int64(); err == nil {
					return n, nil
				}
			case string:
				if n, err := strconv.ParseInt(vv, 10, 64); err == nil {
					return n, nil
				}
			}
			return 0, ErrBadSub
		}
	}
	return 0, ErrNoSub
}
