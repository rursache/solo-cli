package client

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ExchangeRate is a BNR rate from FromCurrency to ToCurrency (RON)
type ExchangeRate struct {
	FromCurrency Currency `json:"FromCurrency"`
	ToCurrency   Currency `json:"ToCurrency"`
	RateValue    float64  `json:"RateValue"`
	RateDate     string   `json:"RateDate"`
}

// CurrencyCode is the uppercase source currency code
func (r ExchangeRate) CurrencyCode() string {
	if r.FromCurrency.Code != "" {
		return strings.ToUpper(r.FromCurrency.Code)
	}
	return strings.ToUpper(r.FromCurrency.ShortName)
}

// GetExchangeRates fetches the BNR rates for a day. It is a read-only POST
// whose body is a bare JSON date string. Malformed items are skipped
func (c *Client) GetExchangeRates(day time.Time) ([]ExchangeRate, error) {
	date, err := InvoiceAPIDate(day)
	if err != nil {
		return nil, err
	}
	var raw []json.RawMessage
	if err := c.doJSON("POST", "/proxy/accounting/utility/exchange-rates", "/draft-invoices", date, &raw); err != nil {
		return nil, fmt.Errorf("failed to get exchange rates: %w", err)
	}
	rates := make([]ExchangeRate, 0, len(raw))
	for _, item := range raw {
		var r ExchangeRate
		if json.Unmarshal(item, &r) != nil {
			continue
		}
		scrubValue(reflect.ValueOf(&r))
		rates = append(rates, r)
	}
	return rates, nil
}
