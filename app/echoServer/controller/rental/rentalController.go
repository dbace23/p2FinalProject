package rental

import (
	"log/slog"
	"net/http"
	"strconv"

	svc "bookrental/service/rental"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Controller struct {
	Svc svc.Service
	V   *validator.Validate
	Log *slog.Logger
}

func userIDFrom(c echo.Context) (int64, bool) {
	v, ok := c.Get("user_id").(int64)
	return v, ok
}

// POST /v1/rentals/book
func (h *Controller) BookWithDeposit(c echo.Context) error {
	if h.Log != nil {
		h.Log.Info("rental.BookWithDeposit", "user_id", c.Get("user_id"))
	}
	var req BookWithDepositReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid JSON"})
	}
	if err := h.V.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "validation error", "errors": err.Error()})
	}
	uid, ok := userIDFrom(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"message": "unauthorized"})
	}

	hold := 0
	if req.HoldMinutes != nil {
		hold = *req.HoldMinutes
	}
	if err := h.Svc.BookWithDeposit(c.Request().Context(), uid, req.BookID, hold); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			if h.Log != nil {
				h.Log.Warn("book failed", "err", he, "user_id", uid, "book_id", req.BookID)
			}
			return he
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}
	return c.JSON(http.StatusCreated, echo.Map{"message": "booked"})
}

// POST /v1/rentals/:id/return
func (h *Controller) Return(c echo.Context) error {
	rid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || rid <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid rental id"})
	}
	uid, ok := userIDFrom(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"message": "unauthorized"})
	}

	if err := h.Svc.Return(c.Request().Context(), uid, rid); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return he
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "returned"})
}

// GET /v1/rentals/my
func (h *Controller) MyHistory(c echo.Context) error {
	uid, ok := userIDFrom(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"message": "unauthorized"})
	}
	rows, err := h.Svc.MyHistory(c.Request().Context(), uid)
	if err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return he
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}
	return c.JSON(http.StatusOK, rows)
}
