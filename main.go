// Package main book API.
//
// @title           book Mini API
// @version         1.0
// @description     book service (posts, likes, activities, users).
// @contact.name    Halim Iskandar
// @contact.email   halim.iskandar2323@gmail.com
// @BasePath        /
// @schemes         http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description  Use:  Bearer <JWT>
package main

import (
	"bookrental/app/echoServer"
	authctrl "bookrental/app/echoServer/controller/auth"
	bookctrl "bookrental/app/echoServer/controller/book"
	paymentctrl "bookrental/app/echoServer/controller/payment"
	rentalctrl "bookrental/app/echoServer/controller/rental"
	walletctrl "bookrental/app/echoServer/controller/wallet"
	"bookrental/app/echoServer/validation"
	"bookrental/config"
	authrepo "bookrental/repository/auth"
	bookrepo "bookrental/repository/book"
	rentalrepo "bookrental/repository/rental"
	walletrepo "bookrental/repository/wallet"
	xenditrepo "bookrental/repository/xendit"
	authsvc "bookrental/service/auth"
	booksvc "bookrental/service/book"
	paymentsvc "bookrental/service/payment"
	rentalsvc "bookrental/service/rental"
	walletsvc "bookrental/service/wallet"
	"bookrental/util/database"
	"context"
	"log/slog"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func main() {

	cfg := config.Load()
	ctx := context.Background()

	// logger
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// DB: *sql.DB
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// repos
	ar := authrepo.New(db)
	br := bookrepo.New(db)
	rr := rentalrepo.New(db)
	wr := walletrepo.New(db)
	xr := xenditrepo.NewHTTP(os.Getenv("XENDIT_API_KEY"))

	// services
	as := authsvc.New(ar, cfg.JWTSecret)
	bs := booksvc.New(br)
	rs := rentalsvc.New(db, rr)
	ws := walletsvc.New(db, wr, xr)
	whs := paymentsvc.New(db, xr, wr)

	// controllers
	v := validator.New()
	authC := &authctrl.Controller{Svc: as, V: v, Log: log}
	bookC := &bookctrl.Controller{Svc: bs, V: v, Log: log}
	rentalC := &rentalctrl.Controller{Svc: rs, V: v, Log: log}
	walletC := &walletctrl.Controller{Svc: ws, V: v, Log: log}
	paymentC := &paymentctrl.Controller{Svc: whs, Log: log}

	// echo
	e := echo.New()
	echoServer.RegisterMiddlewares(e)
	e.Validator = validation.New()

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]any{
			"status":  "ok",
			"message": "Service is healthy and connected",
		})
	})

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	echoServer.Register(e, echoServer.C{
		Auth:    authC,
		Book:    bookC,
		Rental:  rentalC,
		Wallet:  walletC,
		Payment: paymentC,

		JWTSecret: cfg.JWTSecret,
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Port
	}
	if port == "" {
		port = "8080"
	}

	slog.Info("starting server", "PORT_env", os.Getenv("PORT"), "chosen_port", port)

	e.Logger.Fatal(e.Start(":" + port))
}
