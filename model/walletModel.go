// model/wallet.go
package model

import "time"

type TopupStatus string

const (
	TopupPending TopupStatus = "PENDING"
	TopupPaid    TopupStatus = "PAID"
	TopupExpired TopupStatus = "EXPIRED"
	TopupFailed  TopupStatus = "FAILED"
)

type WalletTopup struct {
	ID              int64       `json:"id"`
	UserID          int64       `json:"user_id"`
	Amount          float64     `json:"amount"`
	Status          TopupStatus `json:"status"`
	XenditInvoiceID *string     `json:"xendit_invoice_id,omitempty"`
	PaymentLink     *string     `json:"payment_link,omitempty"`
	ExpiresAt       *time.Time  `json:"expires_at,omitempty"`
	PaidAt          *time.Time  `json:"paid_at,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
}

type LedgerType string

const (
	LedgerTopup  LedgerType = "TOPUP_CONFIRMED"
	LedgerCharge LedgerType = "RENTAL_CHARGE"
	LedgerRefund LedgerType = "RENTAL_REFUND"
	LedgerAdjust LedgerType = "ADJUSTMENT"
)

type WalletLedger struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	RefTable     string     `json:"ref_table"`
	RefID        *int64     `json:"ref_id,omitempty"`
	EntryType    LedgerType `json:"entry_type"`
	Amount       float64    `json:"amount"`
	BalanceAfter float64    `json:"balance_after"`
	CreatedAt    time.Time  `json:"created_at"`
}
