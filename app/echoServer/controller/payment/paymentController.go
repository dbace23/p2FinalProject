package payment

import (
	paymentsvc "bookrental/service/payment"
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Controller struct {
	Svc paymentsvc.Service
	Log *slog.Logger
}

func (h *Controller) HandleXendit(c echo.Context) error {
	sig := c.Request().Header.Get("X-Callback-Token")
	h.Log.Info("xendit webhook",
		"ip", c.RealIP(),
		"token_present", sig != "",
	)
	raw, _ := io.ReadAll(c.Request().Body)

	if err := h.Svc.HandleXendit(c.Request().Context(), sig, raw); err != nil {
		h.Log.Error("payment callback error", "err", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "payment rejected"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "ok"})
}
