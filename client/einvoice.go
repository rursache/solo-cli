package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// EInvoiceStatus.Code values, compared lowercased like the web
const (
	EInvoiceNotEligible = "not-eligible"
	EInvoiceQueued      = "queued"
	EInvoiceSending     = "sending"
	EInvoiceSent        = "sent"
	EInvoiceError       = "error"
	EInvoiceSkipped     = "skipped"
	EInvoiceUnknown     = "unknown"
)

// Kind is the lowercased status code, EInvoiceUnknown for codes the web does not know
func (s *EInvoiceStatus) Kind() string {
	if s == nil {
		return EInvoiceUnknown
	}
	switch k := strings.ToLower(strings.TrimSpace(s.Code)); k {
	case EInvoiceNotEligible, EInvoiceQueued, EInvoiceSending, EInvoiceSent, EInvoiceError, EInvoiceSkipped:
		return k
	}
	return EInvoiceUnknown
}

// Tracked follows the web rule for showing the e-Factura card: a known code
// with EnableCloudFormation
func (s *EInvoiceStatus) Tracked() bool {
	return s != nil && s.Kind() != EInvoiceUnknown && s.EnableCloudFormation
}

// TracksEInvoice reports whether SOLO.ro handles the e-Factura of this
// invoice, only for SOLO-issued invoices with a tracked status
func (r *Revenue) TracksEInvoice() bool {
	return !r.IsExternalDocument && r.EInvoiceStatus.Tracked()
}

// EInvoiceStatusMeaning explains an EInvoiceStatus kind
func EInvoiceStatusMeaning(kind string) string {
	switch kind {
	case EInvoiceNotEligible:
		return "This invoice does not need an e-Factura"
	case EInvoiceQueued:
		return "Waiting to be sent to ANAF"
	case EInvoiceSending:
		return "Being sent to ANAF"
	case EInvoiceSent:
		return "Sent to ANAF"
	case EInvoiceError:
		return "Sending to ANAF failed"
	case EInvoiceSkipped:
		return "Left for manual sending, it goes to ANAF only when sent manually"
	}
	return "Unknown e-Factura status"
}

// FindRevenueByCode looks an issued invoice up by UniqueCode or SerialCode,
// trying a server search first and then paging through the whole list
func (c *Client) FindRevenueByCode(code string) (*Revenue, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("missing invoice code")
	}
	if resp, err := c.ListRevenues(0, 50, code); err == nil {
		if r := matchRevenue(resp.Items, code); r != nil {
			return r, nil
		}
	}

	all, err := c.ListAllRevenues(200, "")
	if err != nil && !errors.Is(err, ErrListTruncated) {
		return nil, err
	}
	if r := matchRevenue(all, code); r != nil {
		return r, nil
	}
	return nil, fmt.Errorf("no issued invoice with code or serial %q", code)
}

func matchRevenue(items []Revenue, code string) *Revenue {
	for i := range items {
		if items[i].UniqueCode == code {
			return &items[i]
		}
	}
	for i := range items {
		if strings.EqualFold(items[i].SerialCode, code) {
			return &items[i]
		}
	}
	return nil
}

func einvoicePath(action, uniqueCode string) string {
	return "/proxy/accounting/e-invoice/" + action + "/" + url.PathEscape(uniqueCode)
}

// EInvoiceQueueStatus is the answer of queue-status for a queued e-Factura
type EInvoiceQueueStatus struct {
	Status                string `json:"Status"`
	SendOn                string `json:"SendOn"`
	AllowExpeditedSending bool   `json:"AllowExpeditedSending"`
}

// EInvoiceQueueStatus.Status values
const (
	QueueInformationMissing = "QueueInformationMissing"
	InvoiceSendingScheduled = "InvoiceSendingScheduled"
	InvoiceSendingStarted   = "InvoiceSendingStarted"
	NoSpvAccess             = "NoSpvAccess"
)

