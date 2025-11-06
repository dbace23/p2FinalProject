// model/rental.go
package model

import "time"

type RentalStatus string

const (
	RentalBooked   RentalStatus = "BOOKED"
	RentalPaid     RentalStatus = "PAID"
	RentalActive   RentalStatus = "ACTIVE"
	RentalReturned RentalStatus = "RETURNED"
	RentalCanceled RentalStatus = "CANCELED"
)

type Rental struct {
	ID              int64        `json:"id"`
	UserID          int64        `json:"user_id"`
	BookID          int64        `json:"book_id"`
	BookItemID      int64        `json:"book_item_id"`
	Status          RentalStatus `json:"status"`
	RentalCost      float64      `json:"rental_cost"`
	BookedAt        string       `json:"booked_at"`
	PaymentDueAt    time.Time    `json:"payment_due_at"`
	PaidAt          *time.Time   `json:"paid_at,omitempty"`
	ActivatedAt     *time.Time   `json:"activated_at,omitempty"`
	ReturnedAt      *time.Time   `json:"returned_at,omitempty"`
	CanceledAt      *time.Time   `json:"canceled_at,omitempty"`
	XenditInvoiceID *string      `json:"xendit_invoice_id,omitempty"`
	Notes           *string      `json:"notes,omitempty"`
}
