package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ssoBaseURL is the cont.solo.ro Razor Pages login. Like baseURL it is a var
// so tests can point it at a mock, SOLO_SSO_BASE overrides it for the e2e suite
var ssoBaseURL = defaultSSOBaseURL

const (
	defaultSSOBaseURL = "https://cont.solo.ro"
	ssoLoginPath      = "/login?returnTo=%2fdashboard"
	ssoMaxSteps       = 8
	ssoMaxMFARounds   = 8
	ssoMaxPageBytes   = 4 << 20
)

// ErrMFARequired is returned when the login reached the 2FA code page and
// no code prompt was supplied, reaching that page already texted a code
var ErrMFARequired = errors.New("this account uses two-factor authentication and a code is required")

// ErrMFAUnsupported is returned when the login reached a 2FA code page it
// cannot fill in. The code may already be on its way, so there is no fallback
var ErrMFAUnsupported = errors.New("the two-factor code page changed and solo-cli cannot fill it in, sign in on solo.ro or update solo-cli")

// ErrMFAResend is returned by an MFAPromptFunc to ask for a new code
var ErrMFAResend = errors.New("resend the two-factor code")

// MFAPrompt describes the 2FA code page for the prompt callback
type MFAPrompt struct {
	Hint       string
	Error      string
	CodeLength int
	CanResend  bool
}

// MFAPromptFunc asks the user for the 2FA code. Returning ErrMFAResend posts
// the page's resend form instead, any other error aborts the login
type MFAPromptFunc func(MFAPrompt) (string, error)

// ssoStructuralError means the pages stopped looking like the known flow,
// the only case where Login may fall back to the legacy JSON endpoint
type ssoStructuralError struct{ reason string }

func (e *ssoStructuralError) Error() string { return "cont.solo.ro login: " + e.reason }

// structural builds a fallback reason, cleaned because it reaches stderr
func structural(format string, args ...any) error {
	return &ssoStructuralError{reason: cleanString(fmt.Sprintf(format, args...))}
}

type ssoPageKind int

const (
	ssoPageUnknown ssoPageKind = iota
	ssoPageEmail
	ssoPagePassword
	ssoPageMFARecommendation
	ssoPageMFACode
	ssoPageMFAUnreadable
	ssoPageSuccess
)

type ssoPage struct {
	kind      ssoPageKind
	form      *htmlForm
	message   string
	challenge *mfaChallenge
	redirect  *url.URL
}

// mfaChallenge is the parsed code entry page. Field names come from the HTML
// because the page was modeled on login.js and never observed live
type mfaChallenge struct {
	form       *htmlForm
	submitter  *htmlControl
	pageURL    *url.URL
	codeNames  []string
	digitNames []string
	remember   map[string]bool
	codeLength int
	hint       string
	errMessage string
	resend     *mfaAction
}

type mfaAction struct {
	form      *htmlForm
	submitter *htmlControl
}

func (ch *mfaChallenge) fieldsFor(code string) []formField {
	values := map[string]string{}
	for _, name := range ch.codeNames {
		values[name] = code
	}
	for i, name := range ch.digitNames {
		if i < len(code) {
			values[name] = code[i : i+1]
		} else {
			values[name] = ""
		}
	}
	return ch.form.fields(values, ch.remember, ch.submitter)
}

func (ch *mfaChallenge) prompt() MFAPrompt {
	return MFAPrompt{
		Hint:       cleanString(ch.hint),
		Error:      cleanString(ch.errMessage),
		CodeLength: ch.codeLength,
		CanResend:  ch.resend != nil,
	}
}

