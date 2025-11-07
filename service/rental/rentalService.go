// service/rental/service.go
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
	BookWithDeposit(ctx context.Context, userID, bookID int64, holdMinutes int) error
}

type service struct {
	db *sql.DB
	rr rentalrepo.Repo
}

func New(db *sql.DB, rr rentalrepo.Repo) Service {
	return &service{db: db, rr: rr}
}

func (s *service) BookWithDeposit(ctx context.Context, userID, bookID int64, holdMinutes int) (err error) {
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

	deposit, err := s.rr.LockUserForUpdate(ctx, tx, userID)
	if err != nil {
		return echo.NewHTTPError(404, echo.Map{"message": "user not found"})
	}

	price, err := s.rr.GetBookPrice(ctx, tx, bookID)
	if err != nil {
		return echo.NewHTTPError(404, echo.Map{"message": "book not found"})
	}

	if deposit < price {
		return echo.NewHTTPError(402, echo.Map{
			"message": "insufficient deposit",
			"needed":  price - deposit,
		})
	}

	itemID, err := s.rr.LockOneAvailableItem(ctx, tx, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(409, echo.Map{"message": "no available copy"})
		}
		return fmt.Errorf("select item: %w", err)
	}

	if err = s.rr.DeductDeposit(ctx, tx, userID, price); err != nil {
		return fmt.Errorf("deduct deposit: %w", err)
	}

	var holdUntil *time.Time
	if holdMinutes > 0 {
		t := time.Now().Add(time.Duration(holdMinutes) * time.Minute).UTC()
		holdUntil = &t
	}
	if err = s.rr.ReserveItem(ctx, tx, itemID, holdUntil); err != nil {
		return fmt.Errorf("reserve item: %w", err)
	}

	if err = s.rr.InsertRental(ctx, tx, userID, bookID, itemID, price, holdUntil); err != nil {
		return fmt.Errorf("insert rental: %w", err)
	}

	return nil
}
