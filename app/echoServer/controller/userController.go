// app/echoServer/controller/userController.go
package controller

import (
	"errors"
	"log/slog"
	"net/http"

	"instagram/model"
	authsvc "instagram/service/auth"

	"github.com/labstack/echo/v4"
)

type UserController struct {
	s         authsvc.Service
	jwtSecret string
	log       *slog.Logger
}

func NewUserController(s authsvc.Service, secret string, log *slog.Logger) *UserController {
	return &UserController{
		s:         s,
		jwtSecret: secret,
		log:       log,
	}
}

// Register a new user
// @Summary      Register user
// @Description  Register a new user with email/username uniqueness and validation
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        payload  body  model.RegisterReq  true  "Register payload"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  map[string]any
// @Failure      409  {object}  map[string]any "email/username already taken"
// @Failure      500  {object}  map[string]any "internal server error"
// @Router       /v1/users/register [post]
func (ct *UserController) Register(c echo.Context) error {
	var req model.RegisterReq

	// Bind
	if err := c.Bind(&req); err != nil {
		ct.log.Warn("bind failed", "path", c.Path(), "err", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	// Validate
	if err := c.Validate(&req); err != nil {
		ct.log.Warn("validation failed", "path", c.Path(), "err", err)
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	// Business logic
	u, _, err := ct.s.Register(c.Request().Context(), req, ct.jwtSecret)
	if err != nil {
		switch {
		case errors.Is(err, authsvc.ErrEmailTaken):
			// 409
			return echo.NewHTTPError(http.StatusConflict, "email already registered")
		case errors.Is(err, authsvc.ErrUsernameTaken):
			// 409
			return echo.NewHTTPError(http.StatusConflict, "username already taken")
		case errors.Is(err, authsvc.ErrBadInput):
			// 400
			ct.log.Warn("bad input", "path", c.Path(), "err", err)
			return echo.NewHTTPError(http.StatusBadRequest)
		default:
			// Unknown → 500 (details only in logs)
			rid := c.Response().Header().Get(echo.HeaderXRequestID)
			ct.log.Error("register failed",
				"err", err,
				"req_id", rid,
				"path", c.Path(),
				"method", c.Request().Method,
			)
			return echo.NewHTTPError(http.StatusInternalServerError, "register failed")
		}
	}

	// Success
	return c.JSON(http.StatusCreated, echo.Map{
		"message": "registered",
		"user":    u,
	})
}

// Login
// @Summary      Login
// @Description  Login with email + password, returns JWT
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
		ct.log.Warn("bind failed", "path", c.Path(), "err", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	if err := c.Validate(&req); err != nil {
		ct.log.Warn("validation failed", "path", c.Path(), "err", err)
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	_, token, err := ct.s.Login(c.Request().Context(), req, ct.jwtSecret)
	if err != nil {
		switch {
		case errors.Is(err, authsvc.ErrInvalidCreds):
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
		case errors.Is(err, authsvc.ErrBadInput):
			ct.log.Warn("bad input", "path", c.Path(), "err", err)
			return echo.NewHTTPError(http.StatusBadRequest)
		default:
			rid := c.Response().Header().Get(echo.HeaderXRequestID)
			ct.log.Error("login failed",
				"err", err,
				"req_id", rid,
				"path", c.Path(),
				"method", c.Request().Method,
			)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "login success",
		"token":   token,
	})
}
