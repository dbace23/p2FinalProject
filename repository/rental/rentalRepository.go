package rentalrepo

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type HistoryRow struct {
	RentalID   int64
	BookName   string
	Status     string
	BookedAt   time.Time
	PaidAt     sql.NullTime
	ReturnedAt sql.NullTime
}

type Repo interface {
	CheckBookExists(ctx context.Context, bookID int64) (bool, error)
	GetBookRentalCost(ctx context.Context, bookID int64) (float64, error)

	PickAvailableCopyForUpdate(ctx context.Context, tx *sql.Tx, bookID int64) (copyID int64, err error)
	MarkCopyBooked(ctx context.Context, tx *sql.Tx, copyID int64, due time.Time) error

	InsertRental(ctx context.Context, tx *sql.Tx, userID, bookID, copyID int64, cost float64, due time.Time, xInvoiceID, link, expires string) (int64, error)

	GetRentalOwnerAndStatus(ctx context.Context, tx *sql.Tx, rentalID int64) (ownerID int64, status string, copyID int64, err error)
	MarkReturned(ctx context.Context, tx *sql.Tx, rentalID int64) error
	FreeCopy(ctx context.Context, tx *sql.Tx, copyID int64) error

	ListMyRentals(ctx context.Context, userID int64) ([]HistoryRow, error)

	FindRentalByInvoiceID(ctx context.Context, invoiceID string) (rentalID int64, userID int64, bookItemID int64, cost float64, status string, err error)
	ActivateRental(ctx context.Context, tx *sql.Tx, rentalID int64) error
	MarkCopyRented(ctx context.Context, tx *sql.Tx, copyID int64) error
	ReleaseExpiredBookings(ctx context.Context, now time.Time) (released int64, err error)
}

type repo struct{ db *sql.DB }

func New(db *sql.DB) Repo { return &repo{db} }

func (r *repo) CheckBookExists(ctx context.Context, bookID int64) (bool, error) {
	const q = `SELECT 1 FROM books WHERE id=$1`
	var x int
	err := r.db.QueryRowContext(ctx, q, bookID).Scan(&x)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *repo) GetBookRentalCost(ctx context.Context, bookID int64) (float64, error) {
	const q = `SELECT rental_cost FROM books WHERE id=$1`
	var cost float64
	if err := r.db.QueryRowContext(ctx, q, bookID).Scan(&cost); err != nil {
		return 0, err
	}
	return cost, nil
}

func (r *repo) PickAvailableCopyForUpdate(ctx context.Context, tx *sql.Tx, bookID int64) (int64, error) {
	const q = `
SELECT id
FROM book_items
WHERE book_id=$1 AND status='AVAILABLE'
FOR UPDATE SKIP LOCKED
LIMIT 1`
	var id int64
	err := tx.QueryRowContext(ctx, q, bookID).Scan(&id)
	return id, err
}

func (r *repo) MarkCopyBooked(ctx context.Context, tx *sql.Tx, copyID int64, due time.Time) error {
	const q = `
UPDATE book_items
SET status='BOOKED', booked_until=$2
WHERE id=$1 AND status='AVAILABLE'`
	res, err := tx.ExecContext(ctx, q, copyID, due)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *repo) InsertRental(ctx context.Context, tx *sql.Tx, userID, bookID, copyID int64, cost float64, due time.Time, xInvoiceID, link, expires string) (int64, error) {
	const q = `
INSERT INTO rentals (user_id, book_id, book_item_id, status, rental_cost, payment_due_at, xendit_invoice_id)
VALUES ($1,$2,$3,'BOOKED',$4,$5,$6)
RETURNING id`
	var id int64
	if err := tx.QueryRowContext(ctx, q, userID, bookID, copyID, cost, due, xInvoiceID).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (r *repo) GetRentalOwnerAndStatus(ctx context.Context, tx *sql.Tx, rentalID int64) (int64, string, int64, error) {
	const q = `
SELECT user_id, status, book_item_id
FROM rentals
WHERE id=$1`
	var u int64
	var s string
	var copyID int64
	err := tx.QueryRowContext(ctx, q, rentalID).Scan(&u, &s, &copyID)
	return u, s, copyID, err
}

func (r *repo) MarkReturned(ctx context.Context, tx *sql.Tx, rentalID int64) error {
	const q = `
UPDATE rentals
SET status='RETURNED', returned_at=NOW()
WHERE id=$1 AND status='ACTIVE'`
	res, err := tx.ExecContext(ctx, q, rentalID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *repo) FreeCopy(ctx context.Context, tx *sql.Tx, copyID int64) error {
	const q = `
UPDATE book_items
SET status='AVAILABLE', booked_until=NULL
WHERE id=$1`
	_, err := tx.ExecContext(ctx, q, copyID)
	return err
}

func (r *repo) ListMyRentals(ctx context.Context, userID int64) ([]HistoryRow, error) {
	const q = `
SELECT r.id, b.name, r.status, r.booked_at, r.paid_at, r.returned_at
FROM rentals r
JOIN books b ON b.id = r.book_id
WHERE r.user_id=$1
ORDER BY r.booked_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HistoryRow
	for rows.Next() {
		var h HistoryRow
		if err := rows.Scan(&h.RentalID, &h.BookName, &h.Status, &h.BookedAt, &h.PaidAt, &h.ReturnedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *repo) FindRentalByInvoiceID(ctx context.Context, invoiceID string) (int64, int64, int64, float64, string, error) {
	const q = `
SELECT id, user_id, book_item_id, rental_cost, status
FROM rentals
WHERE xendit_invoice_id=$1`
	var id, uid, copyID int64
	var cost float64
	var status string
	err := r.db.QueryRowContext(ctx, q, invoiceID).Scan(&id, &uid, &copyID, &cost, &status)
	return id, uid, copyID, cost, status, err
}

func (r *repo) ActivateRental(ctx context.Context, tx *sql.Tx, rentalID int64) error {
	const q = `
UPDATE rentals
SET status='ACTIVE', paid_at=NOW(), activated_at=NOW()
WHERE id=$1 AND status IN ('BOOKED','PAID')`
	res, err := tx.ExecContext(ctx, q, rentalID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *repo) MarkCopyRented(ctx context.Context, tx *sql.Tx, copyID int64) error {
	const q = `
UPDATE book_items
SET status='RENTED', booked_until=NULL
WHERE id=$1`
	_, err := tx.ExecContext(ctx, q, copyID)
	return err
}

func (r *repo) ReleaseExpiredBookings(ctx context.Context, now time.Time) (int64, error) {
	// Cancel rentals first
	const q1 = `
UPDATE rentals
SET status='CANCELED', canceled_at=NOW()
WHERE status='BOOKED' AND payment_due_at < $1`
	res1, err := r.db.ExecContext(ctx, q1, now)
	if err != nil {
		return 0, err
	}
	n1, _ := res1.RowsAffected()

	// Free copies
	const q2 = `
UPDATE book_items
SET status='AVAILABLE', booked_until=NULL
WHERE status='BOOKED' AND booked_until < $1`
	res2, err := r.db.ExecContext(ctx, q2, now)
	if err != nil {
		return n1, err
	}
	n2, _ := res2.RowsAffected()

	return n1 + n2, nil
}
