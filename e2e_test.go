package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// The e2e suite builds the real binary once and runs it against a mock
// SOLO.ro API with an isolated HOME, exercising the full stack: flag
// parsing, config and cookie files, login flow and command output

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "solo-cli-e2e")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binPath = filepath.Join(dir, "solo-cli")
	if out, err := exec.Command("go", "build", "-o", binPath, ".").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building binary: %v\n%s", err, out)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// mockAPI is a fake SOLO.ro backend with hit counters for behavioral asserts
type mockAPI struct {
	server     *httptest.Server
	sso        *httptest.Server
	loginHits  atomic.Int32
	ssoHits    atomic.Int32
	bridgeHits atomic.Int32
	deleteHits atomic.Int32
	uploadHits atomic.Int32
	ssoBroken  atomic.Bool
}

// Minimal cont.solo.ro pages, enough to drive the real SSO parser
const (
	e2eEmailPage = `<html><body><form method="post" id="emailForm" action="/login">
<input type="email" name="Input.Email" value="" /><input name="__RequestVerificationToken" type="hidden" value="t1" /></form></body></html>`
	e2ePasswordPage = `<html><body><form method="post" id="passwordForm" action="/login">
<input type="hidden" name="Input.Email" value="user@example.com" /><input type="password" name="Input.Password" />
%s<input name="__RequestVerificationToken" type="hidden" value="t2" /></form></body></html>`
	e2eRecommendationPage = `<html><body>
<form method="post" action="/login?handler=ActivateMfa"><button type="submit">Activate</button><input name="__RequestVerificationToken" type="hidden" value="t3" /></form>
<form method="post" action="/login?handler=SkipMfa"><button type="submit">Skip</button><input name="__RequestVerificationToken" type="hidden" value="t4" /></form></body></html>`
	e2eMFAPage = `<html><body><p class="subtitle">Code sent by SMS</p><form method="post" id="mfaForm" action="/login?handler=VerifyMfa">
<input type="hidden" id="otpCodeHidden" name="Input.Code" value="" /><input type="checkbox" name="Input.RememberDevice" value="true" />
<button type="submit" id="mfaSubmitBtn">Verify</button><input name="__RequestVerificationToken" type="hidden" value="t5" /></form></body></html>`
	e2eSuccessPage = `<html><body><div id="loginSuccess" data-redirect-url="%s/bridge?redirect=%%2Fdashboard"></div></body></html>`
)

const mockCompanyID = "0123456789abcdef0123456789abcdef"

