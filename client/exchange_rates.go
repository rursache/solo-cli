package client

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
	_ "time/tzdata" // Europe/Bucharest must resolve on systems without zoneinfo
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

// exchangeRateDate is the civil day at Bucharest midnight, the format the
// invoice endpoints take. A UTC timestamp could land on the previous day
func exchangeRateDate(day time.Time) string {
	loc, err := time.LoadLocation("Europe/Bucharest")
	if err != nil {
		loc = time.FixedZone("EET", 2*3600)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc).Format("2006-01-02T15:04:05-07:00")
}

// GetExchangeRates fetches the BNR rates for a day. It is a read-only POST
// whose body is a bare JSON date string. Malformed items are skipped
func (c *Client) GetExchangeRates(day time.Time) ([]ExchangeRate, error) {
	var raw []json.RawMessage
	if err := c.doJSON("POST", "/proxy/accounting/utility/exchange-rates", "/draft-invoices", exchangeRateDate(day), &raw); err != nil {
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
