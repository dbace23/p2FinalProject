package paymentsvc

import (
	walletrepo "bookrental/repository/wallet"
	xenditrepo "bookrental/repository/xendit"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type Service interface {
	HandleXendit(ctx context.Context, sigHeader string, raw []byte) error
}

type service struct {
	db    *sql.DB
	xv    xenditrepo.Repo
	wRepo walletrepo.Repo
}

func New(db *sql.DB, xv xenditrepo.Repo, w walletrepo.Repo) Service {
	return &service{db: db, xv: xv, wRepo: w}
}

type xInvoiceEvent struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	ExternalID string `json:"external_id"`
}

func (s *service) HandleXendit(ctx context.Context, sigHeader string, raw []byte) error {
	// Verify HMAC signature from Xendit
	if err := s.xv.VerifyCallbackSignature(sigHeader, raw); err != nil {
		return fmt.Errorf("invalid callback signature: %w", err)
	}

	// Parse JSON
	var ev xInvoiceEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return fmt.Errorf("bad webhook json: %w", err)
	}
	if ev.ID == "" || ev.Status == "" {
		return errors.New("missing invoice fields")
	}

	//  Handle event types
	switch ev.Status {
	case "PAID":
		return s.onTopupPaid(ctx, ev.ID)
	case "EXPIRED":

		return nil
	default:

		return nil
	}
}

func (s *service) onTopupPaid(ctx context.Context, invoiceID string) (err error) {

	topupID, userID, amt, status, err := s.wRepo.FindTopupByInvoiceID(ctx, invoiceID)
	if err != nil {

		return nil
	}

	if status == "PAID" {
		return nil
	}

	// tx: mark paid + credit balance + ledger
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = s.wRepo.MarkTopupPaidAndCredit(ctx, tx, topupID, userID, amt); err != nil {
		return err
	}

	// Commit
	return tx.Commit()
}
