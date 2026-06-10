package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestClient spins up a mock SOLO.ro server and points the client at it
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	orig := baseURL
	baseURL = server.URL
	t.Cleanup(func() { baseURL = orig })

	c, err := New("test-agent")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLoginSuccess(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/security/login" || r.Method != "POST" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["UserName"] != "user" || body["Password"] != "pass" {
			t.Errorf("credentials not sent: %v", body)
		}
		json.NewEncoder(w).Encode(map[string]string{"AuthenticationStatus": "OK"})
	}))

	if err := c.Login("user", "pass"); err != nil {
		t.Errorf("Login: %v", err)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"AuthenticationStatus": "FAILED"})
	}))

	if err := c.Login("user", "wrong"); err != ErrAuthenticationFailed {
		t.Errorf("Login = %v, want ErrAuthenticationFailed", err)
	}
}

func TestGetSummary(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/accounting/dashboard/summary" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Summary{
			Year: 2026, DisplayCurrency: "RON",
			TotalRevenues: 50000.50, TotalDeductibleExpenses: 20000.25,
			HasTaxes: true, Taxes: 6000.75,
		})
	}))

	s, err := c.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if s.Year != 2026 || s.TotalRevenues != 50000.50 {
		t.Errorf("summary not parsed: %+v", s)
	}
}

func TestGetSummaryForYearPassesYearParam(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("year"); got != "2025" {
			t.Errorf("year param = %q, want 2025", got)
		}
		json.NewEncoder(w).Encode(Summary{Year: 2025})
	}))

	if _, err := c.GetSummaryForYear(2025); err != nil {
		t.Fatalf("GetSummaryForYear: %v", err)
	}
}

func TestListRevenues(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/accounting/revenues/list" || r.Method != "POST" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req listRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.StartIndex != 0 || req.MaxResults != 10 {
			t.Errorf("pagination not sent: %+v", req)
		}
		if req.SearchText != "acme" {
			t.Errorf("SearchText = %q, want acme", req.SearchText)
		}
		json.NewEncoder(w).Encode(RevenueListResponse{Items: []Revenue{
			{SerialCode: "INV-001", ClientName: "ACME", Total: 1000, IsPaid: true},
		}})
	}))

	resp, err := c.ListRevenues(0, 10, "acme")
	if err != nil {
		t.Fatalf("ListRevenues: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].SerialCode != "INV-001" {
		t.Errorf("revenues not parsed: %+v", resp.Items)
	}
}

func TestListExpensesAndQueueAndRejected(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/proxy/accounting/expenses/list":
			json.NewEncoder(w).Encode(ExpenseListResponse{Items: []Expense{
				{SupplierName: "Vendor", Total: 99.99, Category: "Software"},
			}})
		case "/proxy/accounting/expenses/queued":
			json.NewEncoder(w).Encode(QueuedExpenseResponse{Items: []QueuedExpense{
				{Id: 42, DocumentName: "receipt.pdf", DaysPassed: 3},
			}})
		case "/proxy/accounting/expenses/rejected":
			json.NewEncoder(w).Encode(RejectedExpenseResponse{Items: []RejectedExpense{
				{Id: 7, DocumentName: "blurry.jpg", Reason: "unreadable"},
			}})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))

	expenses, err := c.ListExpenses(0, 10, "")
	if err != nil || len(expenses.Items) != 1 {
		t.Errorf("ListExpenses: %v, items %d", err, len(expenses.Items))
	}
	queue, err := c.ListQueuedExpenses(0, 10, "")
	if err != nil || len(queue.Items) != 1 || queue.Items[0].Id != 42 {
		t.Errorf("ListQueuedExpenses: %v, %+v", err, queue)
	}
	rejected, err := c.ListRejectedExpenses(0, 10)
	if err != nil || len(rejected.Items) != 1 || rejected.Items[0].Reason != "unreadable" {
		t.Errorf("ListRejectedExpenses: %v, %+v", err, rejected)
	}
}

func TestDeleteExpense(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/proxy/accounting/expenses/42" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))

	if err := c.DeleteExpense(42); err != nil {
		t.Errorf("DeleteExpense: %v", err)
	}
}

func TestListEFactura(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/accounting/e-invoice/list-expenses" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(EFacturaListResponse{Items: []EFactura{
			{SerialCode: "EF-1", TotalAmount: 500, CurrencyCode: "RON", PartyName: "Telecom"},
		}})
	}))

	resp, err := c.ListEFactura(0, 10, "")
	if err != nil || len(resp.Items) != 1 {
		t.Fatalf("ListEFactura: %v, %+v", err, resp)
	}
}

func TestGetCompanyInfo(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/accounting/company/basic-profile/company_abc123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(CompanyInfoResponse{
			Ok:   true,
			Data: &CompanyInfo{Name: "Test PFA", Code1: "12345678"},
		})
	}))

	info, err := c.GetCompanyInfo("abc123")
	if err != nil {
		t.Fatalf("GetCompanyInfo: %v", err)
	}
	if info.Name != "Test PFA" {
		t.Errorf("company not parsed: %+v", info)
	}

	if _, err := c.GetCompanyInfo(""); err == nil {
		t.Error("GetCompanyInfo with empty ID should fail")
	}
}

func TestGetCompanyInfoNullData(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Ok":true,"Data":null}`))
	}))

	if _, err := c.GetCompanyInfo("abc123"); err == nil {
		t.Error("Ok:true with null Data must return an error, not (nil, nil)")
	}
}

