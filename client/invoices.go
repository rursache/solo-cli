package client

import (
	"errors"
	"fmt"
	"slices"
	"time"
	_ "time/tzdata" // Europe/Bucharest must resolve on systems without zoneinfo
)

// InvoiceDetail is an issued invoice with its line items
type InvoiceDetail struct {
	UniqueCode         string              `json:"UniqueCode"`
	SerialCode         string              `json:"SerialCode"`
	ClientName         string              `json:"ClientName"`
	IssueDate          string              `json:"IssueDate"`
	DueDate            string              `json:"DueDate"`
	PaymentDate        string              `json:"PaymentDate"`
	IsPaid             bool                `json:"IsPaid"`
	IsExternalDocument bool                `json:"IsExternalDocument"`
	IsLocked           bool                `json:"IsLocked"`
	Amount             float64             `json:"Amount"`
	VAT                float64             `json:"VAT"`
	Total              float64             `json:"Total"`
	Currency           Currency            `json:"Currency"`
	InvoiceLocalAmount *LocalAmount        `json:"InvoiceLocalAmount"`
	Status             *InvoiceStatus      `json:"Status"`
	EInvoiceStatus     *EInvoiceStatus     `json:"EInvoiceStatus"`
	DocumentCode       *string             `json:"DocumentCode"`
	LineItems          []InvoiceDetailLine `json:"LineItems"`
}

// InvoiceDetailLine is an issued line item with server computed amounts
type InvoiceDetailLine struct {
	ItemName      string           `json:"ItemName"`
	Quantity      float64          `json:"Quantity"`
	UnitPrice     float64          `json:"UnitPrice"`
	MeasureUnit   *InvoiceLineUnit `json:"MeasureUnit"`
	Amount        float64          `json:"Amount"`
	VatPercentage float64          `json:"VatPercentage"`
	VAT           float64          `json:"VAT"`
	Total         float64          `json:"Total"`
}

// InvoiceLineUnit is the measure unit of a line item
type InvoiceLineUnit struct {
	Id        int    `json:"Id"`
	Code      string `json:"Code"`
	LabelRo   string `json:"LabelRo"`
	LabelEn   string `json:"LabelEn"`
	IsDefault bool   `json:"IsDefault"`
}

// Invoice download formats from download-options
const (
	InvoiceFormatPDF      = "InternalPdf"
	InvoiceFormatXML      = "InternalXml"
	InvoiceFormatOriginal = "ExternalFile"
)

// InvoiceDownloadOptions lists which files exist for an invoice
type InvoiceDownloadOptions struct {
	InvoiceCode        string   `json:"InvoiceCode"`
	IsExternalDocument bool     `json:"IsExternalDocument"`
	DocumentCode       *string  `json:"DocumentCode"`
	AvailableFormats   []string `json:"AvailableFormats"`
}

// Has reports whether format is offered for the invoice
func (o *InvoiceDownloadOptions) Has(format string) bool {
	return slices.Contains(o.AvailableFormats, format)
}

// InvoicePreCancel says whether an invoice can be deleted. Only the newest
// issued invoice is Deletable, older ones answer Voidable
type InvoicePreCancel struct {
	Status           string `json:"Status"`
	InvoiceIssueYear *int   `json:"InvoiceIssueYear"`
}

// Deletable reports whether CancelInvoice may be called
func (p *InvoicePreCancel) Deletable() bool {
	return p.Status == "Deletable"
}

// InvoicePaymentState is the payment state after mark-paid
type InvoicePaymentState struct {
	IsPaid      bool   `json:"IsPaid"`
	PaymentDate string `json:"PaymentDate"`
}

var (
	ErrInvoiceNotDeletable = errors.New("only the last issued invoice can be deleted (Doar ultima factură emisă poate fi ștearsă)")
	ErrInvalidPaymentDate  = errors.New("the payment date is before the invoice issue date (Data încasării trebuie să fie ulterioară datei emiterii)")
)

const invoiceChangeFailed = "the changes could not be saved (Modificările nu au putut fi efectuate)"

// ValidCode reports whether an invoice or document code is safe to put in a
// URL path segment
func ValidCode(code string) bool {
	if code == "" || code == "." || code == ".." {
		return false
	}
	for _, r := range code {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.'
		if !ok {
			return false
		}
	}
	return true
}

func invoicePath(prefix, code string) (string, error) {
	if !ValidCode(code) {
		return "", fmt.Errorf("invalid code %q", code)
	}
	return prefix + code, nil
}

// GetInvoice loads an issued invoice with its line items
func (c *Client) GetInvoice(code string) (*InvoiceDetail, error) {
	path, err := invoicePath("/proxy/accounting/invoices/load/", code)
	if err != nil {
		return nil, err
	}
	var result InvoiceDetail
	if err := c.doJSON("GET", path, "/invoices/"+code+"/view", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to load invoice: %w", err)
	}
	return &result, nil
}

