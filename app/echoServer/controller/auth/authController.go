// app/echoServer/controller/userController.go
package auth

import (
	"log/slog"
	"net/http"

	"bookrental/model"
	authsvc "bookrental/service/auth"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Controller struct {
	Svc authsvc.Service
	V   *validator.Validate
	Log *slog.Logger
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
func (ct *Controller) Register(c echo.Context) error {
	var req model.RegisterReq

	// Bind
	if err := c.Bind(&req); err != nil {
		if ct.Log != nil {
			ct.Log.Warn("bind failed", "path", c.Path(), "err", err)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	// Validate
	if ct.V != nil {
		if err := ct.V.Struct(req); err != nil {
			if ct.Log != nil {
				ct.Log.Warn("validation failed", "path", c.Path(), "err", err)
			}
			return echo.NewHTTPError(http.StatusBadRequest, "validation error")
		}
	} else if err := c.Validate(&req); err != nil {
		if ct.Log != nil {
			ct.Log.Warn("validation failed", "path", c.Path(), "err", err)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "validation error")
	}

	// Business logic
	u, _, err := ct.Svc.Register(c.Request().Context(), req)
	if err != nil {
		switch authsvc.Code(err) {
		case authsvc.ErrEmailTaken:
			return echo.NewHTTPError(http.StatusConflict, "email already registered")
		case authsvc.ErrUsernameTaken:
			return echo.NewHTTPError(http.StatusConflict, "username already taken")
		case authsvc.ErrInvalidCreds:
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
		case authsvc.ErrBadInput:
			return echo.NewHTTPError(http.StatusBadRequest, "bad input")
		default:
			if ct.Log != nil {
				rid := c.Response().Header().Get(echo.HeaderXRequestID)
				ct.Log.Error("register failed",
					"err", err,
					"req_id", rid,
					"path", c.Path(),
					"method", c.Request().Method,
				)
			}
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
func (ct *Controller) Login(c echo.Context) error {
	var req model.LoginReq

	if err := c.Bind(&req); err != nil {
		if ct.Log != nil {
			ct.Log.Warn("bind failed", "path", c.Path(), "err", err)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	if ct.V != nil {
		if err := ct.V.Struct(req); err != nil {
			if ct.Log != nil {
				ct.Log.Warn("validation failed", "path", c.Path(), "err", err)
			}
			return echo.NewHTTPError(http.StatusBadRequest, "validation error")
		}
	} else if err := c.Validate(&req); err != nil {
		if ct.Log != nil {
			ct.Log.Warn("validation failed", "path", c.Path(), "err", err)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "validation error")
	}

	_, token, err := ct.Svc.Login(c.Request().Context(), req)
	if err != nil {
		switch authsvc.Code(err) {
		case authsvc.ErrInvalidCreds:
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
		case authsvc.ErrBadInput:
			if ct.Log != nil {
				ct.Log.Warn("bad input", "path", c.Path(), "err", err)
			}
			return echo.NewHTTPError(http.StatusBadRequest, "bad input")
		default:
			rid := c.Response().Header().Get(echo.HeaderXRequestID)
			if ct.Log != nil {
				ct.Log.Error("login failed",
					"err", err,
					"req_id", rid,
					"path", c.Path(),
					"method", c.Request().Method,
				)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
		}

	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "login success",
		"token":   token,
	})
}
