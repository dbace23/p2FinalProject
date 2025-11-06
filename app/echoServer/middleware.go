// app/echoServer/middleware.go
package echoServer

import (
	"log/slog"
	"time"

	"instagram/util/jwt"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func RegisterMiddlewares(e *echo.Echo) {

	e.Use(middleware.Recover())

	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string { return uuid.NewString() },
	}))

	e.Use(Slog())
}

func Slog() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			lat := time.Since(start).Milliseconds()

			rid := c.Response().Header().Get(echo.HeaderXRequestID)
			slog.Info("http",
				"method", c.Request().Method,
				"path", c.Path(),
				"status", c.Response().Status,
				"latency_ms", lat,
				"req_id", rid,
				"ip", c.RealIP(),
				"ua", c.Request().UserAgent(),
			)
			return err
		}
	}
}

func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, err := jwt.ParseAuth(c.Request().Header.Get("Authorization"), secret)
			if err != nil {
				return echo.NewHTTPError(401, "unauthenticated")
			}
			idf, ok := claims["sub"].(float64)
			if !ok {
				return echo.NewHTTPError(401, "unauthenticated")
			}
			c.Set("uid", int64(idf))
			return next(c)
		}
	}
}
