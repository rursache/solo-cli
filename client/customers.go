package client

import "fmt"

// Customer is a row of the clients directory
type Customer struct {
	CustomerId int              `json:"CustomerId"`
	TypeId     int              `json:"TypeId"`
	TypeLabel  string           `json:"TypeLabel"`
	Country    *CustomerCountry `json:"Country"`
	Name       string           `json:"Name"`
	Code1      string           `json:"Code1"`
	Code2      string           `json:"Code2"`
	Email      string           `json:"Email"`
	Phone      string           `json:"Phone"`
}

// CustomerCountry is the customer's country
type CustomerCountry struct {
	Id          int    `json:"Id"`
	Code        string `json:"Code"`
	Name        string `json:"Name"`
	IsDefault   bool   `json:"IsDefault"`
	IsEuCountry bool   `json:"IsEuCountry"`
}

// CustomerListResponse is one page of clients. TotalResults is null live, so
// paging stops on a short page
type CustomerListResponse struct {
	Items        []Customer `json:"Items"`
	TotalResults *int       `json:"TotalResults"`
}

// ListCustomers fetches a page of the clients directory, optionally filtered
// by a server-side search query
func (c *Client) ListCustomers(startIndex, maxResults int, search string) (*CustomerListResponse, error) {
	var result CustomerListResponse
	if err := c.doJSON("POST", "/proxy/accounting/customers/list", "/clients", newListRequest(startIndex, maxResults, search), &result); err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}
	return &result, nil
}
