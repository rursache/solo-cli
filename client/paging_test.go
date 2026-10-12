package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// pager serves ids 0..n-1, capping pages at limit when limit > 0
func pager(n, limit int, withTotal bool, starts *[]int) func(start, max int) ([]int, *int, error) {
	return func(start, max int) ([]int, *int, error) {
		*starts = append(*starts, start)
		if limit > 0 && max > limit {
			max = limit
		}
		var items []int
		for i := start; i < n && i < start+max; i++ {
			items = append(items, i)
		}
		if !withTotal {
			return items, nil, nil
		}
		total := n
		return items, &total, nil
	}
}

func TestFetchAll(t *testing.T) {
	tests := []struct {
		name      string
		n, limit  int
		withTotal bool
		wantStart string
	}{
		{"total over pages", 5, 0, true, "[0 2 4]"},
		{"total on exact page", 4, 0, true, "[0 2]"},
		{"total continues past a capped page", 5, 1, true, "[0 1 2 3 4]"},
		{"null total stops on short page", 5, 0, false, "[0 2 4]"},
		{"null total stops on empty page", 4, 0, false, "[0 2 4]"},
		{"empty list", 0, 0, true, "[0]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var starts []int
			all, err := fetchAll(2, pager(tt.n, tt.limit, tt.withTotal, &starts))
			if err != nil || len(all) != tt.n {
				t.Fatalf("got %v, %v, want %d items", all, err, tt.n)
			}
			for i, v := range all {
				if v != i {
					t.Fatalf("items out of order: %v", all)
				}
			}
			if fmt.Sprint(starts) != tt.wantStart {
				t.Errorf("start indexes = %v, want %s", starts, tt.wantStart)
			}
		})
	}
}

func TestFetchAllStopsOnLyingTotal(t *testing.T) {
	var starts []int
	page := pager(3, 0, false, &starts)
	all, err := fetchAll(2, func(start, max int) ([]int, *int, error) {
		items, _, err := page(start, max)
		total := 10
		return items, &total, err
	})
	if err != nil || len(all) != 3 || fmt.Sprint(starts) != "[0 2 3]" {
		t.Errorf("got %v, %v, starts %v", all, err, starts)
	}
}

func TestFetchAllRepeatedPage(t *testing.T) {
	calls := 0
	all, err := fetchAll(2, func(start, max int) ([]int, *int, error) {
		calls++
		return []int{1, 2}, nil, nil
	})
	if !errors.Is(err, ErrListTruncated) || len(all) != 2 || calls != 2 {
		t.Errorf("server ignoring StartIndex: %v, %v, %d calls", all, err, calls)
	}
}

func TestFetchAllPageBound(t *testing.T) {
	calls := 0
	all, err := fetchAll(1, func(start, max int) ([]int, *int, error) {
		calls++
		return []int{start}, nil, nil
	})
	if !errors.Is(err, ErrListTruncated) || calls != maxListPages || len(all) != maxListPages {
		t.Errorf("endless list: %v, %d calls, %d items", err, calls, len(all))
	}
}

func TestFetchAllError(t *testing.T) {
	boom := errors.New("boom")
	_, err := fetchAll(2, func(start, max int) ([]int, *int, error) {
		if start > 0 {
			return nil, nil, boom
		}
		return []int{1, 2}, nil, nil
	})
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

func TestListAllExpensesPagesWithPageSize(t *testing.T) {
	var reqs []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req listRequest
		json.NewDecoder(r.Body).Decode(&req)
		reqs = append(reqs, fmt.Sprintf("%s@%d/%d", r.URL.Path, req.StartIndex, req.MaxResults))
		var items []string
		for i := req.StartIndex; i < 3 && i < req.StartIndex+req.MaxResults; i++ {
			items = append(items, fmt.Sprintf(`{"SupplierName":"Supplier %d","Total":10}`, i))
		}
		fmt.Fprintf(w, `{"Items":[%s],"TotalResults":3}`, strings.Join(items, ","))
	}))
	all, err := c.ListAllExpenses(2, "")
	if err != nil || len(all) != 3 || all[2].SupplierName != "Supplier 2" {
		t.Fatalf("expenses = %+v, %v", all, err)
	}
	want := "[/proxy/accounting/expenses/list@0/2 /proxy/accounting/expenses/list@2/2]"
	if fmt.Sprint(reqs) != want {
		t.Errorf("requests = %v, want %s", reqs, want)
	}
}

func TestListAllDefaultPageSize(t *testing.T) {
	var max int
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req listRequest
		json.NewDecoder(r.Body).Decode(&req)
		max = req.MaxResults
		w.Write([]byte(`{"Items":[],"TotalResults":null}`))
	}))
	if _, err := c.ListAllQueuedExpenses(0, ""); err != nil || max != DefaultPageSize {
		t.Errorf("MaxResults = %d, %v", max, err)
	}
}

// Live e-Factura answers TotalResults with the length of the page it sent
func TestListAllEFacturaIgnoresPageTotal(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req listRequest
		json.NewDecoder(r.Body).Decode(&req)
		var items []string
		for i := req.StartIndex; i < 5 && i < req.StartIndex+req.MaxResults; i++ {
			items = append(items, fmt.Sprintf(`{"SerialCode":"EF-%d"}`, i))
		}
		fmt.Fprintf(w, `{"Items":[%s],"TotalResults":%d}`, strings.Join(items, ","), len(items))
	}))
	all, err := c.ListAllEFactura(2, "")
	if err != nil || len(all) != 5 || all[4].SerialCode != "EF-4" {
		t.Errorf("efactura = %+v, %v", all, err)
	}
}
