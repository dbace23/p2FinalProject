package xenditrepo

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

type Repo interface {
	CreateInvoice(req CreateInvoiceReq) (*CreateInvoiceResp, error)
	VerifyCallbackSignature(sigHeader string, rawBody []byte) error
}
