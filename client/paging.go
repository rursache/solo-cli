package client

import (
	"errors"
	"reflect"
)

// DefaultPageSize is the MaxResults used when the config has no page_size
const DefaultPageSize = 100

// maxListPages bounds fetchAll in case the server never ends a list
const maxListPages = 1000

// ErrListTruncated means paging stopped early, the items returned so far are
// still valid
var ErrListTruncated = errors.New("list truncated, SOLO.ro kept returning pages")

// fetchAll pages a list endpoint from StartIndex 0. It continues while
// TotalResults says more exist, or while pages come back full when
// TotalResults is null, and stops on an empty page
func fetchAll[T any](pageSize int, page func(start, max int) ([]T, *int, error)) ([]T, error) {
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	var all, prev []T
	for n := 0; n < maxListPages; n++ {
		items, total, err := page(len(all), pageSize)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return all, nil
		}
		// A server ignoring StartIndex answers the same page forever
		if n > 0 && reflect.DeepEqual(items, prev) {
			return all, ErrListTruncated
		}
		all = append(all, items...)
		prev = items
		if total != nil {
			if len(all) >= *total {
				return all, nil
			}
		} else if len(items) < pageSize {
			return all, nil
		}
	}
	return all, ErrListTruncated
}

// ListAllRevenues pages through the issued invoices
func (c *Client) ListAllRevenues(pageSize int, search string) ([]Revenue, error) {
	return fetchAll(pageSize, func(start, max int) ([]Revenue, *int, error) {
		resp, err := c.ListRevenues(start, max, search)
		if err != nil {
			return nil, nil, err
		}
		return resp.Items, resp.TotalResults, nil
	})
}

// ListAllExpenses pages through the registered expenses
func (c *Client) ListAllExpenses(pageSize int, search string) ([]Expense, error) {
	return fetchAll(pageSize, func(start, max int) ([]Expense, *int, error) {
		resp, err := c.ListExpenses(start, max, search)
		if err != nil {
			return nil, nil, err
		}
		return resp.Items, resp.TotalResults, nil
	})
}

// ListAllQueuedExpenses pages through the expense documents queue
func (c *Client) ListAllQueuedExpenses(pageSize int, search string) ([]QueuedExpense, error) {
	return fetchAll(pageSize, func(start, max int) ([]QueuedExpense, *int, error) {
		resp, err := c.ListQueuedExpenses(start, max, search)
		if err != nil {
			return nil, nil, err
		}
		return resp.Items, resp.TotalResults, nil
	})
}

// ListAllRejectedExpenses pages through the rejected expense documents
func (c *Client) ListAllRejectedExpenses(pageSize int) ([]RejectedExpense, error) {
	return fetchAll(pageSize, func(start, max int) ([]RejectedExpense, *int, error) {
		resp, err := c.ListRejectedExpenses(start, max)
		if err != nil {
			return nil, nil, err
		}
		return resp.Items, resp.TotalResults, nil
	})
}

// ListAllQueuedRevenues pages through the revenue documents queue
func (c *Client) ListAllQueuedRevenues(pageSize int, search string) ([]QueuedRevenue, error) {
	return fetchAll(pageSize, func(start, max int) ([]QueuedRevenue, *int, error) {
		resp, err := c.ListQueuedRevenues(start, max, search)
		if err != nil {
			return nil, nil, err
		}
		return resp.Items, resp.TotalResults, nil
	})
}

// ListAllRejectedRevenues pages through the rejected revenue documents
func (c *Client) ListAllRejectedRevenues(pageSize int) ([]RejectedRevenue, error) {
	return fetchAll(pageSize, func(start, max int) ([]RejectedRevenue, *int, error) {
		resp, err := c.ListRejectedRevenues(start, max)
		if err != nil {
			return nil, nil, err
		}
		return resp.Items, resp.TotalResults, nil
	})
}

// ListAllEFactura pages through the received e-Factura documents. Live
// TotalResults is the length of the returned page, so only full pages count
func (c *Client) ListAllEFactura(pageSize int, search string) ([]EFactura, error) {
	return fetchAll(pageSize, func(start, max int) ([]EFactura, *int, error) {
		resp, err := c.ListEFactura(start, max, search)
		if err != nil {
			return nil, nil, err
		}
		return resp.Items, nil, nil
	})
}

// ListAllCustomers pages through the clients directory
func (c *Client) ListAllCustomers(pageSize int, search string) ([]Customer, error) {
	return fetchAll(pageSize, func(start, max int) ([]Customer, *int, error) {
		resp, err := c.ListCustomers(start, max, search)
		if err != nil {
			return nil, nil, err
		}
		return resp.Items, resp.TotalResults, nil
	})
}
