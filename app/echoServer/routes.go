package echoServer

import (
	"bookrental/app/echoServer/controller/auth"
	"bookrental/app/echoServer/controller/book"
	"bookrental/app/echoServer/controller/payment"
	"bookrental/app/echoServer/controller/rental"
	"bookrental/app/echoServer/controller/wallet"

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

	// Books
	auth.GET("/books", c.Book.List)
	auth.GET("/books/:id", c.Book.Detail)
	// Admin endpoints
	auth.POST("/books", c.Book.Create)
	auth.POST("/books/:id/copies", c.Book.AddCopies)

	// Wallet
	auth.POST("/wallet/topups", c.Wallet.CreateTopup) // returns payment link
	auth.GET("/wallet/ledger", c.Wallet.Ledger)       // list ledger

	// Rentals
	auth.POST("/rentals", c.Rental.Create)            // book + invoice
	auth.POST("/rentals/:id/return", c.Rental.Return) // return flow
	auth.GET("/users/me/rentals", c.Rental.MyHistory)
}
