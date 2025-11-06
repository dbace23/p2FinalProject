package echoServer

import (
	"instagram/app/echoServer/controller"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

type C struct {
	User     *controller.UserController
	Post     *controller.PostController
	Like     *controller.LikeController
	Activity *controller.ActivityController

	JWTSecret string
}

func Register(e *echo.Echo, c C) {
	// Public group
	pub := e.Group("/v1")
	pub.POST("/users/register", c.User.Register)
	pub.POST("/users/login", c.User.Login)

	// Protected group (JWT required)
	auth := e.Group("/v1")

	auth.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte(c.JWTSecret),
		TokenLookup: "header:Authorization",
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return jwt.MapClaims{}
		},
		ErrorHandler: func(ctx echo.Context, err error) error {
			hdr := ctx.Request().Header.Get("Authorization")
			return ctx.JSON(401, echo.Map{
				"message": "invalid or missing token",
				"error":   err.Error(),
				"got":     hdr,
			})
		},
	}))

	// Routes under auth
	auth.POST("/posts", c.Post.Create)
	auth.GET("/posts", c.Post.List)
	auth.GET("/posts/:id", c.Post.Detail)
	auth.DELETE("/posts/:id", c.Post.Delete)

	auth.POST("/likes", c.Like.Create)
	auth.DELETE("/likes/:id", c.Like.Delete)

	auth.GET("/activities", c.Activity.ListMine)
}
