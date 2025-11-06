package xenditrepo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type httpRepo struct {
	apiKey string
	client *http.Client
}

func NewHTTP(apiKey string) Repo { return &httpRepo{apiKey: apiKey, client: &http.Client{}} }

func (r *httpRepo) CreateInvoice(req CreateInvoiceReq) (*CreateInvoiceResp, error) {
	body := map[string]any{
		"external_id":      req.ExternalID,
		"amount":           req.Amount,
		"description":      req.Description,
		"payer_email":      req.PayerEmail,
		"invoice_duration": req.ExpirySec,
	}
	b, _ := json.Marshal(body)
	httpReq, _ := http.NewRequest("POST", "https://api.xendit.co/v2/invoices", bytes.NewReader(b))
	httpReq.SetBasicAuth(r.apiKey, "")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("xendit create invoice failed: %s", resp.Status)
	}

	var out struct {
		ID, InvoiceURL, ExpiryDate string `json:"id","invoice_url","expiry_date"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.ID == "" {
		return nil, errors.New("xendit: empty invoice id")
	}

	return &CreateInvoiceResp{InvoiceID: out.ID, InvoiceURL: out.InvoiceURL, ExpiresAt: out.ExpiryDate}, nil
}

func (r *httpRepo) VerifyCallbackSignature(sigHeader string, rawBody []byte) error { return nil }