func (q *EInvoiceQueueStatus) UnmarshalJSON(b []byte) error {
	var raw struct {
		Status                string          `json:"Status"`
		SendOn                json.RawMessage `json:"SendOn"`
		AllowExpeditedSending bool            `json:"AllowExpeditedSending"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*q = EInvoiceQueueStatus{Status: raw.Status, SendOn: looseText(raw.SendOn), AllowExpeditedSending: raw.AllowExpeditedSending}
	return nil
}

// CanExpedite is the web rule for offering "send now"
func (q *EInvoiceQueueStatus) CanExpedite() bool {
	return q.Status == InvoiceSendingScheduled && q.AllowExpeditedSending
}

// Message explains the queue status
func (q *EInvoiceQueueStatus) Message() string {
	switch q.Status {
	case QueueInformationMissing:
		return "No queue information yet, try again later"
	case InvoiceSendingScheduled:
		if q.SendOn != "" {
			return "Scheduled to be sent to ANAF on " + q.SendOn
		}
		return "Scheduled to be sent to ANAF"
	case InvoiceSendingStarted:
		return "Already being sent, waiting for ANAF to receive it"
	case NoSpvAccess:
		return "The SOLO account is not connected to the ANAF SPV"
	}
	return "Try again later"
}

// GetEInvoiceQueueStatus tells when a queued e-Factura will be sent
func (c *Client) GetEInvoiceQueueStatus(uniqueCode string) (*EInvoiceQueueStatus, error) {
	var result EInvoiceQueueStatus
	if err := c.doJSON("GET", einvoicePath("queue-status", uniqueCode), "/revenues", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get e-Factura queue status: %w", err)
	}
	return &result, nil
}

// EInvoiceErrorReason says why sending an e-Factura failed
type EInvoiceErrorReason struct {
	Reason string `json:"Reason"`
}

// Message explains the error reason code
func (e *EInvoiceErrorReason) Message() string {
	switch e.Reason {
	case "CustomerLiquidated":
		return "The client is deregistered and ANAF does not accept e-Facturi for deregistered companies, nothing else to do"
	case "UnknownReason":
		return "ANAF gave no extra details about the rejection"
	case "InvoiceNotFound":
		return "The invoice does not exist (it was deleted)"
	}
	return "No information about this e-Factura yet, try again later"
}

// GetEInvoiceErrorReason fetches why an e-Factura in the error state failed
func (c *Client) GetEInvoiceErrorReason(uniqueCode string) (*EInvoiceErrorReason, error) {
	var result EInvoiceErrorReason
	if err := c.doJSON("GET", einvoicePath("error-reason", uniqueCode), "/revenues", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get e-Factura error reason: %w", err)
	}
	return &result, nil
}

// EInvoiceSpvAttributes holds the ANAF SPV indexes of a sent e-Factura
type EInvoiceSpvAttributes struct {
	IsUploaded    bool   `json:"IsUploaded"`
	UploadIndex   string `json:"UploadIndex"`
	DownloadIndex string `json:"DownloadIndex"`
}

func (s *EInvoiceSpvAttributes) UnmarshalJSON(b []byte) error {
	var raw struct {
		IsUploaded    bool            `json:"IsUploaded"`
		UploadIndex   json.RawMessage `json:"UploadIndex"`
		DownloadIndex json.RawMessage `json:"DownloadIndex"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*s = EInvoiceSpvAttributes{IsUploaded: raw.IsUploaded, UploadIndex: looseText(raw.UploadIndex), DownloadIndex: looseText(raw.DownloadIndex)}
	return nil
}

// DisplayIndex is the upload index to show, empty until ANAF confirms the upload
func (s *EInvoiceSpvAttributes) DisplayIndex() string {
	if !s.IsUploaded {
		return ""
	}
	return s.UploadIndex
}

// GetEInvoiceSpvAttributes fetches the ANAF SPV upload index of a sent e-Factura
func (c *Client) GetEInvoiceSpvAttributes(uniqueCode string) (*EInvoiceSpvAttributes, error) {
	var result EInvoiceSpvAttributes
	if err := c.doJSON("GET", einvoicePath("spv-attributes", uniqueCode), "/revenues", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get e-Factura SPV attributes: %w", err)
	}
	return &result, nil
}

// EInvoiceSendError is a manual send SOLO.ro answered with a Status other than Queued
type EInvoiceSendError struct {
	Status string
}

func (e *EInvoiceSendError) Error() string {
	return ManualSendMessage(e.Status)
}

// ManualSendMessage explains a manual-send Status value
func ManualSendMessage(status string) string {
	switch status {
	case "Queued":
		return "The e-Factura will be sent to ANAF shortly"
	case "InvoiceNotFound":
		return "the invoice does not exist (it was deleted)"
	case "InvalidStatus":
		return "the invoice cannot be sent manually right now, try again later"
	case "InvoiceLinesMissing":
		return "there is not enough invoice data to build the e-Factura"
	case "ObsoleteInvoice":
		return "invoices issued before 2024 do not go to e-Factura"
	// The typo is in the server value
	case "SpvAccessUnvailable", "SpvAccessUnavailable":
		return "the SOLO account is not connected to the ANAF SPV"
	case "":
		return "SOLO.ro could not send the e-Factura, try again later"
	}
	return "SOLO.ro could not send the e-Factura (" + status + ")"
}

// SendEInvoice submits an invoice left on manual sending (status skipped) to
// ANAF. A Status other than Queued comes back as *EInvoiceSendError
func (c *Client) SendEInvoice(uniqueCode string) error {
	var result StatusResponse
	if err := c.doJSON("POST", einvoicePath("manual-send", uniqueCode), "/revenues", struct{}{}, &result); err != nil {
		return fmt.Errorf("failed to send e-Factura: %w", err)
	}
	if result.Status != "Queued" {
		return &EInvoiceSendError{Status: result.Status}
	}
	return nil
}

// ExpediteEInvoice sends a scheduled e-Factura now instead of waiting for its send date
func (c *Client) ExpediteEInvoice(uniqueCode string) error {
	err := c.doOk("POST", einvoicePath("expedited-send", uniqueCode), "/revenues", struct{}{}, nil)
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Code == "" && apiErr.Message == "" {
		return errors.New("the e-Factura send date could not be changed, try again later")
	}
	if err != nil {
		return fmt.Errorf("failed to expedite e-Factura: %w", err)
	}
	return nil
}

// EInvoiceSettings is the ANAF SPV connection state and the sending
// preferences. SendingDelayDays and SendContactInfo are nil when missing
type EInvoiceSettings struct {
	IsConnected           bool   `json:"IsConnected"`
	HasPersonalAccess     bool   `json:"HasPersonalAccess"`
	HasDelegatedAccess    bool   `json:"HasDelegatedAccess"`
	DelegatedAccessStatus string `json:"DelegatedAccessStatus"`
	PersonalAccessStatus  string `json:"PersonalAccessStatus"`
	SendingDelayDays      *int   `json:"SendingDelayDays"`
	SendContactInfo       *bool  `json:"SendContactInfo"`
	IsTransitioning       bool   `json:"IsTransitioning"`
}

func (s *EInvoiceSettings) UnmarshalJSON(b []byte) error {
	var raw struct {
		IsConnected           bool            `json:"IsConnected"`
		HasPersonalAccess     bool            `json:"HasPersonalAccess"`
		HasDelegatedAccess    bool            `json:"HasDelegatedAccess"`
		DelegatedAccessStatus json.RawMessage `json:"DelegatedAccessStatus"`
		PersonalAccessStatus  json.RawMessage `json:"PersonalAccessStatus"`
		SendingDelayDays      json.RawMessage `json:"SendingDelayDays"`
		SendContactInfo       *bool           `json:"SendContactInfo"`
		IsTransitioning       bool            `json:"IsTransitioning"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*s = EInvoiceSettings{
		IsConnected:           raw.IsConnected,
		HasPersonalAccess:     raw.HasPersonalAccess,
		HasDelegatedAccess:    raw.HasDelegatedAccess,
		DelegatedAccessStatus: looseText(raw.DelegatedAccessStatus),
		PersonalAccessStatus:  looseText(raw.PersonalAccessStatus),
		SendContactInfo:       raw.SendContactInfo,
		IsTransitioning:       raw.IsTransitioning,
	}
	// Decoded leniently, a string or fractional number must not break the copy into the update
	if f, err := strconv.ParseFloat(looseText(raw.SendingDelayDays), 64); err == nil {
		days := int(f)
		s.SendingDelayDays = &days
	}
	return nil
}

// SendingDelayLabel is the web label for a SendingDelayDays value
func SendingDelayLabel(days int) string {
	switch days {
	case 0:
		return "as soon as possible after issuing"
	case 1:
		return "within 1 day of issuing"
	}
	return fmt.Sprintf("within %d days of issuing", days)
}

// ContactInfoLabel is the web label for a SendContactInfo value
func ContactInfoLabel(send bool) string {
	if send {
		return "e-mail and phone included in the e-Factura"
	}
	return "not included in the e-Factura"
}

// GetEInvoiceSettings fetches the SPV connection state and sending preferences
func (c *Client) GetEInvoiceSettings() (*EInvoiceSettings, error) {
	var result EInvoiceSettings
	if err := c.doJSON("GET", "/proxy/accounting/e-invoice/settings", "/settings", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get e-Factura settings: %w", err)
	}
	return &result, nil
}

// UpdateEInvoiceSettings saves the sending preferences, the web sends exactly these two keys
func (c *Client) UpdateEInvoiceSettings(sendingDelayDays int, sendContactInfo bool) error {
	body := struct {
		SendingDelayDays int  `json:"SendingDelayDays"`
		SendContactInfo  bool `json:"SendContactInfo"`
	}{sendingDelayDays, sendContactInfo}
	if err := c.doOk("POST", "/proxy/accounting/e-invoice/settings/update", "/settings", body, nil); err != nil {
		return fmt.Errorf("failed to update e-Factura settings: %w", err)
	}
	return nil
}

// looseText reads a JSON string, number or bool as text, null or missing as empty
func looseText(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	if raw[0] == '{' || raw[0] == '[' {
		return ""
	}
	return string(raw)
}
