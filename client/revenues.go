package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

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

// RevenueListRequest represents request body for listing revenues
type RevenueListRequest struct {
	SearchText              string `json:"SearchText"`
	StartIndex              int    `json:"StartIndex"`
	MaxResults              int    `json:"MaxResults"`
	SortBy                  string `json:"SortBy"`
	SortAsc                 bool   `json:"SortAsc"`
	InvoiceStatus           int    `json:"InvoiceStatus"`
	ElectronicInvoiceStatus int    `json:"ElectronicInvoiceStatus"`
}

// RevenueListResponse represents response from revenue list endpoint
type RevenueListResponse struct {
	Items        []Revenue `json:"Items"`
	TotalResults *int      `json:"TotalResults"`
}

// RevenueSummary represents revenue summary response
type RevenueSummary struct {
	TotalAmount float64 `json:"TotalAmount"`
}

// ListRevenues fetches the list of revenues/invoices
func (c *Client) ListRevenues(startIndex, maxResults int) (*RevenueListResponse, error) {
	reqBody := RevenueListRequest{
		SearchText:              "",
		StartIndex:              startIndex,
		MaxResults:              maxResults,
		SortBy:                  "",
		SortAsc:                 true,
		InvoiceStatus:           1,
		ElectronicInvoiceStatus: 0,
	}

	bodyData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL+"/proxy/accounting/revenues/list", bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/revenues")

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
		return nil, fmt.Errorf("failed to list revenues: status %d", resp.StatusCode)
	}

	var result RevenueListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetRevenueSummary fetches revenue summary for a given year
func (c *Client) GetRevenueSummary(year int) (*RevenueSummary, error) {
	url := fmt.Sprintf("%s/proxy/accounting/revenues/summary", baseURL)
	if year > 0 {
		url = fmt.Sprintf("%s?year=%d", url, year)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", baseURL+"/")

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
		return nil, fmt.Errorf("failed to get revenue summary: status %d", resp.StatusCode)
	}

	var result RevenueSummary
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
