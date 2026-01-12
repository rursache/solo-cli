package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Expense represents a single expense item
type Expense struct {
	UniqueCode         string              `json:"UniqueCode"`
	DocumentCode       *string             `json:"DocumentCode"`
	DocumentMimeType   *string             `json:"DocumentMimeType"`
	SupplierName       string              `json:"SupplierName"`
	PurchaseDate       string              `json:"PurchaseDate"`
	Category           string              `json:"Category"`
	PrimaryCategory    string              `json:"PrimaryCategory"`
	CategoryCount      int                 `json:"CategoryCount"`
	Total              float64             `json:"Total"`
	Deductibility      string              `json:"Deductibility"`
	Currency           Currency            `json:"Currency"`
	ExpenseLocalAmount *ExpenseLocalAmount `json:"ExpenseLocalAmount"`
}

// ExpenseLocalAmount represents expense amount in local currency
type ExpenseLocalAmount struct {
	Total    float64  `json:"Total"`
	Currency Currency `json:"Currency"`
}

// ExpenseListRequest represents request body for listing expenses
type ExpenseListRequest struct {
	SearchText string `json:"SearchText"`
	StartIndex int    `json:"StartIndex"`
	MaxResults int    `json:"MaxResults"`
	SortBy     string `json:"SortBy"`
	SortAsc    bool   `json:"SortAsc"`
}

// ExpenseListResponse represents response from expense list endpoint
type ExpenseListResponse struct {
	Items        []Expense `json:"Items"`
	TotalResults *int      `json:"TotalResults"`
}

// ExpenseSummary represents expense summary response
type ExpenseSummary struct {
	TotalAmount float64 `json:"TotalAmount"`
}

// QueuedExpense represents a document in the expense queue
type QueuedExpense struct {
	Id                 int    `json:"Id"`
	DocumentCode       string `json:"DocumentCode"`
	DocumentName       string `json:"DocumentName"`
	DocumentMimeType   string `json:"DocumentMimeType"`
	CreatedOn          string `json:"CreatedOn"`
	DaysPassed         int    `json:"DaysPassed"`
	ProcessingDeadline string `json:"ProcessingDeadline"`
	IsOverdue          bool   `json:"IsOverdue"`
}

// QueuedExpenseResponse represents response from queued expenses endpoint
type QueuedExpenseResponse struct {
	Items        []QueuedExpense `json:"Items"`
	TotalResults *int            `json:"TotalResults"`
}

// RejectedExpense represents an expense that was rejected
type RejectedExpense struct {
	Id               int    `json:"Id"`
	DocumentCode     string `json:"DocumentCode"`
	DocumentName     string `json:"DocumentName"`
	DocumentMimeType string `json:"DocumentMimeType"`
	Reason           string `json:"Reason"`
	AllowResubmit    bool   `json:"AllowResubmit"`
	CreatedOn        string `json:"CreatedOn"`
	RejectedOn       string `json:"RejectedOn"`
	DaysPassed       int    `json:"DaysPassed"`
}

// RejectedExpenseResponse represents response from rejected expenses endpoint
type RejectedExpenseResponse struct {
	Items        []RejectedExpense `json:"Items"`
	TotalResults *int              `json:"TotalResults"`
}

// ListExpenses fetches the list of expenses
func (c *Client) ListExpenses(startIndex, maxResults int) (*ExpenseListResponse, error) {
	reqBody := ExpenseListRequest{
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

	req, err := http.NewRequest("POST", baseURL+"/proxy/accounting/expenses/list", bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/expenses")

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
		return nil, fmt.Errorf("failed to list expenses: status %d", resp.StatusCode)
	}

	var result ExpenseListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetExpenseSummary fetches expense summary for a given year
func (c *Client) GetExpenseSummary(year int) (*ExpenseSummary, error) {
	url := fmt.Sprintf("%s/proxy/accounting/expenses/summary", baseURL)
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
		return nil, fmt.Errorf("failed to get expense summary: status %d", resp.StatusCode)
	}

	var result ExpenseSummary
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListQueuedExpenses fetches documents pending processing
func (c *Client) ListQueuedExpenses(startIndex, maxResults int) (*QueuedExpenseResponse, error) {
	reqBody := ExpenseListRequest{
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

	req, err := http.NewRequest("POST", baseURL+"/proxy/accounting/expenses/queued", bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/expenses")

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
		return nil, fmt.Errorf("failed to list queued expenses: status %d", resp.StatusCode)
	}

	var result QueuedExpenseResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListRejectedExpenses fetches expenses that were rejected
func (c *Client) ListRejectedExpenses(startIndex, maxResults int) (*RejectedExpenseResponse, error) {
	reqBody := ExpenseListRequest{
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

	req, err := http.NewRequest("POST", baseURL+"/proxy/accounting/expenses/rejected", bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/expenses")

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
		return nil, fmt.Errorf("failed to list rejected expenses: status %d", resp.StatusCode)
	}

	var result RejectedExpenseResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteExpense deletes an expense by ID
func (c *Client) DeleteExpense(id int) error {
	url := fmt.Sprintf("%s/proxy/accounting/expenses/%d", baseURL, id)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/expenses")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete expense: status %d", resp.StatusCode)
	}

	return nil
}
