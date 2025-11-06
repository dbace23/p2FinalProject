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
	echoServer "book/app/echoServer"
	"book/app/echoServer/controller"
	"book/app/echoServer/validation"
	"book/config"
	authsvc "book/service/auth"
	"context"

	userrepo "book/repository/user"

	"book/util/database"
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func main() {

	cfg := config.Load()
	ctx := context.Background()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Pool.Close()

	// repos

	ur := userrepo.New(db)

	// services

	aus := authsvc.New(ur)

	// controllers

	uc := controller.NewUserController(aus, cfg.JWTSecret, slog.Default())

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
		User: uc,

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
