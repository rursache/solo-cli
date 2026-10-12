package client

import "fmt"

// QueuedRevenue is an uploaded revenue document the accountant has not
// processed yet. Shaped like QueuedExpense, only Id and DocumentCode are
// confirmed by the web app
type QueuedRevenue QueuedExpense

// QueuedRevenueResponse represents response from queued revenues endpoint
type QueuedRevenueResponse struct {
	Items        []QueuedRevenue `json:"Items"`
	TotalResults *int            `json:"TotalResults"`
}

// RejectedRevenue is an uploaded revenue document the accountant rejected
type RejectedRevenue RejectedExpense

// RejectedRevenueResponse represents response from rejected revenues endpoint
type RejectedRevenueResponse struct {
	Items        []RejectedRevenue `json:"Items"`
	TotalResults *int              `json:"TotalResults"`
}

// ListQueuedRevenues fetches uploaded revenue documents pending processing,
// optionally filtered by a server-side search query
func (c *Client) ListQueuedRevenues(startIndex, maxResults int, search string) (*QueuedRevenueResponse, error) {
	var result QueuedRevenueResponse
	if err := c.doJSON("POST", "/proxy/accounting/revenues/queued", "/revenues", newListRequest(startIndex, maxResults, search), &result); err != nil {
		return nil, fmt.Errorf("failed to list queued revenue documents: %w", err)
	}
	return &result, nil
}

// ListRejectedRevenues fetches uploaded revenue documents that were rejected
func (c *Client) ListRejectedRevenues(startIndex, maxResults int) (*RejectedRevenueResponse, error) {
	var result RejectedRevenueResponse
	if err := c.doJSON("POST", "/proxy/accounting/revenues/rejected", "/revenues", newListRequest(startIndex, maxResults, ""), &result); err != nil {
		return nil, fmt.Errorf("failed to list rejected revenue documents: %w", err)
	}
	return &result, nil
}

// DeleteRevenueDocument deletes a queued or rejected revenue document by its
// Id. Issued invoices are deleted with CancelInvoice instead
func (c *Client) DeleteRevenueDocument(id int) error {
	path := fmt.Sprintf("/proxy/accounting/revenues/%d", id)
	if err := c.doJSON("DELETE", path, "/revenues", nil, nil); err != nil {
		return fmt.Errorf("failed to delete revenue document: %w", err)
	}
	return nil
}