func classifySSOPage(src string, pageURL *url.URL) ssoPage {
	doc, err := parseHTMLDoc(src)
	if err != nil {
		return ssoPage{}
	}

	attrs, _, ok := doc.firstElement(func(_ string, a map[string]string) bool { return a["id"] == "loginSuccess" })
	if !ok {
		attrs, _, ok = doc.firstElement(func(_ string, a map[string]string) bool { _, has := a["data-redirect-url"]; return has })
	}
	if ok {
		if raw, has := attrs["data-redirect-url"]; has {
			if u, err := pageURL.Parse(raw); err == nil {
				return ssoPage{kind: ssoPageSuccess, redirect: u}
			}
		}
	}

	mfaForm := doc.formByID("mfaForm")
	if mfaForm == nil {
		for _, f := range doc.forms {
			for _, c := range f.controls {
				if c.attrs["id"] == "otpCodeHidden" || hasClass(c.attrs, "otp-digit") {
					mfaForm = f
					break
				}
			}
			if mfaForm != nil {
				break
			}
		}
	}
	if mfaForm != nil {
		ch := makeMFAChallenge(mfaForm, doc, pageURL)
		if ch == nil {
			return ssoPage{kind: ssoPageMFAUnreadable}
		}
		return ssoPage{kind: ssoPageMFACode, challenge: ch}
	}

	if doc.formWithAction("handler=ActivateMfa") != nil || doc.formWithAction("handler=SkipMfa") != nil {
		return ssoPage{kind: ssoPageMFARecommendation, form: doc.formWithAction("handler=SkipMfa")}
	}

	if f := doc.formByID("passwordForm"); f != nil {
		return ssoPage{kind: ssoPagePassword, form: f, message: ssoErrorMessage(doc)}
	}
	for _, f := range doc.forms {
		if f.passwordControl() != nil {
			return ssoPage{kind: ssoPagePassword, form: f, message: ssoErrorMessage(doc)}
		}
	}

	// A code page whose ids changed still has a field named like a code,
	// checked before the email page because it may carry a hidden email
	for _, f := range doc.forms {
		if len(codeFieldNames(f)) > 0 && !hasEmailInput(f) {
			if ch := makeMFAChallenge(f, doc, pageURL); ch != nil {
				return ssoPage{kind: ssoPageMFACode, challenge: ch}
			}
		}
	}

	if f := doc.formByID("emailForm"); f != nil {
		return ssoPage{kind: ssoPageEmail, form: f, message: ssoErrorMessage(doc)}
	}
	for _, f := range doc.forms {
		if f.emailControl() != nil {
			return ssoPage{kind: ssoPageEmail, form: f, message: ssoErrorMessage(doc)}
		}
	}
	return ssoPage{}
}

// ssoErrorMessage is the server-side validation text, if the page shows any
func ssoErrorMessage(doc *htmlDoc) string {
	if t := doc.firstText(func(a map[string]string) bool { return a["id"] == "otpErrorAlert" }); t != "" {
		return t
	}
	if t := doc.firstText(func(a map[string]string) bool { return hasClass(a, "field-validation-error") }); t != "" {
		return t
	}
	if t := doc.firstText(func(a map[string]string) bool { return hasClass(a, "validation-summary-errors") }); t != "" {
		return t
	}
	return doc.firstText(func(a map[string]string) bool {
		return !hasClass(a, "field-validation-valid") &&
			(hasClass(a, "alert-danger") || hasClass(a, "alert-error") || hasClass(a, "text-danger"))
	})
}

func makeMFAChallenge(form *htmlForm, doc *htmlDoc, pageURL *url.URL) *mfaChallenge {
	ch := &mfaChallenge{form: form, pageURL: pageURL, remember: map[string]bool{}}

	var digits []*htmlControl
	for _, c := range form.controls {
		if hasClass(c.attrs, "otp-digit") {
			digits = append(digits, c)
		}
	}
	if hidden := form.controlByID("otpCodeHidden"); hidden != nil && hidden.name() != "" {
		ch.codeNames = []string{hidden.name()}
	} else {
		ch.codeNames = codeFieldNames(form)
	}
	for _, d := range digits {
		if d.name() != "" {
			ch.digitNames = append(ch.digitNames, d.name())
		}
	}
	if len(ch.codeNames) == 0 && (len(digits) == 0 || len(ch.digitNames) != len(digits)) {
		return nil
	}

	var boxes []*htmlControl
	for _, c := range form.controls {
		if c.typ() == "checkbox" {
			boxes = append(boxes, c)
		}
	}
	for _, box := range boxes {
		key := strings.ToLower(box.name() + " " + box.attrs["id"])
		matches := len(boxes) == 1
		for _, word := range []string{"remember", "trust", "device", "minte"} {
			if strings.Contains(key, word) {
				matches = true
			}
		}
		if matches && box.name() != "" {
			ch.remember[box.name()] = true
		}
	}

	ch.codeLength = 6
	if len(digits) > 0 {
		ch.codeLength = len(digits)
	} else {
		for _, name := range ch.codeNames {
			for _, c := range form.controls {
				if c.name() != name {
					continue
				}
				if n, err := strconv.Atoi(c.attrs["maxlength"]); err == nil && n >= 4 && n <= 8 {
					ch.codeLength = n
				}
			}
		}
	}

	ch.submitter = form.controlByID("mfaSubmitBtn")
	if ch.submitter == nil {
		for _, c := range form.controls {
			if c.isSubmitButton() && mfaActionKind(c) == "" {
				ch.submitter = c
				break
			}
		}
	}

	ch.hint = doc.firstText(func(a map[string]string) bool { return hasClass(a, "subtitle") })
	ch.errMessage = ssoErrorMessage(doc)

	for _, f := range doc.forms {
		for _, c := range f.controls {
			if c.isSubmitButton() && mfaActionKind(c) == "resend" && ch.resend == nil {
				ch.resend = &mfaAction{form: f, submitter: c}
			}
		}
	}
	return ch
}

