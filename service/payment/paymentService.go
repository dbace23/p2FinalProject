package paymentsvc

import (
	rentalrepo "bookrental/repository/rental"
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
	rRepo rentalrepo.Repo
}

func New(db *sql.DB, xv xenditrepo.Repo, w walletrepo.Repo, r rentalrepo.Repo) Service {
	return &service{db: db, xv: xv, wRepo: w, rRepo: r}
}

type xInvoiceEvent struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	ExternalID string `json:"external_id"`
}

func (s *service) HandleXendit(ctx context.Context, sigHeader string, raw []byte) error {

	var ev xInvoiceEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return fmt.Errorf("bad webhook json: %w", err)
	}
	if ev.ID == "" || ev.Status == "" {
		return errors.New("missing invoice fields")
	}
	switch ev.Status {
	case "PAID":
		return s.onPaid(ctx, ev)
	case "EXPIRED":

		return nil
	default:
		return nil
	}
}

func (s *service) onPaid(ctx context.Context, ev xInvoiceEvent) error {

	if topupID, userID, amt, status, err := s.wRepo.FindTopupByInvoiceID(ctx, ev.ID); err == nil {
		if status == "PAID" {

			return nil
		}
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
		return tx.Commit()
	}

	rentalID, userID, copyID, cost, status, err := s.rRepo.FindRentalByInvoiceID(ctx, ev.ID)
	if err != nil {
		return fmt.Errorf("invoice not mapped to topup nor rental: %w", err)
	}
	if status == "ACTIVE" || status == "RETURNED" {

		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	cur, err := s.wRepo.GetUserBalanceForUpdate(ctx, tx, userID)
	if err != nil {
		return err
	}
	if cur < cost {

		return fmt.Errorf("insufficient wallet balance for rental: have %.2f need %.2f", cur, cost)
	}
	newBal := cur - cost
	if err = s.wRepo.UpdateUserBalance(ctx, tx, userID, newBal); err != nil {
		return err
	}
	if err = s.wRepo.InsertLedger(ctx, tx, userID, "rentals", &rentalID, "RENTAL_CHARGE", -cost, newBal); err != nil {
		return err
	}

	if err = s.rRepo.ActivateRental(ctx, tx, rentalID); err != nil {
		return err
	}
	if err = s.rRepo.MarkCopyRented(ctx, tx, copyID); err != nil {
		return err
	}

	return tx.Commit()
}
