package xenditrepo

import "time"

type CreateInvoiceReq struct {
	ExternalID  string
	Amount      float64
	PayerEmail  string
	Description string
	ExpirySec   int
}

type CreateInvoiceResp struct {
	InvoiceID  string
	InvoiceURL string
	ExpiresAt  string
}

type Invoice struct {
	ID         string     `json:"id"`
	ExternalID string     `json:"external_id"`
	Status     string     `json:"status"`
	Amount     float64    `json:"amount"`
	PaidAmount float64    `json:"paid_amount"`
	InvoiceURL string     `json:"invoice_url"`
	ExpiryDate time.Time  `json:"expiry_date"`
	PaidAt     *time.Time `json:"paid_at,omitempty"`
}

type Repo interface {
	CreateInvoice(req CreateInvoiceReq) (*CreateInvoiceResp, error)
	VerifyCallbackSignature(sigHeader string, rawBody []byte) error
}
