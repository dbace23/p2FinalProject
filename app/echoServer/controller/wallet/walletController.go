package wallet

import (
	"bookrental/service/wallet"
	"log/slog"
	"net/http"

	"bookrental/app/echoServer/jwtx"

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
func (ct *Controller) CreateTopup(c echo.Context) error {
	uid, err := jwtx.UserIDFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or missing token")
	}

	var req CreateTopupReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	if ct.V != nil {
		if err := ct.V.Struct(req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "validation error")
		}
	}

	res, svcErr := ct.Svc.CreateTopup(c.Request().Context(), uid, req.Amount)
	if svcErr != nil {

		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create topup")
	}
	return c.JSON(http.StatusCreated, res)
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