// codeFieldNames lists the inputs named like a 2FA code, digit boxes excluded
func codeFieldNames(form *htmlForm) []string {
	var names []string
	for _, c := range form.controls {
		switch c.typ() {
		case "hidden", "text", "number", "tel", "password":
		default:
			continue
		}
		lower := strings.ToLower(c.name())
		if !hasClass(c.attrs, "otp-digit") && lower != "" && (strings.Contains(lower, "code") || strings.Contains(lower, "otp")) {
			names = append(names, c.name())
		}
	}
	return names
}

func hasEmailInput(form *htmlForm) bool {
	for _, c := range form.controls {
		if c.typ() == "email" {
			return true
		}
	}
	return false
}

// mfaActionKind tells the resend and email fallback buttons apart from the
// verify button
func mfaActionKind(c *htmlControl) string {
	key := strings.ToLower(c.attrs["id"] + " " + c.name() + " " + c.attrs["formaction"])
	switch {
	case strings.Contains(key, "email"):
		return "email"
	case strings.Contains(key, "resend"):
		return "resend"
	}
	return ""
}

type ssoState struct {
	email, password       string
	sentEmail, sentPasswd bool
	skippedRecommendation bool
}

// ssoLogin drives email, password, the 2FA recommendation (always skipped,
// never activated) and the code page, then opens the falcon bridge. Errors
// of type *ssoStructuralError before the code page allow the legacy fallback,
// network failures and unexpected form targets never do
func (c *Client) ssoLogin(email, password string, prompt MFAPromptFunc) error {
	start, err := url.Parse(ssoBaseURL + ssoLoginPath)
	if err != nil {
		return structural("bad SSO base URL: %v", err)
	}
	req, err := c.ssoRequest(http.MethodGet, start, nil, nil)
	if err != nil {
		return structural("%v", err)
	}
	src, pageURL, err := c.ssoLoad(req)
	if err != nil {
		return err
	}
	st := &ssoState{email: email, password: password}
	ch, err := c.ssoAdvance(src, pageURL, st)
	if err != nil || ch == nil {
		return err
	}

	// From here on the account has 2FA, never fall back to the legacy login
	if prompt == nil {
		return ErrMFARequired
	}
	for round := 0; round < ssoMaxMFARounds; round++ {
		code, perr := prompt(ch.prompt())
		var req *http.Request
		switch {
		case errors.Is(perr, ErrMFAResend):
			if ch.resend == nil {
				ch.errMessage = "this page does not offer a resend"
				continue
			}
			req, err = c.ssoPostForm(ch.resend.form, ch.resend.form.fields(nil, nil, ch.resend.submitter), ch.pageURL, ch.resend.submitter)
		case perr != nil:
			return perr
		default:
			req, err = c.ssoPostForm(ch.form, ch.fieldsFor(code), ch.pageURL, ch.submitter)
		}
		if err == nil {
			var nextSrc string
			var nextURL *url.URL
			if nextSrc, nextURL, err = c.ssoLoad(req); err == nil {
				ch, err = c.ssoAdvance(nextSrc, nextURL, &ssoState{sentEmail: true, sentPasswd: true})
			}
		}
		if err != nil {
			var se *ssoStructuralError
			if errors.As(err, &se) {
				return fmt.Errorf("two-factor login failed: %s", se.reason)
			}
			return err
		}
		if ch == nil {
			return nil
		}
	}
	return fmt.Errorf("two-factor login failed: too many attempts")
}

// ssoAdvance walks the pages until success, a code page (returned) or an error
func (c *Client) ssoAdvance(src string, pageURL *url.URL, st *ssoState) (*mfaChallenge, error) {
	for step := 0; step < ssoMaxSteps; step++ {
		page := classifySSOPage(src, pageURL)
		var req *http.Request
		var err error
		switch page.kind {
		case ssoPageEmail:
			if st.sentEmail {
				return nil, credentialsError(page.message)
			}
			ctl := page.form.emailControl()
			if ctl == nil {
				return nil, structural("email form without an email input")
			}
			st.sentEmail = true
			req, err = c.ssoPostForm(page.form, page.form.fields(map[string]string{ctl.name(): st.email}, nil, nil), pageURL, nil)
		case ssoPagePassword:
			if st.sentPasswd {
				return nil, credentialsError(page.message)
			}
			ctl := page.form.passwordControl()
			if ctl == nil {
				return nil, structural("password form without a password input")
			}
			values := map[string]string{ctl.name(): st.password}
			if ec := page.form.emailControl(); ec != nil && ec.name() != "" && ec.value() == "" {
				values[ec.name()] = st.email
			}
			st.sentEmail, st.sentPasswd = true, true
			req, err = c.ssoPostForm(page.form, page.form.fields(values, nil, nil), pageURL, nil)
		case ssoPageMFARecommendation:
			if page.form == nil || st.skippedRecommendation {
				return nil, structural("cannot skip the 2FA recommendation")
			}
			st.skippedRecommendation = true
			req, err = c.ssoPostForm(page.form, page.form.fields(nil, nil, nil), pageURL, nil)
		case ssoPageMFACode:
			return page.challenge, nil
		case ssoPageMFAUnreadable:
			return nil, ErrMFAUnsupported
		case ssoPageSuccess:
			return nil, c.ssoOpenBridge(page.redirect, pageURL)
		default:
			return nil, structural("unrecognised login page at %s", pageURL.EscapedPath())
		}
		if err != nil {
			return nil, err
		}
		if src, pageURL, err = c.ssoLoad(req); err != nil {
			return nil, err
		}
	}
	return nil, structural("too many login steps")
}

