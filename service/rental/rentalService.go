package rental

import (
	rentalrepo "bookrental/repository/rental"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

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
	var dbgCount int64
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books`).Scan(&dbgCount)
	slog.Info("dbg books count", "count", dbgCount)
	//debugging

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	// Lock user for deposit check
	var deposit float64
	err = tx.QueryRowContext(ctx,
		`SELECT deposit_balance FROM users WHERE id=$1 FOR UPDATE`, userID).
		Scan(&deposit)
	if err != nil {
		return echo.NewHTTPError(404, echo.Map{"message": "user not found"})
	}

	// Lock book for availability check
	var price float64
	var stock int64
	err = tx.QueryRowContext(ctx,
		`SELECT rental_cost, stock_availability
		 FROM books
		 WHERE id=$1
		 FOR UPDATE`, bookID).
		Scan(&price, &stock)
	if err != nil {
		return echo.NewHTTPError(404, echo.Map{"message": "book not found"})
	}

	if stock <= 0 {
		return echo.NewHTTPError(409, echo.Map{"message": "no available copy"})
	}
	if deposit < price {
		return echo.NewHTTPError(402, echo.Map{
			"message": "insufficient deposit",
			"needed":  price - deposit,
		})
	}

	// Deduct deposit and decrease stock
	if _, err = tx.ExecContext(ctx,
		`UPDATE users
		 SET deposit_balance = deposit_balance - $1
		 WHERE id = $2`,
		price, userID); err != nil {
		return fmt.Errorf("deduct deposit: %w", err)
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE books
		 SET stock_availability = stock_availability - 1
		 WHERE id = $1`, bookID); err != nil {
		return fmt.Errorf("decrease stock: %w", err)
	}

	// Insert rental record
	if _, err = tx.ExecContext(ctx,
		`INSERT INTO rentals (user_id, book_id, status, price)
		 VALUES ($1, $2, 'BOOKED', $3)`,
		userID, bookID, price); err != nil {
		return fmt.Errorf("insert rental: %w", err)
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
