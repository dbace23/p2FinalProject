package wallet

import (
	wrepo "bookrental/repository/wallet"
	xenditrepo "bookrental/repository/xendit"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type LedgerRow = wrepo.LedgerRow

type Service interface {
	CreateTopup(ctx context.Context, userID int64, amount float64) (*TopupCreated, error)
	Ledger(ctx context.Context, userID int64) ([]LedgerRow, error)
}

type TopupCreated struct {
	InvoiceID, PaymentLink, ExpiresAt string
}

type Repo interface {
	InsertTopup(ctx context.Context, tx *sql.Tx, userID int64, amount float64, invID, link, expires string) (int64, error)
	ListLedger(ctx context.Context, userID int64) ([]LedgerRow, error)
}

type service struct {
	db *sql.DB
	r  Repo
	x  xenditrepo.Repo
}

func New(db *sql.DB, r Repo, x xenditrepo.Repo) Service { return &service{db: db, r: r, x: x} }

func (s *service) CreateTopup(ctx context.Context, userID int64, amount float64) (*TopupCreated, error) {
	if amount <= 0 {
		return nil, errors.New("invalid amount")
	}
	iv, err := s.x.CreateInvoice(xenditrepo.CreateInvoiceReq{
		ExternalID:  fmt.Sprintf("topup:%d:%d", userID, time.Now().UnixNano()),
		Amount:      amount,
		Description: "Wallet top-up",
		ExpirySec:   3600,
	})
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = s.r.InsertTopup(ctx, tx, userID, amount, iv.InvoiceID, iv.InvoiceURL, iv.ExpiresAt); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &TopupCreated{InvoiceID: iv.InvoiceID, PaymentLink: iv.InvoiceURL, ExpiresAt: iv.ExpiresAt}, nil
}

func (s *service) Ledger(ctx context.Context, userID int64) ([]LedgerRow, error) {
	return s.r.ListLedger(ctx, userID)
}
