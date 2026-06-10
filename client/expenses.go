package client

import "fmt"

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

// ExpenseListResponse represents response from expense list endpoint
type ExpenseListResponse struct {
	Items        []Expense `json:"Items"`
	TotalResults *int      `json:"TotalResults"`
}

// ExpenseCounts represents the expenses summary response, which holds
// document counts, not monetary totals
type ExpenseCounts struct {
	RegisteredExpenses int `json:"RegisteredExpenses"`
	QueuedExpenses     int `json:"QueuedExpenses"`
	RejectedExpenses   int `json:"RejectedExpenses"`
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

// ListExpenses fetches the list of expenses, optionally filtered by a
// server-side search query
func (c *Client) ListExpenses(startIndex, maxResults int, search string) (*ExpenseListResponse, error) {
	var result ExpenseListResponse
	if err := c.doJSON("POST", "/proxy/accounting/expenses/list", "/expenses", newListRequest(startIndex, maxResults, search), &result); err != nil {
		return nil, fmt.Errorf("failed to list expenses: %w", err)
	}
	return &result, nil
}

// GetExpenseCounts fetches expense document counts for a given year
func (c *Client) GetExpenseCounts(year int) (*ExpenseCounts, error) {
	path := "/proxy/accounting/expenses/summary"
	if year > 0 {
		path = fmt.Sprintf("%s?year=%d", path, year)
	}

	var result ExpenseCounts
	if err := c.doJSON("GET", path, "/", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get expense counts: %w", err)
	}
	return &result, nil
}

// ListQueuedExpenses fetches documents pending processing, optionally
// filtered by a server-side search query
func (c *Client) ListQueuedExpenses(startIndex, maxResults int, search string) (*QueuedExpenseResponse, error) {
	var result QueuedExpenseResponse
	if err := c.doJSON("POST", "/proxy/accounting/expenses/queued", "/expenses", newListRequest(startIndex, maxResults, search), &result); err != nil {
		return nil, fmt.Errorf("failed to list queued expenses: %w", err)
	}
	return &result, nil
}

// ListRejectedExpenses fetches expenses that were rejected
func (c *Client) ListRejectedExpenses(startIndex, maxResults int) (*RejectedExpenseResponse, error) {
	var result RejectedExpenseResponse
	if err := c.doJSON("POST", "/proxy/accounting/expenses/rejected", "/expenses", newListRequest(startIndex, maxResults, ""), &result); err != nil {
		return nil, fmt.Errorf("failed to list rejected expenses: %w", err)
	}
	return &result, nil
}

// DeleteExpense deletes an expense by ID
func (c *Client) DeleteExpense(id int) error {
	path := fmt.Sprintf("/proxy/accounting/expenses/%d", id)
	if err := c.doJSON("DELETE", path, "/expenses", nil, nil); err != nil {
		return fmt.Errorf("failed to delete expense: %w", err)
	}
	return nil
}
