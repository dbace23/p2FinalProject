// repository/rental/repo.go
package rentalrepo

import (
	"context"
	"database/sql"
	"time"
)

type Repo interface {
	LockUserForUpdate(ctx context.Context, tx *sql.Tx, userID int64) (deposit int64, err error)
	GetBookPrice(ctx context.Context, tx *sql.Tx, bookID int64) (price int64, err error)
	LockOneAvailableItem(ctx context.Context, tx *sql.Tx, bookID int64) (itemID int64, err error)
	DeductDeposit(ctx context.Context, tx *sql.Tx, userID int64, amount int64) error
	ReserveItem(ctx context.Context, tx *sql.Tx, itemID int64, holdUntil *time.Time) error
	InsertRental(ctx context.Context, tx *sql.Tx, userID, bookID, itemID, price int64, holdUntil *time.Time) error
}

type repo struct{ db *sql.DB }

func New(db *sql.DB) Repo { return &repo{db} }

func (r *repo) LockUserForUpdate(ctx context.Context, tx *sql.Tx, userID int64) (int64, error) {
	var deposit int64
	err := tx.QueryRowContext(ctx, `SELECT deposit_balance FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&deposit)
	return deposit, err
}

func (r *repo) GetBookPrice(ctx context.Context, tx *sql.Tx, bookID int64) (int64, error) {
	var price int64
	err := tx.QueryRowContext(ctx, `SELECT rental_cost FROM books WHERE id=$1`, bookID).Scan(&price)
	return price, err
}

func (r *repo) LockOneAvailableItem(ctx context.Context, tx *sql.Tx, bookID int64) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM book_items
		WHERE book_id=$1 AND status='AVAILABLE'
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, bookID).Scan(&id)
	return id, err
}

func (r *repo) DeductDeposit(ctx context.Context, tx *sql.Tx, userID int64, amount int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE users SET deposit_balance=deposit_balance-$1 WHERE id=$2`, amount, userID)
	return err
}

func (r *repo) ReserveItem(ctx context.Context, tx *sql.Tx, itemID int64, holdUntil *time.Time) error {
	_, err := tx.ExecContext(ctx, `UPDATE book_items SET status='BOOKED', booked_until=$1 WHERE id=$2`, holdUntil, itemID)
	return err
}

func (r *repo) InsertRental(ctx context.Context, tx *sql.Tx, userID, bookID, itemID, price int64, holdUntil *time.Time) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO rentals (user_id, book_id, book_item_id, status, price, booked_until)
		VALUES ($1,$2,$3,'BOOKED',$4,$5)
	`, userID, bookID, itemID, price, holdUntil)
	return err
}
