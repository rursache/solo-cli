package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Trimmed from the live cont.solo.ro pages with the account anonymised. The
// 2FA code page follows the ids and classes login.js expects

const fixtureEmail = "test@exemplu.ro"

func ssoPageHTML(main string) string {
	return `<!DOCTYPE html><html lang="ro"><head><meta charset="UTF-8" />
<script nonce="abc">(function(){ document.write('<form id="scriptForm" action="/nope"><input name="x" /></form>'); })();</script>
<title>SOLO &#x2013; Intr&#x103; &#xEE;n cont</title></head>
<body><div class="app"><main class="content">` + main + `</main></div></body></html>`
}

var fixtureEmailPage = ssoPageHTML(`
<h1>Bine ai revenit!</h1>
<p class="subtitle">Introdu emailul</p>
<form method="post" class="form" id="emailForm" action="/login">
  <input type="email" required id="Input_Email" name="Input.Email" value="" />
  <span class="text-danger field-validation-valid" data-valmsg-for="Input.Email"></span>
  <button type="submit" class="btn-primary" id="emailSubmitBtn"><span>Continuă</span></button>
<input name="__RequestVerificationToken" type="hidden" value="token-email" /></form>`)

func fixturePasswordPage(errText string) string {
	validation := `<span class="text-danger field-validation-valid" data-valmsg-for="Input.Password"></span>`
	if errText != "" {
		validation = `<span class="text-danger field-validation-error" data-valmsg-for="Input.Password">` + errText + `</span>`
	}
	return ssoPageHTML(`
<form method="post" class="form" novalidate id="passwordForm" action="/login">
  <input type="hidden" id="Input_Email" name="Input.Email" value="` + fixtureEmail + `" />
  <input
    type="email"
    value="` + fixtureEmail + `"
    readonly
  />
  <input type="password" placeholder="Introdu parola" id="passwordInput" name="Input.Password" />
  <button type="button" class="password-toggle" id="passwordToggle">eye</button>
  ` + validation + `
  <button type="submit" class="btn-primary" id="submitBtn">Conectează-te</button>
<input name="__RequestVerificationToken" type="hidden" value="token-password" /></form>`)
}

var fixtureRecommendationPage = ssoPageHTML(`
<div class="mfa-recommendation">
  <form method="post" action="/login?handler=ActivateMfa">
    <button type="submit" class="btn-primary" id="activateMfaBtn">Activează</button>
  <input name="__RequestVerificationToken" type="hidden" value="token-activate" /></form>
  <form method="post" action="/login?handler=SkipMfa">
    <button type="submit" class="btn-secondary" id="skipMfaBtn">Continuă fără activare</button>
  <input name="__RequestVerificationToken" type="hidden" value="token-skip" /></form>
</div>`)

func fixtureSuccessPage(redirect string) string {
	return ssoPageHTML(`<div id="loginSuccess" class="login-success" data-redirect-url="` + redirect + `" data-login-type="client">
<p class="subtitle">Te conectăm...</p><noscript><a href="` + redirect + `">Continuă</a></noscript></div>`)
}

func fixtureMFAPage(errText string) string {
	alert := ""
	if errText != "" {
		alert = `<div class="alert alert-danger" id="otpErrorAlert" role="alert">` + errText + `</div>`
	}
	var digits strings.Builder
	for i := 1; i <= 6; i++ {
		fmt.Fprintf(&digits, `<input type="text" inputmode="numeric" maxlength="1" class="otp-digit" aria-label="Cifra %d" />`, i)
	}
	return ssoPageHTML(`
<h1>Verificare în 2 pași</h1>
<p class="subtitle">Am trimis un cod de 6 cifre prin SMS la numărul ******000.</p>
<form method="post" class="form" id="mfaForm" action="/login?handler=VerifyMfa">
  ` + alert + `
  <div class="otp-inputs">` + digits.String() + `</div>
  <input type="hidden" id="otpCodeHidden" name="Input.Code" value="" />
  <label class="checkbox"><input type="checkbox" id="Input_RememberDevice" name="Input.RememberDevice" value="true" /> Ține minte acest dispozitiv 14 zile</label>
  <button type="submit" class="btn-primary" id="mfaSubmitBtn">Verifică</button>
<input name="__RequestVerificationToken" type="hidden" value="token-mfa" /><input name="Input.RememberDevice" type="hidden" value="false" /></form>
<form method="post" action="/login?handler=ResendOtp">
  <button type="submit" class="btn-link" id="resendBtn"><span id="resendBtnText">Retrimite codul</span></button>
<input name="__RequestVerificationToken" type="hidden" value="token-resend" /></form>
<form method="post" action="/login?handler=SendEmailOtp">
  <button type="submit" class="btn-link" id="emailOtpBtn" data-unlock-seconds="45">Primește codul pe email</button>
<input name="__RequestVerificationToken" type="hidden" value="token-email-otp" /></form>`)
}

