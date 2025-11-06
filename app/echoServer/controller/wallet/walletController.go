package wallet

import (
	"bookrental/service/wallet"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Controller struct {
	Svc wallet.Service
	V   *validator.Validate
	Log *slog.Logger
}

// POST /v1/wallet/topups
// @Summary Create top-up invoice (Xendit)
// @Success 201 {object} map[string]any
// @Failure 400,401,500
func (h *Controller) CreateTopup(c echo.Context) error {
	var req CreateTopupReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid json"})
	}
	if err := h.V.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "validation error",
			"errors":  map[string]string{"amount": "required, gt 0"},
		})
	}
	userID := c.Get("user_id").(int64)
	res, err := h.Svc.CreateTopup(c.Request().Context(), userID, req.Amount)
	if err != nil {
		h.Log.Error("CreateTopup failed", "err", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}
	return c.JSON(http.StatusCreated, echo.Map{
		"invoice_id":   res.InvoiceID,
		"payment_link": res.PaymentLink,
		"expires_at":   res.ExpiresAt,
	})
}

// GET /v1/wallet/ledger
func (h *Controller) Ledger(c echo.Context) error {
	userID := c.Get("user_id").(int64)
	rows, err := h.Svc.Ledger(c.Request().Context(), userID)
	if err != nil {
		h.Log.Error("Ledger failed", "err", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}
	return c.JSON(http.StatusOK, echo.Map{"data": rows})
}
