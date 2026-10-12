package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestListAllCustomersPages(t *testing.T) {
	const total = 5
	var starts []int
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/proxy/accounting/customers/list" || !strings.HasSuffix(r.Header.Get("Referer"), "/clients") {
			t.Errorf("unexpected request %s %s referer %s", r.Method, r.URL.Path, r.Header.Get("Referer"))
		}
		var req listRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.SearchText != "acme" || req.MaxResults != 2 {
			t.Errorf("request = %+v", req)
		}
		starts = append(starts, req.StartIndex)
		var items []string
		for i := req.StartIndex; i < total && i < req.StartIndex+req.MaxResults; i++ {
			items = append(items, fmt.Sprintf(`{"CustomerId":%d,"TypeId":1,"TypeLabel":"Persoana juridica","Country":{"Id":1,"Code":"RO","Name":"Romania","IsDefault":true,"IsEuCountry":true},"Name":"Client %d","Code1":"RO100","Code2":"J40/1/2020","Email":"client@example.com","Phone":null}`, i+1, i+1))
		}
		// TotalResults is null live, paging must stop on the short page
		fmt.Fprintf(w, `{"Items":[%s],"TotalResults":null}`, strings.Join(items, ","))
	}))

	all, err := c.ListAllCustomers(2, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != total || all[4].CustomerId != 5 || all[0].Country.Code != "RO" || all[0].Phone != "" {
		t.Errorf("customers = %+v", all)
	}
	if fmt.Sprint(starts) != "[0 2 4]" {
		t.Errorf("start indexes = %v", starts)
	}
}

func TestListAllCustomersTotalResults(t *testing.T) {
	hits := 0
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write([]byte(`{"Items":[{"CustomerId":1,"Name":"A"},{"CustomerId":2,"Name":"B"}],"TotalResults":2}`))
	}))
	all, err := c.ListAllCustomers(2, "")
	if err != nil || len(all) != 2 || hits != 1 {
		t.Errorf("full page with TotalResults reached must stop: %d items, %d hits, %v", len(all), hits, err)
	}
}