const fixtureMaintenancePage = `<html><body><h1>Revenim imediat</h1><p>Lucrăm la platformă.</p></body></html>`

// ssoMock serves cont.solo.ro on localhost and falcon on 127.0.0.1 so the
// two hosts keep separate cookies like the real ones do
type ssoMock struct {
	t      *testing.T
	falcon *httptest.Server
	sso    *httptest.Server

	mu          sync.Mutex
	requests    []string
	legacyHits  int
	bridgeHits  int
	resendHits  int
	codeBodies  []string
	broken      bool
	bridgeURL   string
	noSolAuth   bool
	hadRemember bool
}

func newSSOMock(t *testing.T) (*ssoMock, *Client) {
	t.Helper()
	m := &ssoMock{t: t}

	falcon := http.NewServeMux()
	falcon.HandleFunc("/bridge", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.bridgeHits++
		m.mu.Unlock()
		if !m.noSolAuth {
			http.SetCookie(w, &http.Cookie{Name: "solo_auth", Value: "sso-token", Path: "/", HttpOnly: true})
		}
		http.SetCookie(w, &http.Cookie{Name: "solo_sso_bridged", Value: "1", Path: "/"})
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	})
	falcon.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><div id="root"></div></body></html>`)
	})
	falcon.HandleFunc("/api/security/login", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.legacyHits++
		m.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "solo_auth", Value: "legacy-token", Path: "/"})
		json.NewEncoder(w).Encode(map[string]string{"AuthenticationStatus": "OK"})
	})
	m.falcon = httptest.NewServer(falcon)
	t.Cleanup(m.falcon.Close)

	m.sso = httptest.NewServer(http.HandlerFunc(m.serveSSO))
	t.Cleanup(m.sso.Close)

	origBase, origSSO := baseURL, ssoBaseURL
	baseURL = m.falcon.URL
	ssoBaseURL = strings.Replace(m.sso.URL, "127.0.0.1", "localhost", 1)
	t.Cleanup(func() { baseURL, ssoBaseURL = origBase, origSSO })

	c, err := New("test-agent")
	if err != nil {
		t.Fatal(err)
	}
	return m, c
}

func (m *ssoMock) serveSSO(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	m.mu.Lock()
	defer m.mu.Unlock()
	handler := r.URL.Query().Get("handler")
	m.requests = append(m.requests, r.Method+" "+r.URL.RequestURI())
	if _, err := r.Cookie("__Host-solo_remember"); err == nil {
		m.hadRemember = true
	}

	if r.URL.Path != "/login" || m.broken {
		fmt.Fprint(w, fixtureMaintenancePage)
		return
	}
	if r.Method == http.MethodPost {
		if r.URL.Query().Get("returnTo") != "/dashboard" {
			m.t.Errorf("returnTo not carried over: %s", r.URL.RequestURI())
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || r.Header.Get("Origin") == "" || r.Header.Get("Referer") == "" {
			m.t.Errorf("form post headers missing: %v", r.Header)
		}
		if r.PostForm.Get("__RequestVerificationToken") == "" {
			m.t.Errorf("antiforgery token not posted to %s", r.URL.RequestURI())
		}
	}

	bridge := m.bridgeURL
	if bridge == "" {
		bridge = m.falcon.URL + "/bridge?redirect=%2Fdashboard"
	}
	switch {
	case r.Method == http.MethodGet:
		fmt.Fprint(w, fixtureEmailPage)
	case handler == "ActivateMfa":
		m.t.Error("ActivateMfa must never be posted")
	case handler == "SkipMfa":
		fmt.Fprint(w, fixtureSuccessPage(bridge))
	case handler == "ResendOtp":
		m.resendHits++
		fmt.Fprint(w, fixtureMFAPage(""))
	case handler == "VerifyMfa":
		m.codeBodies = append(m.codeBodies, r.PostForm.Encode())
		if r.PostForm.Get("Input.Code") != "123456" {
			fmt.Fprint(w, fixtureMFAPage("Codul nu este valid"))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "__Host-solo_remember", Value: "trusted", Path: "/", MaxAge: 14 * 24 * 3600})
		fmt.Fprint(w, fixtureSuccessPage(bridge))
	case r.PostForm.Get("Input.Password") == "":
		if r.PostForm.Get("Input.Email") != fixtureEmail {
			m.t.Errorf("email not posted: %v", r.PostForm)
		}
		fmt.Fprint(w, fixturePasswordPage(""))
	default:
		switch r.PostForm.Get("Input.Password") {
		case "secret":
			fmt.Fprint(w, fixtureRecommendationPage)
		case "direct":
			fmt.Fprint(w, fixtureSuccessPage(bridge))
		case "mfa":
			if m.hadRemember {
				fmt.Fprint(w, fixtureSuccessPage(bridge))
				return
			}
			fmt.Fprint(w, fixtureMFAPage(""))
		default:
			fmt.Fprint(w, fixturePasswordPage("Email sau parolă greșită"))
		}
	}
}

func (m *ssoMock) sessionValue(t *testing.T, c *Client) string {
	t.Helper()
	u, _ := url.Parse(baseURL)
	for _, ck := range c.httpClient.Jar.Cookies(u) {
		if ck.Name == "solo_auth" {
			return ck.Value
		}
	}
	return ""
}

func TestSSOLoginSkipsMFARecommendation(t *testing.T) {
	m, c := newSSOMock(t)

	if err := c.Login(fixtureEmail, "secret", nil); err != nil {
		t.Fatalf("Login: %v", err)
	}
	want := []string{
		"GET /login?returnTo=%2fdashboard",
		"POST /login?returnTo=%2fdashboard",
		"POST /login?returnTo=%2fdashboard",
		"POST /login?handler=SkipMfa&returnTo=%2fdashboard",
	}
	if strings.Join(m.requests, "\n") != strings.Join(want, "\n") {
		t.Errorf("requests:\n%s\nwant:\n%s", strings.Join(m.requests, "\n"), strings.Join(want, "\n"))
	}
	if m.bridgeHits != 1 || m.legacyHits != 0 {
		t.Errorf("bridge hits %d, legacy hits %d", m.bridgeHits, m.legacyHits)
	}
	if got := m.sessionValue(t, c); got != "sso-token" {
		t.Errorf("solo_auth = %q, want sso-token", got)
	}
	if c.LoginFallback != "" {
		t.Errorf("unexpected fallback: %s", c.LoginFallback)
	}
}

func TestSSOLoginDirectSuccess(t *testing.T) {
	m, c := newSSOMock(t)

	if err := c.Login(fixtureEmail, "direct", nil); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if len(m.requests) != 3 || m.bridgeHits != 1 || m.legacyHits != 0 {
		t.Errorf("requests %v, bridge %d, legacy %d", m.requests, m.bridgeHits, m.legacyHits)
	}
}

func TestSSOLoginWrongPassword(t *testing.T) {
	m, c := newSSOMock(t)

	err := c.Login(fixtureEmail, "wrong", nil)
	if !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("Login = %v, want ErrAuthenticationFailed", err)
	}
	if !strings.Contains(err.Error(), "Email sau parolă greșită") {
		t.Errorf("validation text missing: %v", err)
	}
	if m.legacyHits != 0 {
		t.Error("rejected credentials must not fall back to the legacy login")
	}
}

func TestSSOLoginMFAWithoutPrompt(t *testing.T) {
	m, c := newSSOMock(t)

	if err := c.Login(fixtureEmail, "mfa", nil); !errors.Is(err, ErrMFARequired) {
		t.Fatalf("Login = %v, want ErrMFARequired", err)
	}
	if m.legacyHits != 0 || m.bridgeHits != 0 {
		t.Errorf("2FA page must stop the flow: legacy %d, bridge %d", m.legacyHits, m.bridgeHits)
	}
}

func TestSSOLoginMFAWithPrompt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, c := newSSOMock(t)

	var prompts []MFAPrompt
	answers := []string{"", "000000", "123456"}
	prompt := func(p MFAPrompt) (string, error) {
		prompts = append(prompts, p)
		a := answers[0]
		answers = answers[1:]
		if a == "" {
			return "", ErrMFAResend
		}
		return a, nil
	}
	if err := c.Login(fixtureEmail, "mfa", prompt); err != nil {
		t.Fatalf("Login: %v", err)
	}

	if len(prompts) != 3 || m.resendHits != 1 {
		t.Fatalf("prompts %d, resends %d", len(prompts), m.resendHits)
	}
	if !prompts[0].CanResend || prompts[0].CodeLength != 6 || !strings.Contains(prompts[0].Hint, "******000") {
		t.Errorf("first prompt: %+v", prompts[0])
	}
	if prompts[2].Error != "Codul nu este valid" {
		t.Errorf("wrong code error not surfaced: %+v", prompts[2])
	}
	last := m.codeBodies[len(m.codeBodies)-1]
	if !strings.Contains(last, "Input.Code=123456") || !strings.Contains(last, "Input.RememberDevice=true") {
		t.Errorf("code post body: %s", last)
	}
	if m.legacyHits != 0 || m.sessionValue(t, c) != "sso-token" {
		t.Errorf("legacy %d, session %q", m.legacyHits, m.sessionValue(t, c))
	}
	if !c.HasRememberCookie() {
		t.Fatal("remember cookie not recorded")
	}

	// Both hosts persist with real metadata and the remember cookie spares
	// the code on the next login
	if err := c.SaveCookies(); err != nil {
		t.Fatal(err)
	}
	path, _ := getCookiePath()
	data, _ := os.ReadFile(path)
	var saved []SavedCookie
	json.Unmarshal(data, &saved)
	byName := map[string]SavedCookie{}
	for _, sc := range saved {
		byName[sc.Name] = sc
	}
	if sc := byName["solo_auth"]; sc.Domain != "127.0.0.1" || sc.Path != "/" || !sc.Expires.IsZero() || !sc.HttpOnly {
		t.Errorf("solo_auth saved as %+v", sc)
	}
	if sc := byName["__Host-solo_remember"]; sc.Domain != "localhost" || sc.Expires.Before(time.Now().Add(13*24*time.Hour)) {
		t.Errorf("remember cookie saved as %+v", sc)
	}

	c2, _ := New("test-agent")
	if loaded, err := c2.LoadCookies(); err != nil || !loaded {
		t.Fatalf("LoadCookies = %v, %v", loaded, err)
	}
	if err := c2.Login(fixtureEmail, "mfa", nil); err != nil {
		t.Fatalf("login with remember cookie: %v", err)
	}
	if !m.hadRemember {
		t.Error("remember cookie was not sent to cont.solo.ro")
	}
}

func TestSSOStructuralFailureFallsBackToLegacy(t *testing.T) {
	m, c := newSSOMock(t)
	m.broken = true

	if err := c.Login(fixtureEmail, "secret", nil); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if m.legacyHits != 1 || c.LoginFallback == "" {
		t.Errorf("legacy hits %d, fallback %q", m.legacyHits, c.LoginFallback)
	}
	if got := m.sessionValue(t, c); got != "legacy-token" {
		t.Errorf("solo_auth = %q", got)
	}
}

func TestSSOOffDomainBridgeRejected(t *testing.T) {
	m, c := newSSOMock(t)
	m.bridgeURL = "https://example.com/bridge?redirect=%2Fdashboard"

	if err := c.Login(fixtureEmail, "direct", nil); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if m.bridgeHits != 0 || m.legacyHits != 1 || !strings.Contains(c.LoginFallback, "bridge host") {
		t.Errorf("bridge %d, legacy %d, fallback %q", m.bridgeHits, m.legacyHits, c.LoginFallback)
	}
}

func TestSSOBridgeWithoutSessionFallsBack(t *testing.T) {
	m, c := newSSOMock(t)
	m.noSolAuth = true

	if err := c.Login(fixtureEmail, "direct", nil); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if m.legacyHits != 1 || !strings.Contains(c.LoginFallback, "solo_auth") {
		t.Errorf("legacy %d, fallback %q", m.legacyHits, c.LoginFallback)
	}
}

func TestAllowedBridgeHosts(t *testing.T) {
	orig := ssoBaseURL
	t.Cleanup(func() { ssoBaseURL = orig })
	ssoBaseURL = defaultSSOBaseURL

	for raw, want := range map[string]bool{
		"https://falcon.solo.ro/bridge": true,
		"https://solo.ro/bridge":        true,
		"http://falcon.solo.ro/bridge":  false,
		"https://solo.ro.evil.com/x":    false,
		"https://evilsolo.ro/x":         false,
		"http://localhost:8080/bridge":  false,
	} {
		u, _ := url.Parse(raw)
		if got := allowedBridge(u); got != want {
			t.Errorf("allowedBridge(%s) = %v, want %v", raw, got, want)
		}
	}
}

func TestHTMLFormParsing(t *testing.T) {
	doc, err := parseHTMLDoc(fixtureEmailPage)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.forms) != 1 || doc.forms[0].attrs["id"] != "emailForm" {
		t.Fatalf("markup inside scripts parsed as a form: %d forms", len(doc.forms))
	}

	page, _ := url.Parse("https://cont.solo.ro/login?returnTo=%2fdashboard")
	pw := classifySSOPage(fixturePasswordPage(""), page)
	if pw.kind != ssoPagePassword {
		t.Fatalf("password page classified as %v", pw.kind)
	}
	fields := pw.form.fields(map[string]string{"Input.Password": "p@ss w0rd"}, nil, nil)
	if got := formEncode(fields); got != "Input.Email=test%40exemplu.ro&Input.Password=p%40ss+w0rd&__RequestVerificationToken=token-password" {
		t.Errorf("password body = %s", got)
	}
	if got := formEncode([]formField{{"a b", "ă&=+/*-._~"}}); got != "a+b=%C4%83%26%3D%2B%2F*-._%7E" {
		t.Errorf("formEncode = %s", got)
	}

	rec := classifySSOPage(fixtureRecommendationPage, page)
	skipURL, _ := rec.form.actionURL(page, nil)
	if rec.kind != ssoPageMFARecommendation || skipURL.String() != "https://cont.solo.ro/login?handler=SkipMfa&returnTo=%2fdashboard" {
		t.Errorf("skip form: kind %v, url %v", rec.kind, skipURL)
	}

	mfa := classifySSOPage(fixtureMFAPage(""), page)
	if mfa.kind != ssoPageMFACode {
		t.Fatalf("MFA page classified as %v", mfa.kind)
	}
	if got := formEncode(mfa.challenge.fieldsFor("123456")); got != "Input.Code=123456&Input.RememberDevice=true&__RequestVerificationToken=token-mfa&Input.RememberDevice=false" {
		t.Errorf("code body = %s", got)
	}

	if classifySSOPage(fixtureMaintenancePage, page).kind != ssoPageUnknown {
		t.Error("maintenance page must be unknown")
	}
}

func TestCookieClearForLoginKeepsRemember(t *testing.T) {
	_, c := newSSOMock(t)
	c.jar.restore([]SavedCookie{
		{Name: "solo_auth", Value: "old", Domain: "127.0.0.1", Path: "/", HostOnly: true},
		{Name: "__Host-solo_remember", Value: "keep", Domain: "localhost", Path: "/", HostOnly: true, Expires: time.Now().Add(time.Hour)},
		{Name: "solo_sso_auth", Value: "drop", Domain: "localhost", Path: "/"},
	})
	if !c.hasSessionCookie() || !c.HasRememberCookie() {
		t.Fatal("restore failed")
	}
	c.clearForLogin()
	if c.hasSessionCookie() {
		t.Error("stale solo_auth survived")
	}
	snap := c.jar.snapshot()
	if len(snap) != 1 || snap[0].Name != "__Host-solo_remember" {
		t.Errorf("after clear: %+v", snap)
	}
	c.ForgetRememberCookie()
	if c.HasRememberCookie() {
		t.Error("ForgetRememberCookie kept the cookie")
	}
}

// cookies.json files written before the SSO login only carry name and value
func TestLoadLegacyCookieFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, c := newSSOMock(t)
	path, _ := getCookiePath()
	os.MkdirAll(filepath.Dir(path), 0755)
	legacy := `[{"name":"solo_auth","value":"tok","domain":"","path":"","expires":"0001-01-01T00:00:00Z"}]`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := c.LoadCookies()
	if err != nil || !loaded {
		t.Fatalf("LoadCookies = %v, %v", loaded, err)
	}
	u, _ := url.Parse(baseURL + "/proxy/accounting/dashboard/summary")
	if cks := c.httpClient.Jar.Cookies(u); len(cks) != 1 || cks[0].Value != "tok" {
		t.Errorf("legacy cookie not sent to falcon: %v", cks)
	}
}
