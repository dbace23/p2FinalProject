// app/echoServer/controller/rental/controller.go
package rental

import (
	"log/slog"
	"net/http"

	svc "bookrental/service/rental"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Controller struct {
	Svc svc.Service
	V   *validator.Validate
	Log *slog.Logger
}

type BookReq struct {
	BookID      int64 `json:"book_id" validate:"required,gt=0"`
	HoldMinutes int   `json:"hold_minutes" validate:"gte=0,lte=1440"`
}

func (rc *Controller) Book(c echo.Context) error {
	var req BookReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echo.Map{"message": "invalid payload"})
	}
	if rc.V != nil {
		if err := rc.V.Struct(req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
				"message": "validation error",
				"errors":  err.Error(),
			})
		}
	}

	uid, ok := c.Get("userID").(int64)
	if !ok || uid == 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, echo.Map{"message": "unauthorized"})
	}

	if err := rc.Svc.BookWithDeposit(c.Request().Context(), uid, req.BookID, req.HoldMinutes); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return he
		}
		return echo.NewHTTPError(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}

	return c.JSON(http.StatusCreated, echo.Map{"message": "booked"})
}
