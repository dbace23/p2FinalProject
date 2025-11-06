package walletrepo

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type LedgerRow struct {
	ID           int64
	EntryType    string
	Amount       float64
	BalanceAfter float64
	CreatedAt    time.Time
}

type Repo interface {
	InsertTopup(ctx context.Context, tx *sql.Tx, userID int64, amount float64, invID, link, expires string) (int64, error)
	ListLedger(ctx context.Context, userID int64) ([]LedgerRow, error)

	FindTopupByInvoiceID(ctx context.Context, invoiceID string) (topupID int64, userID int64, amount float64, status string, err error)
	MarkTopupPaidAndCredit(ctx context.Context, tx *sql.Tx, topupID, userID int64, amount float64) error

	GetUserBalanceForUpdate(ctx context.Context, tx *sql.Tx, userID int64) (float64, error)
	UpdateUserBalance(ctx context.Context, tx *sql.Tx, userID int64, newBalance float64) error
	InsertLedger(ctx context.Context, tx *sql.Tx, userID int64, refTable string, refID *int64, entryType string, amount float64, balanceAfter float64) error
}

type repo struct{ db *sql.DB }

func New(db *sql.DB) Repo { return &repo{db} }

func (r *repo) InsertTopup(ctx context.Context, tx *sql.Tx, userID int64, amount float64, invID, link, expires string) (int64, error) {
	const q = `
INSERT INTO wallet_topups (user_id, amount, status, xendit_invoice_id, payment_link, expires_at)
VALUES ($1,$2,'PENDING',$3,$4,$5)
RETURNING id`
	var id int64
	if err := tx.QueryRowContext(ctx, q, userID, amount, invID, link, expires).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *repo) ListLedger(ctx context.Context, userID int64) ([]LedgerRow, error) {
	const q = `
SELECT id, entry_type, amount, balance_after, created_at
FROM wallet_ledger
WHERE user_id=$1
ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LedgerRow
	for rows.Next() {
		var l LedgerRow
		if err := rows.Scan(&l.ID, &l.EntryType, &l.Amount, &l.BalanceAfter, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *repo) FindTopupByInvoiceID(ctx context.Context, invoiceID string) (int64, int64, float64, string, error) {
	const q = `
SELECT id, user_id, amount, status
FROM wallet_topups
WHERE xendit_invoice_id=$1`
	var id, uid int64
	var amt float64
	var status string
	err := r.db.QueryRowContext(ctx, q, invoiceID).Scan(&id, &uid, &amt, &status)
	return id, uid, amt, status, err
}

func (r *repo) MarkTopupPaidAndCredit(ctx context.Context, tx *sql.Tx, topupID, userID int64, amount float64) error {
	//mark topup as PAID
	const q1 = `
	UPDATE wallet_topups
	SET status='PAID', paid_at=NOW()
	WHERE id=$1 AND status='PENDING'`
	res, err := tx.ExecContext(ctx, q1, topupID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("topup not pending or not found")
	}

	//update user balance (credit)
	var current float64
	const qBal = `SELECT deposit_balance FROM users WHERE id=$1 FOR UPDATE`
	if err := tx.QueryRowContext(ctx, qBal, userID).Scan(&current); err != nil {
		return err
	}
	newBal := current + amount

	const qUp = `UPDATE users SET deposit_balance=$2 WHERE id=$1`
	if _, err := tx.ExecContext(ctx, qUp, userID, newBal); err != nil {
		return err
	}

	// ledger entry
	return r.InsertLedger(ctx, tx, userID, "wallet_topups", &topupID, "TOPUP_CONFIRMED", amount, newBal)
}

func (r *repo) GetUserBalanceForUpdate(ctx context.Context, tx *sql.Tx, userID int64) (float64, error) {
	const q = `SELECT deposit_balance FROM users WHERE id=$1 FOR UPDATE`
	var bal float64
	if err := tx.QueryRowContext(ctx, q, userID).Scan(&bal); err != nil {
		return 0, err
	}
	return bal, nil
}

func (r *repo) UpdateUserBalance(ctx context.Context, tx *sql.Tx, userID int64, newBalance float64) error {
	const q = `UPDATE users SET deposit_balance=$2 WHERE id=$1`
	_, err := tx.ExecContext(ctx, q, userID, newBalance)
	return err
}

func (r *repo) InsertLedger(ctx context.Context, tx *sql.Tx, userID int64, refTable string, refID *int64, entryType string, amount float64, balanceAfter float64) error {
	const q = `
INSERT INTO wallet_ledger (user_id, ref_table, ref_id, entry_type, amount, balance_after)
VALUES ($1,$2,$3,$4,$5,$6)`
	_, err := tx.ExecContext(ctx, q, userID, refTable, refID, entryType, amount, balanceAfter)
	return err
}
