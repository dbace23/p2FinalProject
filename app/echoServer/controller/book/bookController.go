package book

import (
	booksvc "bookrental/service/book"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Controller struct {
	Svc booksvc.Service
	V   *validator.Validate
	Log *slog.Logger
}

func isAdmin(c echo.Context) bool {

	role, _ := c.Get("role").(string)
	return role == "admin"
}

// POST /v1/books  (admin)
func (h *Controller) Create(c echo.Context) error {
	if !isAdmin(c) {
		return c.JSON(http.StatusForbidden, echo.Map{"message": "forbidden"})
	}
	var req CreateBookReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid json"})
	}
	if err := h.V.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "validation error",
			"errors":  echo.Map{"name": "required", "category": "required", "rental_cost": "gte 0"},
		})
	}
	id, err := h.Svc.Create(c.Request().Context(), req.Name, req.Category, req.RentalCost)
	if err != nil {
		h.Log.Error("book create error", "err", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}
	return c.JSON(http.StatusCreated, echo.Map{"id": id})
}

// POST /v1/books/:id/copies  (admin)
func (h *Controller) AddCopies(c echo.Context) error {
	if !isAdmin(c) {
		return c.JSON(http.StatusForbidden, echo.Map{"message": "forbidden"})
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid id"})
	}
	var req AddCopiesReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid json"})
	}
	if err := h.V.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "validation error", "errors": echo.Map{"count": "gt 0"}})
	}
	added, err := h.Svc.AddCopies(c.Request().Context(), id, req.Count)
	if err != nil {
		h.Log.Error("add copies error", "err", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}
	return c.JSON(http.StatusCreated, echo.Map{"added": added})
}

// GET /v1/books
func (h *Controller) List(c echo.Context) error {
	rows, err := h.Svc.List(c.Request().Context())
	if err != nil {
		h.Log.Error("book list error", "err", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}
	return c.JSON(http.StatusOK, echo.Map{"data": rows})
}

// GET /v1/books/:id
func (h *Controller) Detail(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid id"})
	}
	row, err := h.Svc.Detail(c.Request().Context(), id)
	if err != nil {
		h.Log.Error("book detail error", "err", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal error"})
	}
	if row == nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "not found"})
	}
	return c.JSON(http.StatusOK, row)
}
