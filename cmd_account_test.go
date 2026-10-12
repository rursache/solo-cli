package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newAccountMock wraps the e2e mock API with the documents, reports and
// account endpoints, everything else is proxied to the base mock
func newAccountMock(t *testing.T) (*mockAPI, *[]string) {
	t.Helper()
	api := newMockAPI(t)
	target, _ := url.Parse(api.server.URL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	var rateBodies []string

	startYear := time.Now().Year() - 2
	mux := http.NewServeMux()
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<script>var Principal = { CompanyCode: "company_%s", CompanyStartYear: %d };</script>`, mockCompanyID, startYear)
	})
	mux.HandleFunc("/proxy/accounting/expenses/list", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Items":[{"SupplierName":"Hosting SRL","Total":99.99,"Category":"Servicii","Currency":{"ShortName":"RON"},"DocumentCode":"doc_exp1"},{"SupplierName":"Cafe SRL","Total":10,"Category":"Protocol","Currency":{"ShortName":"RON"},"DocumentCode":null}]}`)
	})
	mux.HandleFunc("/proxy/accounting/expenses/queued", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Items":[{"Id":42,"DocumentCode":"doc_q42","DocumentName":"receipt.pdf","DaysPassed":3,"IsOverdue":true}]}`)
	})
	mux.HandleFunc("/proxy/accounting/expenses/rejected", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Items":[{"Id":7,"DocumentCode":"doc_rej7","DocumentName":"blurry.jpg","Reason":"unreadable"}]}`)
	})
	mux.HandleFunc("/api/documents/tax-declarations", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Documents":[
			{"TaxDeclarationType":"D212","ReportingPeriod":"2025","FiledOn":"2026-05-20T10:00:00+03:00","UniqueCode":"doc_d212","MimeType":"application/pdf","OriginalFileName":"D212_test.pdf"},
			{"TaxDeclarationType":"D300","ReportingPeriod":"august 2026","FiledOn":"2026-09-25T10:00:00+03:00","UniqueCode":"doc_d300","MimeType":"application/pdf","OriginalFileName":""}
		]}`)
	})
	mux.HandleFunc("/api/documents/registration-documents", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Sections":[
			{"DocumentTypeCode":"trade_register","Documents":[{"UniqueCode":"doc_reg1","OriginalFileName":"certificat.pdf","DocumentTypeCode":"trade_register","MimeType":"application/pdf","ModifiedOn":"2024-01-10T09:00:00+02:00"}],"HasDocuments":true},
			{"DocumentTypeCode":"trade_certificate_vat","Documents":[],"HasDocuments":false}
		]}`)
	})
	mux.HandleFunc("/api/documents/download/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/missing") {
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		fmt.Fprint(w, "%PDF-1.4 "+filepath.Base(r.URL.Path))
	})
	mux.HandleFunc("/api/financial-documents/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	})
	mux.HandleFunc("/proxy/accounting/reports/cashflow-report/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/proxy/accounting/reports/cashflow-report/company_"+mockCompanyID+"/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		fmt.Fprint(w, "%PDF-1.7 register "+filepath.Base(r.URL.Path))
	})
	mux.HandleFunc("/proxy/accounting/company/vies-profile/company_"+mockCompanyID, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Data":{"Code3":"RO11111111","ViesProfileStatus":"VatRegistered","ViesWorkflowStatus":"None"},"Ok":true}`)
	})
	mux.HandleFunc("/proxy/accounting/bank-accounts/company_"+mockCompanyID, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"Id":1,"Bank":{"Id":1,"Name":"Test Bank"},"BankName":null,"IBAN":"RO00TEST0000000000000001","Currency":{"Code":"RON","ShortName":"lei"}},{"Id":2,"Bank":null,"BankName":"Custom Bank","IBAN":"RO00CUST0000000000000002","Currency":{"Code":"EUR"}}]`)
	})
	mux.HandleFunc("/proxy/accounting/subscription/summary", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"SubscriptionSummary":{"Subscription":{"Name":"Test Plan","Status":"Overdue","IsCollectionPaused":true,"NextBillingDate":"2027-01-01T00:00:00","ProductPrice":1000},"OverduePaymentSummary":{"Total":100,"PaymentCount":2},"DefaultCard":{"ExpiringMonth":8,"ExpiringYear":2099,"IsExpired":false,"LastDigits":"0000","CardBrand":"visa"}},"ChangeOptions":{"CurrentSubscription":{"Name":"Test Plan","BillingPeriod":"yearly"}},"ScheduledChange":null}`)
	})
	mux.HandleFunc("/proxy/accounting/subscription/benefits", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"HasProviderHostedHeadquarter":true}`)
	})
	mux.HandleFunc("/api/referral/campaign", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ReferralCode":"test123","CurrentCampaign":{"ReferrerPrizeValue":100,"GuestPrizeValue":50,"IsTimeLimited":true,"StartDate":"2026-01-01T00:00:00","EndDate":"2026-12-31T00:00:00"}}`)
	})
	mux.HandleFunc("/api/referral/rewards", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"PendingWinnings":50,"EarnedWinnings":200}`)
	})
	mux.HandleFunc("/proxy/accounting/customers/list", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SearchText string
			StartIndex int
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.StartIndex > 0 || req.SearchText == "nobody" {
			fmt.Fprint(w, `{"Items":[],"TotalResults":null}`)
			return
		}
		fmt.Fprint(w, `{"Items":[{"CustomerId":7,"Name":"Client SRL","Code1":"RO100","Code2":"J40/1/2020","Email":"client@example.com","Phone":null}],"TotalResults":null}`)
	})
	mux.HandleFunc("/proxy/accounting/utility/exchange-rates", func(w http.ResponseWriter, r *http.Request) {
		var body string
		json.NewDecoder(r.Body).Decode(&body)
		rateBodies = append(rateBodies, body)
		fmt.Fprint(w, `[{"FromCurrency":{"Id":2,"Code":"EUR","ShortName":"eur"},"ToCurrency":{"Id":1,"Code":"RON"},"RateValue":5,"RateDate":"2026-01-05T00:00:00+02:00"}]`)
	})
	mux.Handle("/", proxy)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	api.server = server
	return api, &rateBodies
}

// runIn is env.run with a working directory, so default download names land
// in a temp dir instead of the repo
func (e *env) runIn(t *testing.T, api *mockAPI, dir string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binPath, append([]string{"--config", e.configPath}, args...)...)
	cmd.Dir = dir
	ssoBase := strings.Replace(api.sso.URL, "127.0.0.1", "localhost", 1)
	cmd.Env = append(os.Environ(), "HOME="+e.home, "SOLO_API_BASE="+api.server.URL, "SOLO_SSO_BASE="+ssoBase)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running binary: %v", err)
	}
	return stdout.String(), stderr.String(), code
}

// expectOutputs runs each command and compares stdout exactly
func expectOutputs(t *testing.T, e *env, api *mockAPI, dir string, cases map[string]string) {
	t.Helper()
	for args, want := range cases {
		out, errOut, code := e.runIn(t, api, dir, strings.Split(args, "|")...)
		if code != 0 || out != want {
			t.Errorf("%s: exit %d\nstdout %q\nwant   %q\nstderr %q", args, code, out, want, errOut)
		}
	}
}

func readIn(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(data)
}

func TestDocumentsCommands(t *testing.T) {
	api, _ := newAccountMock(t)
	e := newEnv(t, "good-password")
	dir := t.TempDir()

	expectOutputs(t, e, api, dir, map[string]string{
		"documents":         "D212\t2025\t2026-05-20\tdoc_d212\nD300\taugust 2026\t2026-09-25\tdoc_d300\n",
		"docs|pfa":          "Acte infiintare\tdoc_reg1\tcertificat.pdf\t2024-01-10\n",
		"expenses":          "99.99 RON\tServicii\tHosting SRL\tdoc_exp1\n10.00 RON\tProtocol\tCafe SRL\n",
		"queue":             "receipt.pdf\t3 days\tOVERDUE\t(ID: 42)\tdoc_q42\n",
		"expenses|rejected": "blurry.jpg\tunreadable\t(ID: 7)\tdoc_rej7\n",
	})

	_, errOut, _ := e.runIn(t, api, dir, "expenses")
	if !strings.Contains(errOut, "blurry.jpg - unreadable (ID: 7, document doc_rej7)") {
		t.Errorf("rejected warning must name the id and document code: %q", errOut)
	}
}

// Typos and stray arguments fail before signing in instead of running the
// plain listing
func TestAccountArgumentErrors(t *testing.T) {
	api, _ := newAccountMock(t)
	e := newEnv(t, "good-password")
	for want, args := range map[string][]string{
		`unknown documents subcommand "bogus"`: {"documents", "bogus"},
		`unknown expenses subcommand "bogus"`:  {"expenses", "bogus"},
		`unknown expenses subcommand "-h"`:     {"expenses", "-h"},
		`unexpected argument "x"`:              {"expenses", "rejected", "x"},
		`unexpected argument "extra"`:          {"documents", "pfa", "extra"},
		`unexpected argument "more"`:           {"company", "more"},
	} {
		out, errOut, code := e.runIn(t, api, t.TempDir(), args...)
		if code != 1 || out != "" || !strings.Contains(errOut, want) {
			t.Errorf("%v: exit %d, out %q, stderr %q", args, code, out, errOut)
		}
	}
	if hits := api.ssoHits.Load(); hits != 0 {
		t.Errorf("argument errors signed in %d times", hits)
	}
}

func TestDocumentDownloads(t *testing.T) {
	api, _ := newAccountMock(t)
	e := newEnv(t, "good-password")
	dir := t.TempDir()

	out, errOut, code := e.runIn(t, api, dir, "documents", "download", "doc_d212")
	if code != 0 || out != "D212_test.pdf\n" || readIn(t, dir, "D212_test.pdf") != "%PDF-1.4 doc_d212" {
		t.Fatalf("documents download: %d %q %q", code, out, errOut)
	}
	_, errOut, code = e.runIn(t, api, dir, "documents", "download", "doc_d212")
	if code != 1 || !strings.Contains(errOut, "already exists") || !strings.Contains(errOut, "--force") {
		t.Errorf("overwrite must be refused: %d %q", code, errOut)
	}
	if out, _, code := e.runIn(t, api, dir, "documents", "download", "doc_d212", "-f"); code != 0 || out != "D212_test.pdf\n" {
		t.Errorf("--force: %d %q", code, out)
	}

	// No original file name falls back to type and period, PFA files keep theirs
	if out, _, code := e.runIn(t, api, dir, "docs", "dl", "doc_d300"); code != 0 || out != "D300 august 2026.pdf\n" {
		t.Errorf("declaration without file name: %d %q", code, out)
	}
	if out, _, code := e.runIn(t, api, dir, "documents", "download", "doc_reg1"); code != 0 || out != "certificat.pdf\n" {
		t.Errorf("pfa download: %d %q", code, out)
	}
	if out, _, code := e.runIn(t, api, dir, "documents", "download", "doc_other", "-o", "custom.pdf"); code != 0 || out != "custom.pdf\n" || readIn(t, dir, "custom.pdf") != "%PDF-1.4 doc_other" {
		t.Errorf("-o: %d %q", code, out)
	}
	if out, errOut, code := e.runIn(t, api, dir, "documents", "download", "missing"); code != 1 || out != "" || !strings.Contains(errOut, "empty file") {
		t.Errorf("empty answer must fail: %d %q %q", code, out, errOut)
	}
	if _, errOut, code := e.runIn(t, api, dir, "documents", "download"); code != 1 || !strings.Contains(errOut, "missing document code") {
		t.Errorf("missing code: %d %q", code, errOut)
	}

	if out, errOut, code := e.runIn(t, api, dir, "expenses", "download", "doc_exp1"); code != 0 || out != "doc_exp1.jpg\n" {
		t.Errorf("expenses download: %d %q %q", code, out, errOut)
	}
	if _, errOut, code := e.runIn(t, api, dir, "expenses", "download"); code != 1 || !strings.Contains(errOut, "missing document code") {
		t.Errorf("expenses download without code: %d %q", code, errOut)
	}
}

func TestReportCommand(t *testing.T) {
	api, _ := newAccountMock(t)
	e := newEnv(t, "good-password")
	dir := t.TempDir()

	year := time.Now().Year()
	name := fmt.Sprintf("Registru-incasari-plati-%d.pdf", year)
	if out, errOut, code := e.runIn(t, api, dir, "report"); code != 0 || out != name+"\n" || readIn(t, dir, name) != fmt.Sprintf("%%PDF-1.7 register %d", year) {
		t.Errorf("report: %d %q %q", code, out, errOut)
	}
	sub := filepath.Join(dir, "reports")
	os.Mkdir(sub, 0755)
	want := filepath.Join(sub, fmt.Sprintf("Registru-incasari-plati-%d.pdf", year-2))
	if out, errOut, code := e.runIn(t, api, dir, "report", fmt.Sprint(year-2), "-o", sub); code != 0 || out != want+"\n" {
		t.Errorf("report into a directory: %d %q %q", code, out, errOut)
	}
	for _, bad := range []string{fmt.Sprint(year - 3), fmt.Sprint(year + 1)} {
		if _, errOut, code := e.runIn(t, api, dir, "report", bad); code != 1 || !strings.Contains(errOut, "available years are") {
			t.Errorf("report %s: %d %q", bad, code, errOut)
		}
	}
	if _, errOut, code := e.runIn(t, api, dir, "report", "last"); code != 1 || !strings.Contains(errOut, "invalid year") {
		t.Errorf("report with a bad year: %d %q", code, errOut)
	}
}

func TestCompanyExtras(t *testing.T) {
	api, _ := newAccountMock(t)
	e := newEnv(t, "good-password")

	out, _, code := e.runIn(t, api, t.TempDir(), "company")
	wantTail := "VIES status: VatRegistered\nVIES code: RO11111111\nBank: Test Bank - RO00TEST0000000000000001 (RON)\nBank: Custom Bank - RO00CUST0000000000000002 (EUR)\n"
	if code != 0 || !strings.HasPrefix(out, "Name: Test PFA\n") || !strings.HasSuffix(out, "CAEN: 6202 - IT consultancy\n"+wantTail) {
		t.Errorf("company output: %q", out)
	}

	// Without the VIES and bank endpoints company must still work
	base := newMockAPI(t)
	baseHandler := base.server.Config.Handler
	base.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/vies-profile/") || strings.Contains(r.URL.Path, "/bank-accounts/") {
			http.NotFound(w, r)
			return
		}
		baseHandler.ServeHTTP(w, r)
	})
	out, errOut, code := e.run(t, base, "company")
	if code != 0 || !strings.HasSuffix(out, "CAEN: 6202 - IT consultancy\n") || !strings.Contains(errOut, "Warning: failed to get VIES profile") {
		t.Errorf("company without extras: %d %q %q", code, out, errOut)
	}
}

func TestSubscriptionAndReferral(t *testing.T) {
	api, _ := newAccountMock(t)
	e := newEnv(t, "good-password")

	expectOutputs(t, e, api, t.TempDir(), map[string]string{
		"referral":     "Link: https://www.solo.ro/ref/test123\nPrize: 100.00 RON for you, 50.00 RON for the invited friend\nPeriod: 2026-01-01 - 2026-12-31\nEarned: 200.00 RON\nPending: 50.00 RON\n",
		"subscription": "Plan: Test Plan (yearly)\nStatus: Paused\nPrice: 1000.00 RON/year\nNext billing: 2027-01-01\nCard: Visa ending in 0000, expires 08/2099\nOverdue: 100.00 RON (2 payments)\nHosted headquarters: yes\nManage: " + api.server.URL + "/settings/subscription\n",
	})
}

func TestClientsAndRates(t *testing.T) {
	api, rateBodies := newAccountMock(t)
	e := newEnv(t, "good-password")
	dir := t.TempDir()

	expectOutputs(t, e, api, dir, map[string]string{
		"clients":          "7\tClient SRL\tRO100\tJ40/1/2020\tclient@example.com\t\n",
		"rates|2026-01-05": "EUR\t5.0000\t2026-01-05\n",
	})
	if len(*rateBodies) != 1 || (*rateBodies)[0] != "2026-01-05T00:00:00+02:00" {
		t.Errorf("rates body = %q", *rateBodies)
	}

	out, errOut, code := e.runIn(t, api, dir, "clients", "nobody")
	if code != 0 || out != "" || !strings.Contains(errOut, "No clients found") {
		t.Errorf("empty clients: %d %q %q", code, out, errOut)
	}
	if _, errOut, code := e.runIn(t, api, dir, "rates", "05.01.2026"); code != 1 || !strings.Contains(errOut, "YYYY-MM-DD") {
		t.Errorf("rates with a bad date: %d %q", code, errOut)
	}
}
