package client

import "fmt"

// EFactura represents an e-invoice from the national e-Factura system
type EFactura struct {
	SerialCode   string  `json:"SerialCode"`
	TotalAmount  float64 `json:"TotalAmount"`
	CurrencyCode string  `json:"CurrencyCode"`
	InvoiceDate  string  `json:"InvoiceDate"`
	PartyCode1   string  `json:"PartyCode1"`
	PartyName    string  `json:"PartyName"`
}

// EFacturaListResponse represents response from e-invoice list endpoint
type EFacturaListResponse struct {
	Items        []EFactura `json:"Items"`
	TotalResults *int       `json:"TotalResults"`
}

// ListEFactura fetches the list of e-invoices from the national system,
// optionally filtered by a server-side search query
func (c *Client) ListEFactura(startIndex, maxResults int, search string) (*EFacturaListResponse, error) {
	var result EFacturaListResponse
	if err := c.doJSON("POST", "/proxy/accounting/e-invoice/list-expenses", "/e-factura", newListRequest(startIndex, maxResults, search), &result); err != nil {
		return nil, fmt.Errorf("failed to list e-factura: %w", err)
	}
	return &result, nil
}