func newMockAPI(t *testing.T) *mockAPI {
	t.Helper()
	m := &mockAPI{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/security/login", func(w http.ResponseWriter, r *http.Request) {
		m.loginHits.Add(1)
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["Password"] != "good-password" {
			json.NewEncoder(w).Encode(map[string]string{"AuthenticationStatus": "FAILED"})
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "solo_auth", Value: "e2e-token", Path: "/"})
		json.NewEncoder(w).Encode(map[string]string{"AuthenticationStatus": "OK"})
	})
	mux.HandleFunc("/bridge", func(w http.ResponseWriter, r *http.Request) {
		m.bridgeHits.Add(1)
		http.SetCookie(w, &http.Cookie{Name: "solo_auth", Value: "e2e-token", Path: "/"})
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	})
	mux.HandleFunc("/proxy/accounting/dashboard/summary", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("solo_auth"); err != nil || c.Value != "e2e-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{"Year":2026,"DisplayCurrency":"RON","TotalRevenues":50000,"TotalDeductibleExpenses":20000,"HasTaxes":true,"Taxes":6000}`)
	})
	mux.HandleFunc("/proxy/accounting/revenues/list", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Items":[
			{"SerialCode":"INV-001","ClientName":"ACME Corp","Total":1000.50,"IsPaid":true,"Currency":{"ShortName":"RON"}},
			{"SerialCode":"INV-002","ClientName":"Globex","Total":250.25,"IsPaid":false,"Currency":{"ShortName":"EUR"}}
		]}`)
	})
	mux.HandleFunc("/proxy/accounting/expenses/list", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Items":[{"SupplierName":"Hosting SRL","Total":99.99,"Category":"Servicii","Currency":{"ShortName":"RON"}}]}`)
	})
	mux.HandleFunc("/proxy/accounting/expenses/rejected", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Items":[{"Id":7,"DocumentName":"blurry.jpg","Reason":"unreadable"}]}`)
	})
	mux.HandleFunc("/proxy/accounting/expenses/queued", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Items":[{"Id":42,"DocumentName":"receipt.pdf","DaysPassed":3,"IsOverdue":true}]}`)
	})
	mux.HandleFunc("/proxy/accounting/expenses/42", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			m.deleteHits.Add(1)
		}
		fmt.Fprint(w, `{}`)
	})
	mux.HandleFunc("/proxy/accounting/e-invoice/list-expenses", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Items":[{"SerialCode":"EF-9","TotalAmount":500,"CurrencyCode":"RON","InvoiceDate":"2026-06-01","PartyName":"Telecom SA"}]}`)
	})
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<script>var Principal = { CompanyCode: "company_%s" };</script>`, mockCompanyID)
	})
	mux.HandleFunc("/proxy/accounting/company/basic-profile/company_"+mockCompanyID, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Ok":true,"Data":{"Name":"Test PFA","Code1":"11111111","Code2":"F1/1/2026","Address":"Str. Exemplu 1"}}`)
	})
	mux.HandleFunc("/proxy/accounting/company/caen-codes/company_"+mockCompanyID, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"Id":1,"IsPrimary":true,"Code":"6201","Name":"Software development"},{"Id":2,"IsPrimary":false,"Code":"6202","Name":"IT consultancy"}]`)
	})
	mux.HandleFunc("/api/local-storage/upload/", func(w http.ResponseWriter, r *http.Request) {
		m.uploadHits.Add(1)
		json.NewEncoder(w).Encode("receipt.pdf")
	})
	mux.HandleFunc("/api/financial-documents/save/expenses/", func(w http.ResponseWriter, r *http.Request) {
		m.uploadHits.Add(1)
		fmt.Fprint(w, `1`)
	})

	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)

	m.sso = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.ssoHits.Add(1)
		r.ParseForm()
		if m.ssoBroken.Load() || r.URL.Path != "/login" {
			fmt.Fprint(w, "<html><body><h1>Maintenance</h1></body></html>")
			return
		}
		switch handler := r.URL.Query().Get("handler"); {
		case r.Method == http.MethodGet:
			fmt.Fprint(w, e2eEmailPage)
		case handler == "SkipMfa":
			fmt.Fprintf(w, e2eSuccessPage, m.server.URL)
		case handler != "":
			t.Errorf("unexpected SSO handler %q", handler)
		case r.PostForm.Get("Input.Password") == "":
			fmt.Fprintf(w, e2ePasswordPage, "")
		case r.PostForm.Get("Input.Password") == "good-password":
			fmt.Fprint(w, e2eRecommendationPage)
		case r.PostForm.Get("Input.Password") == "mfa-password":
			fmt.Fprint(w, e2eMFAPage)
		default:
			fmt.Fprintf(w, e2ePasswordPage, `<span class="field-validation-error">Email sau parolă greșită</span>`)
		}
	}))
	t.Cleanup(m.sso.Close)
	return m
}

// env is one isolated user environment: a HOME with config and cookie files
type env struct {
	home       string
	configPath string
}

func newEnv(t *testing.T, password string) *env {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "solo-cli")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if password != "" {
		cfg := fmt.Sprintf(`{"username":"user@example.com","password":"%s","page_size":100}`, password)
		if err := os.WriteFile(configPath, []byte(cfg), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return &env{home: home, configPath: configPath}
}

// run executes the real binary and returns stdout, stderr and the exit code
func (e *env) run(t *testing.T, api *mockAPI, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binPath, append([]string{"--config", e.configPath}, args...)...)
	// The SSO mock is addressed as localhost so its cookies stay apart from
	// the falcon mock on 127.0.0.1, like cont.solo.ro and falcon.solo.ro
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

func TestE2EVersionAndHelp(t *testing.T) {
	api := newMockAPI(t)
	e := newEnv(t, "good-password")

	out, _, code := e.run(t, api, "version")
	if code != 0 || out != "solo-cli dev\n" {
		t.Errorf("version: code %d, out %q", code, out)
	}

	out, _, code = e.run(t, api, "help")
	if code != 0 {
		t.Errorf("help exit code %d", code)
	}
	for _, cmd := range []string{"summary", "taxes", "revenues", "expenses", "queue", "efactura", "company", "upload", "setup-skills", "demo"} {
		if !strings.Contains(out, cmd) {
			t.Errorf("help output missing command %q", cmd)
		}
	}
}

func TestE2EUnknownCommand(t *testing.T) {
	api := newMockAPI(t)
	e := newEnv(t, "good-password")

	_, errOut, code := e.run(t, api, "bogus")
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut, "Unknown command: bogus") {
		t.Errorf("stderr missing unknown command message: %q", errOut)
	}
}

func TestE2EMissingCredentials(t *testing.T) {
	api := newMockAPI(t)
	e := newEnv(t, "") // no config file

	_, errOut, code := e.run(t, api, "summary")
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut, "credentials missing") {
		t.Errorf("stderr missing credentials error: %q", errOut)
	}

	// The empty config must have been auto-created with safe permissions
	info, err := os.Stat(e.configPath)
	if err != nil {
		t.Fatalf("config not auto-created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config permissions = %o, want 0600", perm)
	}
}

func TestE2EInvalidLogin(t *testing.T) {
	api := newMockAPI(t)
	e := newEnv(t, "wrong-password")

	_, errOut, code := e.run(t, api, "summary")
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut, "check your credentials") {
		t.Errorf("stderr missing credentials hint: %q", errOut)
	}
}

func TestE2ESummaryAndSessionReuse(t *testing.T) {
	api := newMockAPI(t)
	e := newEnv(t, "good-password")

	out, errOut, code := e.run(t, api, "summary")
	if code != 0 {
		t.Fatalf("summary failed (%d): %s", code, errOut)
	}
	want := "Year: 2026\nRevenues: 50000.00 RON\nExpenses: 20000.00 RON\nTaxes: 6000.00 RON\n"
	if out != want {
		t.Errorf("summary output:\n%q\nwant:\n%q", out, want)
	}
	if got := api.bridgeHits.Load(); got != 1 {
		t.Errorf("SSO bridge hits = %d, want 1", got)
	}
	if got := api.loginHits.Load(); got != 0 {
		t.Errorf("legacy login hits = %d, want 0", got)
	}

	// Cookies must persist with safe permissions and be reused on the next run
	cookiePath := filepath.Join(e.home, ".config", "solo-cli", "cookies.json")
	info, err := os.Stat(cookiePath)
	if err != nil {
		t.Fatalf("cookies not saved: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("cookie permissions = %o, want 0600", perm)
	}

	out, _, code = e.run(t, api, "summary")
	if code != 0 || out != want {
		t.Errorf("second summary run: code %d, out %q", code, out)
	}
	if got := api.bridgeHits.Load(); got != 1 {
		t.Errorf("SSO logins after cookie reuse = %d, want still 1", got)
	}
}

func TestE2ELegacyFallback(t *testing.T) {
	api := newMockAPI(t)
	api.ssoBroken.Store(true)
	e := newEnv(t, "good-password")

	out, errOut, code := e.run(t, api, "summary")
	if code != 0 || !strings.HasPrefix(out, "Year: 2026\n") {
		t.Fatalf("summary via legacy login: code %d, out %q, stderr %q", code, out, errOut)
	}
	if api.loginHits.Load() != 1 || api.bridgeHits.Load() != 0 {
		t.Errorf("legacy hits %d, bridge hits %d", api.loginHits.Load(), api.bridgeHits.Load())
	}
	if !strings.Contains(errOut, "legacy login") {
		t.Errorf("stderr missing fallback note: %q", errOut)
	}
}

// Without a terminal a 2FA account must fail clearly, and once known it
// must not even start a login that would text another code
func TestE2EMFANonInteractive(t *testing.T) {
	api := newMockAPI(t)
	e := newEnv(t, "mfa-password")

	_, errOut, code := e.run(t, api, "summary")
	if code != 1 || !strings.Contains(errOut, "two-factor") || !strings.Contains(errOut, "in a terminal") {
		t.Fatalf("first run: code %d, stderr %q", code, errOut)
	}
	if api.loginHits.Load() != 0 {
		t.Error("2FA page must never fall back to the legacy login")
	}
	if _, err := os.Stat(filepath.Join(e.home, ".config", "solo-cli", ".mfa-required")); err != nil {
		t.Errorf("2FA flag not saved: %v", err)
	}

	hits := api.ssoHits.Load()
	_, errOut, code = e.run(t, api, "summary")
	if code != 1 || !strings.Contains(errOut, "two-factor") {
		t.Errorf("second run: code %d, stderr %q", code, errOut)
	}
	if got := api.ssoHits.Load(); got != hits {
		t.Errorf("second run contacted cont.solo.ro %d times, want 0", got-hits)
	}
}

func TestE2EListCommands(t *testing.T) {
	api := newMockAPI(t)
	e := newEnv(t, "good-password")

	out, _, code := e.run(t, api, "revenues")
	if code != 0 {
		t.Fatalf("revenues exit %d", code)
	}
	wantRev := "INV-001\t1000.50 RON\tPAID\tACME Corp\nINV-002\t250.25 EUR\tUNPAID\tGlobex\n"
	if out != wantRev {
		t.Errorf("revenues output:\n%q\nwant:\n%q", out, wantRev)
	}

	out, errOut, code := e.run(t, api, "expenses")
	if code != 0 {
		t.Fatalf("expenses exit %d", code)
	}
	if out != "99.99 RON\tServicii\tHosting SRL\n" {
		t.Errorf("expenses output: %q", out)
	}
	// Rejected expenses must surface as a stderr warning, not pollute stdout
	if !strings.Contains(errOut, "blurry.jpg") || !strings.Contains(errOut, "unreadable") {
		t.Errorf("stderr missing rejected warning: %q", errOut)
	}

	out, _, code = e.run(t, api, "queue")
	if code != 0 {
		t.Fatalf("queue exit %d", code)
	}
	if out != "receipt.pdf\t3 days\tOVERDUE\t(ID: 42)\n" {
		t.Errorf("queue output: %q", out)
	}

	out, _, code = e.run(t, api, "efactura")
	if code != 0 {
		t.Fatalf("efactura exit %d", code)
	}
	if out != "EF-9\t500.00 RON\t2026-06-01\tTelecom SA\n" {
		t.Errorf("efactura output: %q", out)
	}

	out, _, code = e.run(t, api, "company")
	if code != 0 {
		t.Fatalf("company exit %d", code)
	}
	if !strings.Contains(out, "Test PFA") || !strings.Contains(out, "11111111") {
		t.Errorf("company output: %q", out)
	}
	if !strings.Contains(out, "CAEN: 6201 - Software development (principal)") || !strings.Contains(out, "CAEN: 6202 - IT consultancy\n") {
		t.Errorf("company output missing CAEN codes: %q", out)
	}
}

func TestE2EQueueDelete(t *testing.T) {
	api := newMockAPI(t)
	e := newEnv(t, "good-password")

	out, _, code := e.run(t, api, "queue", "delete", "42")
	if code != 0 {
		t.Fatalf("queue delete exit %d", code)
	}
	if !strings.Contains(out, "deleted successfully") {
		t.Errorf("delete output: %q", out)
	}
	if got := api.deleteHits.Load(); got != 1 {
		t.Errorf("DELETE hits = %d, want 1", got)
	}

	_, errOut, code := e.run(t, api, "queue", "delete", "abc")
	if code != 1 || !strings.Contains(errOut, "invalid ID") {
		t.Errorf("invalid id: code %d, stderr %q", code, errOut)
	}
}

func TestE2EUpload(t *testing.T) {
	api := newMockAPI(t)
	e := newEnv(t, "good-password")

	file := filepath.Join(t.TempDir(), "receipt.pdf")
	if err := os.WriteFile(file, []byte("%PDF-1.4 fake"), 0644); err != nil {
		t.Fatal(err)
	}

	out, _, code := e.run(t, api, "upload", file)
	if code != 0 {
		t.Fatalf("upload exit %d", code)
	}
	if !strings.Contains(out, "Uploaded: receipt.pdf") {
		t.Errorf("upload output: %q", out)
	}
	// Both the multipart upload and the confirm call must have happened
	if got := api.uploadHits.Load(); got != 2 {
		t.Errorf("upload+confirm hits = %d, want 2", got)
	}

	_, errOut, code := e.run(t, api, "upload", "/nonexistent.pdf")
	if code != 1 || !strings.Contains(errOut, "file not found") {
		t.Errorf("missing file: code %d, stderr %q", code, errOut)
	}
}

// The CLI taxes command must produce the exact numbers taxes.Calculate
// produces for the TUI: net 30000 at SMB 4050 is 7.4 salarii, so CAS is
// exempt, CASS is proportional 3000 and income tax is 2700
func TestE2ETaxesMathAndConfigFile(t *testing.T) {
	api := newMockAPI(t)
	e := newEnv(t, "good-password")

	out, errOut, code := e.run(t, api, "taxes")
	if code != 0 {
		t.Fatalf("taxes failed (%d): %s", code, errOut)
	}

	for _, want := range []string{
		"Net Income:           30000.00 RON",
		"Fără CAS (sub 12 salarii)",
		"CASS (10%): CASS proporțional",
		"Base: 30000.00 RON → Amount: 3000.00 RON",
		"Income Tax (10%): 2700.00 RON",
		"Total Taxes:          5700.00 RON",
		"Net After Tax:        24300.00 RON",
		"Effective Tax Rate:   19.0%",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("taxes output missing %q\nfull output:\n%s", want, out)
		}
	}

	// taxes.json must have been auto-created with the correct 2026 defaults
	taxesPath := filepath.Join(e.home, ".config", "solo-cli", "taxes.json")
	data, err := os.ReadFile(taxesPath)
	if err != nil {
		t.Fatalf("taxes.json not created: %v", err)
	}
	var taxCfg struct {
		SalariuMinimBrut float64 `json:"salariu_minim_brut"`
	}
	if err := json.Unmarshal(data, &taxCfg); err != nil {
		t.Fatalf("taxes.json invalid: %v", err)
	}
	if taxCfg.SalariuMinimBrut != 4050 {
		t.Errorf("salariu_minim_brut = %v, want 4050 (January 1 2026 value)", taxCfg.SalariuMinimBrut)
	}
}