func credentialsError(message string) error {
	if message = cleanString(message); message != "" {
		return fmt.Errorf("%w: %s", ErrAuthenticationFailed, message)
	}
	return ErrAuthenticationFailed
}

func (c *Client) ssoOpenBridge(bridge, referer *url.URL) error {
	if !allowedBridge(bridge) {
		return structural("unexpected bridge host %q", bridge.Host)
	}
	req, err := c.ssoRequest(http.MethodGet, bridge, nil, referer)
	if err != nil {
		return structural("%v", err)
	}
	if _, _, err := c.ssoLoad(req); err != nil {
		return err
	}
	if !c.hasSessionCookie() {
		return structural("bridge did not set solo_auth")
	}
	return nil
}

// allowedBridge accepts https solo.ro hosts only, plus the mock hosts when
// the SSO base is overridden for tests
func allowedBridge(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	if u.Scheme == "https" && (host == "solo.ro" || strings.HasSuffix(host, ".solo.ro")) {
		return true
	}
	if ssoBaseURL == defaultSSOBaseURL {
		return false
	}
	for _, raw := range []string{baseURL, ssoBaseURL} {
		if base, err := url.Parse(raw); err == nil && base.Scheme == u.Scheme && base.Host == u.Host {
			return true
		}
	}
	return false
}

func (c *Client) hasSessionCookie() bool {
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	for _, ck := range c.httpClient.Jar.Cookies(u) {
		if ck.Name == authCookieName && ck.Value != "" {
			return true
		}
	}
	return false
}

func (c *Client) ssoRequest(method string, u *url.URL, body io.Reader, referer *url.URL) (*http.Request, error) {
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ro-RO,ro;q=0.9,en;q=0.8")
	if referer != nil {
		req.Header.Set("Referer", referer.String())
	}
	return req, nil
}

// ssoPostForm submits a form like a browser, as a GET query when its method
// is get. Credentials only ever go to the cont.solo.ro origin, so a page or
// form target anywhere else is a hard error rather than a fallback
func (c *Client) ssoPostForm(form *htmlForm, fields []formField, pageURL *url.URL, submitter *htmlControl) (*http.Request, error) {
	target, err := form.actionURL(pageURL, submitter)
	if err != nil {
		return nil, structural("%v", err)
	}
	for _, u := range []*url.URL{pageURL, target} {
		if !onSSOOrigin(u) {
			return nil, fmt.Errorf("cont.solo.ro login: refusing to send the sign in form to %s", cleanString(u.Scheme+"://"+u.Host))
		}
	}
	body := formEncode(fields)
	if form.method() == "get" {
		target.RawQuery = body
		req, err := c.ssoRequest(http.MethodGet, target, nil, pageURL)
		if err != nil {
			return nil, structural("%v", err)
		}
		return req, nil
	}
	req, err := c.ssoRequest(http.MethodPost, target, strings.NewReader(body), pageURL)
	if err != nil {
		return nil, structural("%v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", pageURL.Scheme+"://"+pageURL.Host)
	return req, nil
}

// onSSOOrigin reports whether u is on the scheme and host of ssoBaseURL,
// https://cont.solo.ro outside tests
func onSSOOrigin(u *url.URL) bool {
	base, err := url.Parse(ssoBaseURL)
	return err == nil && u != nil && strings.EqualFold(u.Scheme, base.Scheme) && strings.EqualFold(u.Host, base.Host)
}

// ssoLoad performs one step and returns the page HTML and its final URL.
// An HTTP error status is structural, a network failure is not: like the
// iOS app, being offline must not send the password to the legacy login
func (c *Client) ssoLoad(req *http.Request) (string, *url.URL, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("cont.solo.ro login: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, ssoMaxPageBytes))
	if err != nil {
		return "", nil, fmt.Errorf("cont.solo.ro login: reading %s: %w", req.URL.EscapedPath(), err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", nil, structural("HTTP %d for %s", resp.StatusCode, req.URL.EscapedPath())
	}
	return string(data), resp.Request.URL, nil
}
