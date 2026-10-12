package client

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

// baseURL is a var so tests can point the client at a mock server. The
// SOLO_API_BASE env var overrides it so the e2e suite can run the real
// binary against a local mock API
var baseURL = "https://falcon.solo.ro"

func init() {
	if v := os.Getenv("SOLO_API_BASE"); v != "" {
		baseURL = v
	}
	if v := os.Getenv("SOLO_SSO_BASE"); v != "" {
		ssoBaseURL = v
	}
}

const loginPath = "/api/security/login"

// Client wraps an HTTP client with cookie storage for SOLO.ro API
type Client struct {
	httpClient *http.Client
	jar        *recordingJar
	userAgent  string
	CompanyID  string

	// LoginFallback is set when the last Login fell back to the legacy
	// JSON endpoint and says why the cont.solo.ro flow failed
	LoginFallback string
}

// loginRequest represents the login request body
type loginRequest struct {
	UserName string `json:"UserName"`
	Password string `json:"Password"`
}

// loginResponse represents the login response body
type loginResponse struct {
	AuthenticationStatus string `json:"AuthenticationStatus"`
}

// ErrAuthenticationFailed is returned when login credentials are invalid
var ErrAuthenticationFailed = errors.New("authentication failed: invalid credentials")

// New creates a new Client with cookie jar and user agent
func New(userAgent string) (*Client, error) {
	jar := newRecordingJar()
	return &Client{
		jar: jar,
		httpClient: &http.Client{
			Jar: jar,
			// Bound every API call, a hung connection must not freeze
			// the CLI or TUI forever
			Timeout: 30 * time.Second,
		},
		userAgent: userAgent,
	}, nil
}

// Login signs in through cont.solo.ro and bridges the session to falcon.
// prompt is asked for the 2FA code when the account needs one, with a nil
// prompt reaching the code page returns ErrMFARequired. The legacy JSON
// login is used only when the SSO pages stop matching the known flow, never
// after rejected credentials or a 2FA page
func (c *Client) Login(username, password string, prompt MFAPromptFunc) error {
	c.LoginFallback = ""
	c.clearForLogin()
	err := c.ssoLogin(username, password, prompt)
	var se *ssoStructuralError
	if !errors.As(err, &se) {
		return err
	}
	c.LoginFallback = se.reason
	c.clearForLogin()
	return c.legacyLogin(username, password)
}

func (c *Client) legacyLogin(username, password string) error {
	var resp loginResponse
	err := c.doJSON("POST", loginPath, "/authentication", loginRequest{UserName: username, Password: password}, &resp)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if resp.AuthenticationStatus != "OK" {
		return ErrAuthenticationFailed
	}

	return nil
}

// Summary represents the dashboard summary response
type Summary struct {
	Year                    int     `json:"Year"`
	DisplayCurrency         string  `json:"DisplayCurrency"`
	TotalRevenues           float64 `json:"TotalRevenues"`
	TotalDeductibleExpenses float64 `json:"TotalDeductibleExpenses"`
	HasTaxes                bool    `json:"HasTaxes"`
	Taxes                   float64 `json:"Taxes"`
	RevenuesAwaitingReview  int     `json:"RevenuesAwaitingReview"`
	ExpensesAwaitingReview  int     `json:"ExpensesAwaitingReview"`
}

// GetSummary fetches the dashboard summary for the current year
func (c *Client) GetSummary() (*Summary, error) {
	return c.GetSummaryForYear(0)
}

// GetSummaryForYear fetches the dashboard summary for a specific year (0 = current)
func (c *Client) GetSummaryForYear(year int) (*Summary, error) {
	path := "/proxy/accounting/dashboard/summary"
	if year > 0 {
		path = fmt.Sprintf("%s?year=%d", path, year)
	}

	var summary Summary
	if err := c.doJSON("GET", path, "/dashboard", nil, &summary); err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}
	return &summary, nil
}
