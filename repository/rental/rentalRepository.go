// repository/rental/repo.go
package rental

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type HistoryRow struct {
	RentalID   int64      `json:"rental_id"`
	BookID     int64      `json:"book_id"`
	BookName   string     `json:"book_name"`
	ItemID     int64      `json:"item_id"`
	Price      float64    `json:"price"`
	Status     string     `json:"status"` // ACTIVE | RETURNED
	CreatedAt  time.Time  `json:"created_at"`
	ReturnedAt *time.Time `json:"returned_at,omitempty"`
}

type Repo interface {
	// User & money
	LockUserForUpdate(ctx context.Context, tx *sql.Tx, userID int64) (deposit float64, err error)
	DeductDeposit(ctx context.Context, tx *sql.Tx, userID int64, amount float64) error

	// Books & items
	GetBookPrice(ctx context.Context, tx *sql.Tx, bookID int64) (price float64, err error)
	LockOneAvailableItem(ctx context.Context, tx *sql.Tx, bookID int64) (itemID int64, err error)
	ReserveItem(ctx context.Context, tx *sql.Tx, itemID int64, holdUntil *time.Time) error

	// Rentals
	InsertRental(ctx context.Context, tx *sql.Tx, userID, bookID, itemID int64, price float64, holdUntil *time.Time) error
	GetRentalOwnerAndStatus(ctx context.Context, tx *sql.Tx, rentalID int64) (ownerID int64, status string, itemID int64, err error)
	MarkReturned(ctx context.Context, tx *sql.Tx, rentalID int64) error
	FreeCopy(ctx context.Context, tx *sql.Tx, itemID int64) error

	// History
	ListMyRentals(ctx context.Context, userID int64) ([]HistoryRow, error)
}

type repo struct {
	db *sql.DB
}

func New(db *sql.DB) Repo { return &repo{db: db} }

// User & money

func (r *repo) LockUserForUpdate(ctx context.Context, tx *sql.Tx, userID int64) (float64, error) {
	const q = `
				SELECT deposit_balance
				FROM users
				WHERE id = $1
				FOR UPDATE`
	var dep float64
	err := tx.QueryRowContext(ctx, q, userID).Scan(&dep)
	return dep, err
}

func (r *repo) DeductDeposit(ctx context.Context, tx *sql.Tx, userID int64, amount float64) error {
	// Guard: only deduct if sufficient.
	const q = `
			UPDATE users
			SET deposit_balance = deposit_balance - $2
			WHERE id = $1
			AND deposit_balance >= $2`
	res, err := tx.ExecContext(ctx, q, userID, amount)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return errors.New("insufficient deposit")
	}
	return nil
}

// Books & items

func (r *repo) GetBookPrice(ctx context.Context, tx *sql.Tx, bookID int64) (float64, error) {
	const q = `
			SELECT price
			FROM books
			WHERE id = $1`
	var price float64
	err := tx.QueryRowContext(ctx, q, bookID).Scan(&price)
	return price, err
}

func (r *repo) LockOneAvailableItem(ctx context.Context, tx *sql.Tx, bookID int64) (int64, error) {
	// Prevent double booking with SKIP LOCKED
	const q = `
				SELECT id
				FROM books
				WHERE book_id = $1
				AND status = 'AVAILABLE'
				ORDER BY id
				FOR UPDATE SKIP LOCKED
				LIMIT 1`
	var itemID int64
	err := tx.QueryRowContext(ctx, q, bookID).Scan(&itemID)
	return itemID, err
}

func (r *repo) ReserveItem(ctx context.Context, tx *sql.Tx, itemID int64, holdUntil *time.Time) error {
	const q = `
	UPDATE book_items
	SET status = 'BOOKED',
		booked_until = $2
	WHERE id = $1`
	_, err := tx.ExecContext(ctx, q, itemID, holdUntil)
	return err
}

// Rentals
func (r *repo) InsertRental(ctx context.Context, tx *sql.Tx, userID, bookID, itemID int64, price float64, holdUntil *time.Time) error {
	//deposit deducted , rental =ACTIVE
	const q = `
		INSERT INTO rentals (user_id, book_id, item_id, price, status, hold_until)
		VALUES ($1, $2, $3, $4, 'ACTIVE', $5)`
	_, err := tx.ExecContext(ctx, q, userID, bookID, itemID, price, holdUntil)
	if err != nil {
		return err
	}
	// mark item as RENTED
	const q2 = `
		UPDATE book_items
		SET status = 'RENTED'
		WHERE id = $1`
	_, err = tx.ExecContext(ctx, q2, itemID)
	return err
}

func (r *repo) GetRentalOwnerAndStatus(ctx context.Context, tx *sql.Tx, rentalID int64) (int64, string, int64, error) {
	const q = `
		SELECT user_id, status, book_id
		FROM rentals
		WHERE id = $1
		FOR UPDATE`
	var uid int64
	var status string
	var itemID int64
	err := tx.QueryRowContext(ctx, q, rentalID).Scan(&uid, &status, &itemID)
	return uid, status, itemID, err
}

func (r *repo) MarkReturned(ctx context.Context, tx *sql.Tx, rentalID int64) error {
	const q = `
		UPDATE rentals
		SET status = 'RETURNED',
			returned_at = NOW()
		WHERE id = $1`
	_, err := tx.ExecContext(ctx, q, rentalID)
	return err
}

func (r *repo) FreeCopy(ctx context.Context, tx *sql.Tx, itemID int64) error {
	const q = `
		UPDATE book_items
		SET status = 'AVAILABLE',
			booked_until = NULL
		WHERE id = $1`
	_, err := tx.ExecContext(ctx, q, itemID)
	return err
}

// History

func (r *repo) ListMyRentals(ctx context.Context, userID int64) ([]HistoryRow, error) {
	const q = `
			SELECT
			r.id          AS rental_id,
			r.book_id     AS book_id,
			b.name        AS book_name,
			r.item_id     AS item_id,
			r.price       AS price,
			r.status      AS status,
			r.created_at  AS created_at,
			r.returned_at AS returned_at
			FROM rentals r
			JOIN books b ON b.id = r.book_id
			WHERE r.user_id = $1
			ORDER BY r.created_at DESC, r.id DESC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HistoryRow
	for rows.Next() {
		var h HistoryRow
		if err := rows.Scan(
			&h.RentalID, &h.BookID, &h.BookName, &h.ItemID,
			&h.Price, &h.Status, &h.CreatedAt, &h.ReturnedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
func (r *repo) InsertRentalReturningID(ctx context.Context, tx *sql.Tx, userID, bookID int64, price float64) (int64, error) {
	const q = `
		INSERT INTO public.rentals (user_id, book_id, status, price)
		VALUES ($1, $2, 'ACTIVE', $3)
		RETURNING id`
	var id int64
	if err := tx.QueryRowContext(ctx, q, userID, bookID, price).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
