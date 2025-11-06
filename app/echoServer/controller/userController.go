// app/echoServer/controller/userController.go
package controller

import (
	"log/slog"
	"net/http"

	"bookrental/model"
	authsvc "bookrental/service/auth"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type UserController struct {
	Svc authsvc.Service
	V   *validator.Validate
	Log *slog.Logger
}

// Register
// @Summary      Register user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        payload  body  model.RegisterReq  true  "Register payload"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  map[string]any
// @Failure      409  {object}  map[string]any
// @Failure      500  {object}  map[string]any
// @Router       /v1/users/register [post]
func (ct *UserController) Register(c echo.Context) error {
	var req model.RegisterReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	if ct.V != nil {
		if err := ct.V.Struct(req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "validation error")
		}
	} else if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation error")
	}

	u, _, err := ct.Svc.Register(c.Request().Context(), req)
	if err != nil {
		switch authsvc.Code(err) {
		case authsvc.ErrEmailTaken, authsvc.ErrUsernameTaken:
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		case authsvc.ErrBadInput:
			return echo.NewHTTPError(http.StatusBadRequest, "bad input")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "register failed")
		}
	}

	return c.JSON(http.StatusCreated, echo.Map{
		"message": "registered",
		"user":    u,
	})
}

// Login
// @Summary      Login
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        payload  body  model.LoginReq  true  "Login payload"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  map[string]any
// @Failure      401  {object}  map[string]any
// @Failure      500  {object}  map[string]any
// @Router       /v1/users/login [post]
func (ct *UserController) Login(c echo.Context) error {
	var req model.LoginReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	if ct.V != nil {
		if err := ct.V.Struct(req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "validation error")
		}
	} else if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation error")
	}

	_, token, err := ct.Svc.Login(c.Request().Context(), req)
	if err != nil {
		switch authsvc.Code(err) {
		case authsvc.ErrInvalidCreds:
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
		case authsvc.ErrBadInput:
			return echo.NewHTTPError(http.StatusBadRequest, "bad input")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "login success",
		"token":   token,
	})
}
