package rental

import (
	rs "bookrental/service/rental"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Controller struct {
	Svc rs.Service
	V   *validator.Validate
	Log *slog.Logger
}

// POST /v1/rentals
func (h *Controller) Create(c echo.Context) error {
	var req CreateRentalReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid JSON"})
	}
	if err := h.V.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "validation error",
			"errors":  err.Error(),
		})
	}
	uid, _ := c.Get("user_id").(int64)

	out, err := h.Svc.Create(c.Request().Context(), uid, req.BookID, req.PayerEmail)
	if err != nil {
		h.Log.Error("rental create", "err", err)
		switch rs.Code(err) {
		case rs.ErrNoStock:
			return c.JSON(http.StatusConflict, echo.Map{"message": "no stock available"})
		case rs.ErrBookNotFound:
			return c.JSON(http.StatusNotFound, echo.Map{"message": "book not found"})
		default:
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
		}
	}

	return c.JSON(http.StatusCreated, echo.Map{
		"rental_id":      out.RentalID,
		"status":         "BOOKED",
		"payment_link":   out.PaymentLink,
		"payment_due_at": out.PaymentDueAt,
	})
}

// POST /v1/rentals/:id/return
func (h *Controller) Return(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid id"})
	}
	uid, _ := c.Get("user_id").(int64)

	if err := h.Svc.Return(c.Request().Context(), uid, id); err != nil {
		h.Log.Error("rental return", "err", err)
		switch rs.Code(err) {
		case rs.ErrNotOwner:
			return c.JSON(http.StatusForbidden, echo.Map{"message": "forbidden"})
		case rs.ErrNotActive:
			return c.JSON(http.StatusConflict, echo.Map{"message": "rental not active"})
		case rs.ErrNotFound:
			return c.JSON(http.StatusNotFound, echo.Map{"message": "rental not found"})
		default:
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
		}
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "returned"})
}

// GET /v1/users/me/rentals
func (h *Controller) MyHistory(c echo.Context) error {
	uid, _ := c.Get("user_id").(int64)
	rows, err := h.Svc.MyHistory(c.Request().Context(), uid)
	if err != nil {
		h.Log.Error("history", "err", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}
	return c.JSON(http.StatusOK, echo.Map{"data": rows})
}
