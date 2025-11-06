package bookrepo

import (
	"context"
	"database/sql"
	"errors"
)

type Book struct {
	ID                int64
	Name              string
	Category          string
	RentalCost        float64
	StockAvailability int64
}

type Repo interface {
	CreateBook(ctx context.Context, name, category string, rentalCost float64) (int64, error)
	AddCopies(ctx context.Context, bookID int64, n int) (int64, error)
	List(ctx context.Context) ([]Book, error)
	Detail(ctx context.Context, id int64) (*Book, error)
}

type repo struct{ db *sql.DB }

func New(db *sql.DB) Repo { return &repo{db} }

func (r *repo) CreateBook(ctx context.Context, name, category string, rentalCost float64) (int64, error) {
	const q = `
INSERT INTO books (name, category, rental_cost)
VALUES ($1,$2,$3)
RETURNING id`
	var id int64
	if err := r.db.QueryRowContext(ctx, q, name, category, rentalCost).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *repo) AddCopies(ctx context.Context, bookID int64, n int) (int64, error) {
	if n <= 0 {
		return 0, errors.New("n must be > 0")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	const ins = `INSERT INTO book_items (book_id, status) VALUES ($1,'AVAILABLE')`
	for i := 0; i < n; i++ {
		if _, err = tx.ExecContext(ctx, ins, bookID); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return int64(n), nil
}

func (r *repo) List(ctx context.Context) ([]Book, error) {

	const q = `
	SELECT b.id, b.name, b.category, b.rental_cost,
		COALESCE(COUNT(bi.*) FILTER (WHERE bi.status='AVAILABLE'),0)::BIGINT AS stock_availability
	FROM books b
	LEFT JOIN book_items bi ON bi.book_id=b.id
	GROUP BY b.id
	ORDER BY b.id DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Name, &b.Category, &b.RentalCost, &b.StockAvailability); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *repo) Detail(ctx context.Context, id int64) (*Book, error) {
	const q = `
SELECT b.id, b.name, b.category, b.rental_cost,
       COALESCE(COUNT(bi.*) FILTER (WHERE bi.status='AVAILABLE'),0)::BIGINT AS stock_availability
FROM books b
LEFT JOIN book_items bi ON bi.book_id=b.id
WHERE b.id=$1
GROUP BY b.id`
	var b Book
	if err := r.db.QueryRowContext(ctx, q, id).Scan(&b.ID, &b.Name, &b.Category, &b.RentalCost, &b.StockAvailability); err != nil {
		return nil, err
	}
	return &b, nil
}
