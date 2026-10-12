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

// maxCustomerPages bounds ListAllCustomers in case the server ignores paging
const maxCustomerPages = 100

// ListAllCustomers pages through the clients directory
func (c *Client) ListAllCustomers(pageSize int, search string) ([]Customer, error) {
	if pageSize <= 0 {
		pageSize = 100
	}
	var all []Customer
	for page := 0; page < maxCustomerPages; page++ {
		resp, err := c.ListCustomers(len(all), pageSize, search)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Items...)
		if len(resp.Items) < pageSize || (resp.TotalResults != nil && len(all) >= *resp.TotalResults) {
			break
		}
	}
	return all, nil
}
