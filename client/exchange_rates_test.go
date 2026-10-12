package client

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGetExchangeRates(t *testing.T) {
	var body string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/proxy/accounting/utility/exchange-rates" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body must be a bare JSON string: %v", err)
		}
		w.Write([]byte(`[
			{"FromCurrency":{"Id":2,"Code":"EUR","Name":"Euro","ShortName":"eur","IsDefault":false},"ToCurrency":{"Id":1,"Code":"RON","Name":"Lei","ShortName":"lei","IsDefault":true},"RateValue":5,"RateDate":"2026-07-14T00:00:00+03:00"},
			{"FromCurrency":"broken","RateValue":"x"},
			{"FromCurrency":{"Id":3,"Code":"","Name":"Dolar","ShortName":"usd","IsDefault":false},"ToCurrency":{"Id":1,"Code":"RON"},"RateValue":4.5,"RateDate":"2026-07-14T00:00:00+03:00"}
		]`))
	}))

	rates, err := c.GetExchangeRates(time.Date(2026, 7, 15, 23, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if body != "2026-07-15T00:00:00+03:00" {
		t.Errorf("summer body = %q", body)
	}
	if len(rates) != 2 || rates[0].CurrencyCode() != "EUR" || rates[0].RateValue != 5 || rates[1].CurrencyCode() != "USD" {
		t.Errorf("rates = %+v (malformed item must be skipped)", rates)
	}

	if _, err := c.GetExchangeRates(time.Date(2026, 1, 5, 0, 0, 0, 0, time.Local)); err != nil {
		t.Fatal(err)
	}
	if body != "2026-01-05T00:00:00+02:00" {
		t.Errorf("winter body = %q", body)
	}
}