// Server strings are sanitized at the doJSON choke point so terminal
// escape sequences from the API can never reach the terminal
func TestResponseStringSanitization(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Items":[{"SerialCode":"INV\u001b]0;pwned\u0007-1","ClientName":"ACME\u001b[2J","Total":10,"Currency":{"ShortName":"RON"}}]}`))
	}))

	resp, err := c.ListRevenues(0, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{resp.Items[0].SerialCode, resp.Items[0].ClientName} {
		if strings.ContainsAny(field, "\x1b\x07") {
			t.Errorf("control characters survived sanitization: %q", field)
		}
	}
	if !strings.Contains(resp.Items[0].ClientName, "ACME") {
		t.Errorf("printable text lost during sanitization: %q", resp.Items[0].ClientName)
	}
}

func TestGetCAENCodes(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/accounting/company/caen-codes/company_abc123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]CAENCode{
			{Id: 1, IsPrimary: true, Code: "6201", Name: "Software development"},
			{Id: 2, IsPrimary: false, Code: "6202", Name: "IT consultancy"},
		})
	}))

	codes, err := c.GetCAENCodes("abc123")
	if err != nil {
		t.Fatalf("GetCAENCodes: %v", err)
	}
	if len(codes) != 2 || !codes[0].IsPrimary || codes[1].Code != "6202" {
		t.Errorf("CAEN codes not parsed: %+v", codes)
	}

	if _, err := c.GetCAENCodes(""); err == nil {
		t.Error("GetCAENCodes with empty ID should fail")
	}
}

func TestDiscoverCompanyID(t *testing.T) {
	const id = "0123456789abcdef0123456789abcdef"
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<script>var Principal = { CompanyCode: "company_` + id + `" };</script>`))
	}))

	got, err := c.DiscoverCompanyID()
	if err != nil {
		t.Fatalf("DiscoverCompanyID: %v", err)
	}
	if got != id {
		t.Errorf("DiscoverCompanyID = %q, want %q", got, id)
	}
}

func TestDiscoverCompanyIDNotFound(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>no principal here</html>"))
	}))

	if _, err := c.DiscoverCompanyID(); err == nil {
		t.Error("DiscoverCompanyID should fail when marker absent")
	}
}

func TestUploadDocument(t *testing.T) {
	var uploadedID, confirmedID string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case len(r.URL.Path) > len("/api/local-storage/upload/") && r.URL.Path[:26] == "/api/local-storage/upload/":
			uploadedID = r.URL.Path[26:]
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("not multipart: %v", err)
			}
			json.NewEncoder(w).Encode("receipt.pdf")
		case len(r.URL.Path) > 38 && r.URL.Path[:38] == "/api/financial-documents/save/expenses":
			confirmedID = filepath.Base(r.URL.Path)
			w.Write([]byte("{}"))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))

	tmpFile := filepath.Join(t.TempDir(), "receipt.pdf")
	if err := os.WriteFile(tmpFile, []byte("%PDF-1.4 fake"), 0644); err != nil {
		t.Fatal(err)
	}

	name, err := c.UploadDocument(tmpFile)
	if err != nil {
		t.Fatalf("UploadDocument: %v", err)
	}
	if name != "receipt.pdf" {
		t.Errorf("filename = %q", name)
	}
	if uploadedID == "" || uploadedID != confirmedID {
		t.Errorf("upload/confirm ID mismatch: %q vs %q", uploadedID, confirmedID)
	}
}

func TestCookieSaveLoadRoundtrip(t *testing.T) {
	// getCookiePath builds from the home dir, so isolate it
	t.Setenv("HOME", t.TempDir())

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	// LoadCookies with no file is not an error
	loaded, err := c.LoadCookies()
	if err != nil || loaded {
		t.Errorf("LoadCookies on empty = (%v, %v), want (false, nil)", loaded, err)
	}

	// SaveCookies needs the config dir to exist
	home, _ := os.UserHomeDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "solo-cli"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write a cookie file directly: one valid solo_auth and one expired cookie
	cookies := []SavedCookie{
		{Name: "solo_auth", Value: "tok", Path: "/", Expires: time.Now().Add(24 * time.Hour)},
		{Name: "stale", Value: "x", Path: "/", Expires: time.Now().Add(-time.Hour)},
	}
	data, _ := json.Marshal(cookies)
	cookieFile := filepath.Join(home, ".config", "solo-cli", "cookies.json")
	if err := os.WriteFile(cookieFile, data, 0600); err != nil {
		t.Fatal(err)
	}

	loaded, err = c.LoadCookies()
	if err != nil || !loaded {
		t.Fatalf("LoadCookies = (%v, %v), want (true, nil)", loaded, err)
	}

	// Roundtrip back to disk and verify permissions
	if err := c.SaveCookies(); err != nil {
		t.Fatalf("SaveCookies: %v", err)
	}
	info, err := os.Stat(cookieFile)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("cookie file permissions = %o, want 0600", perm)
	}
}

func TestLoadCookiesWithoutAuthCookie(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "solo-cli")
	os.MkdirAll(dir, 0755)

	cookies := []SavedCookie{{Name: "other", Value: "x", Path: "/", Expires: time.Now().Add(time.Hour)}}
	data, _ := json.Marshal(cookies)
	os.WriteFile(filepath.Join(dir, "cookies.json"), data, 0600)

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	loaded, err := c.LoadCookies()
	if err != nil || loaded {
		t.Errorf("LoadCookies without solo_auth = (%v, %v), want (false, nil)", loaded, err)
	}
}
