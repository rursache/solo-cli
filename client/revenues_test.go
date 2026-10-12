package client

import (
	"net/http"
	"testing"
)

// Foreign currency invoices carry the BNR rate as an object, a bare number
// type made the whole list fail to decode
func TestListRevenuesExchangeRateObject(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Items":[
			{"UniqueCode":"aaa111","SerialCode":"INV-1","Total":100,"Currency":{"ShortName":"EUR"},
			 "InvoiceLocalAmount":{"Amount":500,"VAT":0,"Total":500,"Currency":{"ShortName":"RON"},
			  "ExchangeRate":{"FromCurrency":{"Code":"EUR"},"ToCurrency":{"Code":"RON"},"RateValue":5,"RateDate":"2026-01-02T00:00:00+02:00"}}},
			{"UniqueCode":"bbb222","SerialCode":"INV-2","Total":200,"Currency":{"ShortName":"RON"},
			 "InvoiceLocalAmount":{"Amount":200,"VAT":0,"Total":200,"Currency":{"ShortName":"RON"},"ExchangeRate":null}}
		],"TotalResults":null}`))
	}))

	resp, err := c.ListRevenues(0, 100, "")
	if err != nil {
		t.Fatalf("ListRevenues: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("items = %d", len(resp.Items))
	}
	rate := resp.Items[0].InvoiceLocalAmount.ExchangeRate
	if rate == nil || rate.RateValue != 5 || rate.FromCurrency.Code != "EUR" || rate.ToCurrency.Code != "RON" {
		t.Errorf("exchange rate = %+v", rate)
	}
	if resp.Items[1].InvoiceLocalAmount.ExchangeRate != nil {
		t.Error("null exchange rate must stay nil")
	}
}
