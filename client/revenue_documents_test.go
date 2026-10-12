package client

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRevenueDocuments(t *testing.T) {
	var deleted []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.Header.Get("Referer"), "/revenues") {
			t.Errorf("%s referer = %q", r.URL.Path, r.Header.Get("Referer"))
		}
		switch {
		case r.URL.Path == "/proxy/accounting/revenues/queued" && r.Method == "POST":
			var body listRequest
			json.NewDecoder(r.Body).Decode(&body)
			if body.SearchText != "factura" || body.MaxResults != 50 {
				t.Errorf("queued body = %+v", body)
			}
			w.Write([]byte(`{"Items":[{"Id":11,"DocumentCode":"doc_a","DocumentName":"factura\u001b[1m.pdf","DaysPassed":2,"IsOverdue":false}],"TotalResults":null}`))
		case r.URL.Path == "/proxy/accounting/revenues/rejected" && r.Method == "POST":
			w.Write([]byte(`{"Items":[{"Id":12,"DocumentCode":"doc_b","DocumentName":"scan.jpg","Reason":"Ilizibil","AllowResubmit":true}],"TotalResults":null}`))
		case r.URL.Path == "/proxy/accounting/revenues/11" && r.Method == "DELETE":
			deleted = append(deleted, "11")
		case r.URL.Path == "/proxy/accounting/revenues/13" && r.Method == "DELETE":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	queued, err := c.ListQueuedRevenues(0, 50, "factura")
	if err != nil || len(queued.Items) != 1 {
		t.Fatalf("ListQueuedRevenues: %v, %+v", err, queued)
	}
	if q := queued.Items[0]; q.Id != 11 || q.DocumentCode != "doc_a" || q.DocumentName != "factura[1m.pdf" || q.DaysPassed != 2 {
		t.Errorf("queued item = %+v", q)
	}

	rejected, err := c.ListRejectedRevenues(0, 50)
	if err != nil || len(rejected.Items) != 1 {
		t.Fatalf("ListRejectedRevenues: %v, %+v", err, rejected)
	}
	if r := rejected.Items[0]; r.Id != 12 || r.Reason != "Ilizibil" || !r.AllowResubmit {
		t.Errorf("rejected item = %+v", r)
	}

	if err := c.DeleteRevenueDocument(11); err != nil || len(deleted) != 1 {
		t.Errorf("DeleteRevenueDocument: %v, %v", err, deleted)
	}
	if err := c.DeleteRevenueDocument(13); err == nil {
		t.Error("delete with a 404 must fail")
	}
}