// GetInvoiceDownloadOptions fetches which files exist for an invoice
func (c *Client) GetInvoiceDownloadOptions(code string) (*InvoiceDownloadOptions, error) {
	path, err := invoicePath("/proxy/accounting/invoices/download-options/", code)
	if err != nil {
		return nil, err
	}
	var result InvoiceDownloadOptions
	if err := c.doJSON("GET", path, "/revenues", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get invoice download options: %w", err)
	}
	return &result, nil
}

// DownloadInvoicePDF downloads the PDF of an invoice issued in SOLO
func (c *Client) DownloadInvoicePDF(code string) (*DownloadedFile, error) {
	return c.downloadInvoiceFile("/proxy/accounting/invoices/download/", code)
}

// DownloadInvoiceXML downloads the e-Factura XML of an invoice
func (c *Client) DownloadInvoiceXML(code string) (*DownloadedFile, error) {
	return c.downloadInvoiceFile("/proxy/accounting/e-invoice/download/", code)
}

// DownloadInvoiceOriginal downloads the uploaded original of an external
// invoice by its DocumentCode (from download-options), not its UniqueCode
func (c *Client) DownloadInvoiceOriginal(documentCode string) (*DownloadedFile, error) {
	return c.downloadInvoiceFile("/api/financial-documents/download/", documentCode)
}

func (c *Client) downloadInvoiceFile(prefix, code string) (*DownloadedFile, error) {
	path, err := invoicePath(prefix, code)
	if err != nil {
		return nil, err
	}
	f, err := c.download(path, "/revenues")
	if err != nil {
		return nil, fmt.Errorf("failed to download invoice file: %w", err)
	}
	return f, nil
}

// MarkInvoicePaid records the payment of an invoice on the civil date of day
// (its year, month and day, whatever its location)
func (c *Client) MarkInvoicePaid(code string, day time.Time) (*InvoicePaymentState, error) {
	path, err := invoicePath("/proxy/accounting/invoices/mark-paid/", code)
	if err != nil {
		return nil, err
	}
	date, err := InvoiceAPIDate(day)
	if err != nil {
		return nil, err
	}
	state := InvoicePaymentState{IsPaid: true, PaymentDate: date}
	// The body is a bare JSON string, not an object
	if err := c.doOk("POST", path, "/revenues", date, &state); err != nil {
		return nil, fmt.Errorf("failed to mark invoice paid: %w", invoiceActionError(err))
	}
	return &state, nil
}

// MarkInvoiceUnpaid clears the payment of an invoice
func (c *Client) MarkInvoiceUnpaid(code string) error {
	path, err := invoicePath("/proxy/accounting/invoices/mark-unpaid/", code)
	if err != nil {
		return err
	}
	if err := c.doOk("POST", path, "/revenues", struct{}{}, nil); err != nil {
		return fmt.Errorf("failed to mark invoice unpaid: %w", invoiceActionError(err))
	}
	return nil
}

// PreCancelInvoice checks whether an invoice can be deleted
func (c *Client) PreCancelInvoice(code string) (*InvoicePreCancel, error) {
	path, err := invoicePath("/proxy/accounting/invoices/pre-cancel/", code)
	if err != nil {
		return nil, err
	}
	var result InvoicePreCancel
	if err := c.doJSON("GET", path, "/revenues", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to check invoice deletion: %w", err)
	}
	return &result, nil
}

// CancelInvoice permanently deletes an issued invoice. Call PreCancelInvoice
// first, SOLO only deletes the newest invoice
func (c *Client) CancelInvoice(code string) error {
	path, err := invoicePath("/proxy/accounting/invoices/cancel/", code)
	if err != nil {
		return err
	}
	var result StatusResponse
	if err := c.doJSON("POST", path, "/revenues", struct{}{}, &result); err != nil {
		return fmt.Errorf("failed to delete invoice: %w", err)
	}
	if result.Status != "Deleted" {
		return fmt.Errorf("SOLO.ro did not delete the invoice (status %q)", result.Status)
	}
	return nil
}

// invoiceActionError maps the known ErrorCode values of the payment
// endpoints to readable errors
func invoiceActionError(err error) error {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	switch {
	case apiErr.Code == "InvalidPaymentDate":
		return ErrInvalidPaymentDate
	case apiErr.Message != "":
		return apiErr
	case apiErr.Code != "":
		return fmt.Errorf("%s, SOLO.ro answered %s", invoiceChangeFailed, apiErr.Code)
	}
	return errors.New(invoiceChangeFailed)
}

var bucharest = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Bucharest")
	if err != nil {
		return nil
	}
	return loc
}()

// InvoiceAPIDate formats the civil date of day as Bucharest midnight with its
// offset, the format the invoice endpoints expect. A UTC timestamp could land
// on the previous day server side
func InvoiceAPIDate(day time.Time) (string, error) {
	if bucharest == nil {
		return "", errors.New("the Europe/Bucharest time zone is not available")
	}
	y, m, d := day.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, bucharest).Format("2006-01-02T15:04:05-07:00"), nil
}
