package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
)

// baseURL is a var so tests can point the client at a mock server
var baseURL = "https://falcon.solo.ro"

const loginPath = "/api/security/login"

// Client wraps an HTTP client with cookie storage for SOLO.ro API
type Client struct {
	httpClient *http.Client
	userAgent  string
	CompanyID  string
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
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{
			Jar: jar,
		},
		userAgent: userAgent,
	}, nil
}

// Login authenticates with SOLO.ro and stores session cookies
func (c *Client) Login(username, password string) error {
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
