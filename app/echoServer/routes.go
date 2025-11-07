package echoServer

import (
	"bookrental/app/echoServer/controller/auth"
	"bookrental/app/echoServer/controller/book"
	"bookrental/app/echoServer/controller/payment"
	"bookrental/app/echoServer/controller/rental"
	"bookrental/app/echoServer/controller/wallet"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

type C struct {
	Auth      *auth.Controller
	Book      *book.Controller
	Rental    *rental.Controller
	Wallet    *wallet.Controller
	Payment   *payment.Controller
	JWTSecret string
}

func Register(e *echo.Echo, c C) {
	// Public
	pub := e.Group("/v1")
	pub.POST("/users/register", c.Auth.Register)
	pub.POST("/users/login", c.Auth.Login)

	// payment
	pub.POST("/payment/xendit", c.Payment.HandleXendit)

	// Auth
	auth := e.Group("/v1")
	auth.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(c.JWTSecret),

		NewClaimsFunc: func(c echo.Context) jwt.Claims { return jwt.MapClaims{} },
		TokenLookup:   "header:Authorization",
	}))
	// 🔍 JWT debug + user_id extraction
	auth.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			reqID := ctx.Response().Header().Get(echo.HeaderXRequestID)
			authHeader := ctx.Request().Header.Get("Authorization")

			if authHeader == "" {
				ctx.Logger().Warnf("[AUTH] missing Authorization header req_id=%s ip=%s", reqID, ctx.RealIP())
				return ctx.JSON(http.StatusUnauthorized, echo.Map{"message": "unauthorized"})
			}

			tokenObj := ctx.Get("user")
			if tokenObj == nil {
				ctx.Logger().Warnf("[AUTH] tokenObj nil req_id=%s header=%s", reqID, authHeader)
				return ctx.JSON(http.StatusUnauthorized, echo.Map{"message": "unauthorized"})
			}

			claims, ok := tokenObj.(jwt.MapClaims)
			if !ok {
				ctx.Logger().Warnf("[AUTH] failed to cast claims req_id=%s", reqID)
				return ctx.JSON(http.StatusUnauthorized, echo.Map{"message": "unauthorized"})
			}

			sub, ok := claims["sub"].(float64)
			if !ok {
				ctx.Logger().Warnf("[AUTH] missing sub claim req_id=%s claims=%v", reqID, claims)
				return ctx.JSON(http.StatusUnauthorized, echo.Map{"message": "unauthorized"})
			}

			ctx.Set("user_id", int64(sub))
			ctx.Logger().Infof("[AUTH] verified user_id=%d req_id=%s ip=%s", int64(sub), reqID, ctx.RealIP())
			return next(ctx)
		}
	})

	// Books
	auth.GET("/books", c.Book.List)
	auth.GET("/books/:id", c.Book.Detail)
	// Admin endpoints
	auth.POST("/books", c.Book.Create)
	auth.POST("/books/:id/copies", c.Book.AddCopies)

	// Wallet
	auth.POST("/wallet/topups", c.Wallet.CreateTopup) // returns payment link
	auth.GET("/wallet/ledger", c.Wallet.Ledger)       // list ledger

	auth.POST("/rentals/book", c.Rental.BookWithDeposit)
	auth.POST("/rentals/:id/return", c.Rental.Return)
	auth.GET("/rentals/my", c.Rental.MyHistory)
}
