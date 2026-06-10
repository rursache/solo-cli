package client

import "fmt"

// Revenue represents a single revenue/invoice item
type Revenue struct {
	UniqueCode         string          `json:"UniqueCode"`
	SerialCode         string          `json:"SerialCode"`
	ClientName         string          `json:"ClientName"`
	IssueDate          string          `json:"IssueDate"`
	PaymentDate        string          `json:"PaymentDate"`
	IsPaid             bool            `json:"IsPaid"`
	Total              float64         `json:"Total"`
	Currency           Currency        `json:"Currency"`
	InvoiceLocalAmount *LocalAmount    `json:"InvoiceLocalAmount"`
	IsExternalDocument bool            `json:"IsExternalDocument"`
	Status             *InvoiceStatus  `json:"Status"`
	EInvoiceStatus     *EInvoiceStatus `json:"EInvoiceStatus"`
}

// Currency represents a currency type
type Currency struct {
	Id        int    `json:"Id"`
	Code      string `json:"Code"`
	Name      string `json:"Name"`
	ShortName string `json:"ShortName"`
	IsDefault bool   `json:"IsDefault"`
}

// LocalAmount represents amount in local currency
type LocalAmount struct {
	Amount       float64  `json:"Amount"`
	VAT          float64  `json:"VAT"`
	Total        float64  `json:"Total"`
	Currency     Currency `json:"Currency"`
	ExchangeRate *float64 `json:"ExchangeRate"`
}

// InvoiceStatus represents invoice status
type InvoiceStatus struct {
	Code        string `json:"Code"`
	Name        string `json:"Name"`
	IsCancelled bool   `json:"IsCancelled"`
}

// EInvoiceStatus represents e-invoice status
type EInvoiceStatus struct {
	Code                 string `json:"Code"`
	Name                 string `json:"Name"`
	IsEligible           bool   `json:"IsEligible"`
	EnableCloudFormation bool   `json:"EnableCloudFormation"`
}

// RevenueListResponse represents response from revenue list endpoint
type RevenueListResponse struct {
	Items        []Revenue `json:"Items"`
	TotalResults *int      `json:"TotalResults"`
}

// RevenueCounts represents the revenues summary response, which holds
// document counts, not monetary totals
type RevenueCounts struct {
	RegisteredRevenues int `json:"RegisteredRevenues"`
	QueuedRevenues     int `json:"QueuedRevenues"`
	RejectedRevenues   int `json:"RejectedRevenues"`
}

// ListRevenues fetches the list of revenues/invoices
func (c *Client) ListRevenues(startIndex, maxResults int) (*RevenueListResponse, error) {
	reqBody := struct {
		listRequest
		InvoiceStatus           int `json:"InvoiceStatus"`
		ElectronicInvoiceStatus int `json:"ElectronicInvoiceStatus"`
	}{
		listRequest:   newListRequest(startIndex, maxResults),
		InvoiceStatus: 1,
	}

	var result RevenueListResponse
	if err := c.doJSON("POST", "/proxy/accounting/revenues/list", "/revenues", reqBody, &result); err != nil {
		return nil, fmt.Errorf("failed to list revenues: %w", err)
	}
	return &result, nil
}

// GetRevenueCounts fetches revenue document counts for a given year
func (c *Client) GetRevenueCounts(year int) (*RevenueCounts, error) {
	path := "/proxy/accounting/revenues/summary"
	if year > 0 {
		path = fmt.Sprintf("%s?year=%d", path, year)
	}

	var result RevenueCounts
	if err := c.doJSON("GET", path, "/", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get revenue counts: %w", err)
	}
	return &result, nil
}
