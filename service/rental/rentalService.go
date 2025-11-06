package rental

import (
	rrepo "bookrental/repository/rental"
	xenditrepo "bookrental/repository/xendit"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// errors used by controllers

type ErrCode string

const (
	ErrNoStock      ErrCode = "NO_STOCK"
	ErrBookNotFound ErrCode = "BOOK_NOT_FOUND"
	ErrNotOwner     ErrCode = "NOT_OWNER"
	ErrNotActive    ErrCode = "NOT_ACTIVE"
	ErrNotFound     ErrCode = "NOT_FOUND"
)

type codedError struct{ code ErrCode }

func (e codedError) Error() string { return string(e.code) }
func (e codedError) Code() ErrCode { return e.code }
func makeErr(c ErrCode) error      { return codedError{code: c} }

// Code extracts error code
func Code(err error) ErrCode {
	var ce interface{ Code() ErrCode }
	if errors.As(err, &ce) {
		return ce.Code()
	}
	return ""
}

// dto

type Created struct {
	RentalID     int64
	PaymentLink  string
	PaymentDueAt string
}

// HistoryRow = repository shape
type HistoryRow = rrepo.HistoryRow

type Repo interface {
	CheckBookExists(ctx context.Context, bookID int64) (bool, error)
	GetBookRentalCost(ctx context.Context, bookID int64) (float64, error)

	PickAvailableCopyForUpdate(ctx context.Context, tx *sql.Tx, bookID int64) (int64, error)
	MarkCopyBooked(ctx context.Context, tx *sql.Tx, copyID int64, due time.Time) error
	InsertRental(ctx context.Context, tx *sql.Tx, userID, bookID, copyID int64, cost float64, due time.Time, invID, link, expires string) (int64, error)

	GetRentalOwnerAndStatus(ctx context.Context, tx *sql.Tx, rentalID int64) (ownerID int64, status string, copyID int64, err error)
	MarkReturned(ctx context.Context, tx *sql.Tx, rentalID int64) error
	FreeCopy(ctx context.Context, tx *sql.Tx, copyID int64) error

	ListMyRentals(ctx context.Context, userID int64) ([]HistoryRow, error)

	FindRentalByInvoiceID(ctx context.Context, invoiceID string) (rentalID, userID, copyID int64, cost float64, status string, err error)
	ActivateRental(ctx context.Context, tx *sql.Tx, rentalID int64) error
	MarkCopyRented(ctx context.Context, tx *sql.Tx, copyID int64) error
}

type Service interface {
	// Create: book a copy and generate an invoice (status BOOKED).
	Create(ctx context.Context, userID, bookID int64) (*Created, error)

	// Return: mark ACTIVE rental returned and freeing the book
	Return(ctx context.Context, userID, rentalID int64) error

	// MyHistory: list rentals for a user.
	MyHistory(ctx context.Context, userID int64) ([]HistoryRow, error)
}

// ----- Service implementation -----

type service struct {
	db *sql.DB
	r  Repo
	x  xenditrepo.Repo
}

func New(db *sql.DB, r Repo, x xenditrepo.Repo) Service {
	return &service{db: db, r: r, x: x}
}

// Create books a copy (24h hold) and creates a Xendit invoice. when xendit give post
func (s *service) Create(ctx context.Context, userID, bookID int64) (*Created, error) {
	// check book exist
	exists, err := s.r.CheckBookExists(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, makeErr(ErrBookNotFound)
	}

	//Get price
	cost, err := s.r.GetBookRentalCost(ctx, bookID)
	if err != nil {
		return nil, err
	}

	// Prepare invoice
	exp := 24 * time.Hour
	due := time.Now().UTC().Add(exp)
	inv, err := s.x.CreateInvoice(xenditrepo.CreateInvoiceReq{
		ExternalID:  fmt.Sprintf("rental:%d:%d", userID, time.Now().UnixNano()),
		Amount:      cost,
		Description: "Book rental",
		ExpirySec:   int(exp.Seconds()),
	})
	if err != nil {
		return nil, err
	}

	// Book a copy
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	copyID, err := s.r.PickAvailableCopyForUpdate(ctx, tx, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, makeErr(ErrNoStock)
		}
		return nil, err
	}

	if err = s.r.MarkCopyBooked(ctx, tx, copyID, due); err != nil {
		return nil, err
	}

	rentalID, err := s.r.InsertRental(ctx, tx, userID, bookID, copyID, cost, due, inv.InvoiceID, inv.InvoiceURL, inv.ExpiresAt)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &Created{
		RentalID:     rentalID,
		PaymentLink:  inv.InvoiceURL,
		PaymentDueAt: due.Format(time.RFC3339),
	}, nil
}

// Return and frees the copy.
func (s *service) Return(ctx context.Context, userID, rentalID int64) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	owner, status, copyID, err := s.r.GetRentalOwnerAndStatus(ctx, tx, rentalID)
	if err != nil {
		return err
	}
	if owner != userID {
		return makeErr(ErrNotOwner)
	}
	if status != "ACTIVE" {
		return makeErr(ErrNotActive)
	}

	if err = s.r.MarkReturned(ctx, tx, rentalID); err != nil {
		return err
	}
	if err = s.r.FreeCopy(ctx, tx, copyID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *service) MyHistory(ctx context.Context, userID int64) ([]HistoryRow, error) {
	return s.r.ListMyRentals(ctx, userID)
}
