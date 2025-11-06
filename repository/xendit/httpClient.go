package xenditrepo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type httpRepo struct {
	apiKey string
	client *http.Client
}

func NewHTTP(apiKey string) Repo {
	return &httpRepo{
		apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *httpRepo) CreateInvoice(req CreateInvoiceReq) (*CreateInvoiceResp, error) {

	body := map[string]any{
		"external_id":      req.ExternalID,
		"amount":           req.Amount,
		"description":      req.Description,
		"payer_email":      req.PayerEmail,
		"invoice_duration": req.ExpirySec,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	httpReq, err := http.NewRequest("POST", "https://api.xendit.co/v2/invoices", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.SetBasicAuth(r.apiKey, "")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bs, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("xendit create invoice failed: %s: %s", resp.Status, string(bs))
	}

	var out struct {
		ID         string `json:"id"`
		InvoiceURL string `json:"invoice_url"`
		ExpiryDate string `json:"expiry_date"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.ID == "" {
		return nil, errors.New("xendit: empty invoice id")
	}

	return &CreateInvoiceResp{
		InvoiceID:  out.ID,
		InvoiceURL: out.InvoiceURL,
		ExpiresAt:  out.ExpiryDate,
	}, nil
}

func (r *httpRepo) VerifyCallbackSignature(sigHeader string, rawBody []byte) error {
	if sigHeader != os.Getenv("XENDIT_CALLBACK_TOKEN") {
		return errors.New("bad token")
	}
	return nil
}
