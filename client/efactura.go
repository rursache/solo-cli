package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// EFactura represents an e-invoice from the national e-Factura system
type EFactura struct {
	SerialCode   string  `json:"SerialCode"`
	TotalAmount  float64 `json:"TotalAmount"`
	CurrencyCode string  `json:"CurrencyCode"`
	InvoiceDate  string  `json:"InvoiceDate"`
	PartyCode1   string  `json:"PartyCode1"`
	PartyName    string  `json:"PartyName"`
}

// EFacturaListRequest represents request body for listing e-invoices
type EFacturaListRequest struct {
	SearchText string `json:"SearchText"`
	StartIndex int    `json:"StartIndex"`
	MaxResults int    `json:"MaxResults"`
	SortBy     string `json:"SortBy"`
	SortAsc    bool   `json:"SortAsc"`
}

// EFacturaListResponse represents response from e-invoice list endpoint
type EFacturaListResponse struct {
	Items        []EFactura `json:"Items"`
	TotalResults *int       `json:"TotalResults"`
}

// ListEFactura fetches the list of e-invoices from the national system
func (c *Client) ListEFactura(startIndex, maxResults int) (*EFacturaListResponse, error) {
	reqBody := EFacturaListRequest{
		SearchText: "",
		StartIndex: startIndex,
		MaxResults: maxResults,
		SortBy:     "",
		SortAsc:    true,
	}

	bodyData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL+"/proxy/accounting/e-invoice/list-expenses", bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/e-factura")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list e-factura: status %d", resp.StatusCode)
	}

	var result EFacturaListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
