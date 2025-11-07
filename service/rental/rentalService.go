package rental

import (
	rentalrepo "bookrental/repository/rental"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

type Service interface {
	// Borrow using user deposit (deducts deposit, reserves an item, inserts rental).
	BookWithDeposit(ctx context.Context, userID, bookID int64, holdMinutes int) error
	// Return an ACTIVE rental, free the copy.
	Return(ctx context.Context, userID, rentalID int64) error
	// List my rental history.
	MyHistory(ctx context.Context, userID int64) ([]HistoryRow, error)
}

type HistoryRow = rentalrepo.HistoryRow

type service struct {
	db *sql.DB
	rr rentalrepo.Repo
}

func New(db *sql.DB, rr rentalrepo.Repo) Service {
	return &service{db: db, rr: rr}
}

// BookWithDeposit:
// 1) Lock user → check deposit
// 2) Get book price
// 3) Lock one available item
// 4) Deduct deposit
// 5) Reserve item
// 6) Insert rental
func (s *service) BookWithDeposit(ctx context.Context, userID, bookID int64, holdMinutes int) (err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return echo.NewHTTPError(500, echo.Map{"message": "begin tx failed"})
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	// 1) Lock user and read deposit
	deposit, err := s.rr.LockUserForUpdate(ctx, tx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(404, echo.Map{"message": "user not found"})
		}
		return echo.NewHTTPError(500, echo.Map{"message": fmt.Sprintf("lock user: %v", err)})
	}

	// 2) Get price
	price, err := s.rr.GetBookPrice(ctx, tx, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(404, echo.Map{"message": "book not found"})
		}
		return echo.NewHTTPError(500, echo.Map{"message": fmt.Sprintf("get price: %v", err)})
	}

	// 3) Check deposit
	if deposit < price {
		return echo.NewHTTPError(402, echo.Map{
			"message": "insufficient deposit",
			"needed":  price - deposit,
		})
	}

	// 4) Lock one available item
	itemID, err := s.rr.LockOneAvailableItem(ctx, tx, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(409, echo.Map{"message": "no available copy"})
		}
		return echo.NewHTTPError(500, echo.Map{"message": fmt.Sprintf("select item: %v", err)})
	}

	// 5) Deduct deposit
	if err = s.rr.DeductDeposit(ctx, tx, userID, price); err != nil {
		return echo.NewHTTPError(500, echo.Map{"message": fmt.Sprintf("deduct deposit: %v", err)})
	}

	// 6) Reserve item with optional hold
	var holdUntil *time.Time
	if holdMinutes > 0 {
		t := time.Now().Add(time.Duration(holdMinutes) * time.Minute).UTC()
		holdUntil = &t
	}
	if err = s.rr.ReserveItem(ctx, tx, itemID, holdUntil); err != nil {
		return echo.NewHTTPError(500, echo.Map{"message": fmt.Sprintf("reserve item: %v", err)})
	}

	if err = s.rr.InsertRental(ctx, tx, userID, bookID, itemID, price, holdUntil); err != nil {
		return echo.NewHTTPError(500, echo.Map{"message": fmt.Sprintf("insert rental: %v", err)})
	}

	return nil
}

// Return marks an ACTIVE rental returned and frees the copy.
// Business rules:
// - Only owner can return (403)
// - Rental must be ACTIVE (409)
// - Rental must exist (404)
func (s *service) Return(ctx context.Context, userID, rentalID int64) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return echo.NewHTTPError(500, echo.Map{"message": "begin tx failed"})
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	owner, status, copyID, err := s.rr.GetRentalOwnerAndStatus(ctx, tx, rentalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(404, echo.Map{"message": "rental not found"})
		}
		return echo.NewHTTPError(500, echo.Map{"message": fmt.Sprintf("load rental: %v", err)})
	}
	if owner != userID {
		return echo.NewHTTPError(403, echo.Map{"message": "not the owner of this rental"})
	}
	if status != "ACTIVE" {
		return echo.NewHTTPError(409, echo.Map{"message": "rental is not ACTIVE"})
	}

	if err = s.rr.MarkReturned(ctx, tx, rentalID); err != nil {
		return echo.NewHTTPError(500, echo.Map{"message": fmt.Sprintf("mark returned: %v", err)})
	}
	if err = s.rr.FreeCopy(ctx, tx, copyID); err != nil {
		return echo.NewHTTPError(500, echo.Map{"message": fmt.Sprintf("free copy: %v", err)})
	}
	return nil
}

// MyHistory returns rentals for a user.
func (s *service) MyHistory(ctx context.Context, userID int64) ([]HistoryRow, error) {
	rows, err := s.rr.ListMyRentals(ctx, userID)
	if err != nil {
		return nil, echo.NewHTTPError(500, echo.Map{"message": fmt.Sprintf("list rentals: %v", err)})
	}
	return rows, nil
}
