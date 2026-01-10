package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
)

const (
	baseURL   = "https://falcon.solo.ro"
	loginPath = "/api/security/login"
)

// Client wraps an HTTP client with cookie storage for SOLO.ro API
type Client struct {
	httpClient *http.Client
	userAgent  string
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
	reqBody := loginRequest{
		UserName: username,
		Password: password,
	}

	bodyData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", baseURL+loginPath, bytes.NewReader(bodyData))
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/authentication")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp loginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return err
	}

	if loginResp.AuthenticationStatus != "OK" {
		return ErrAuthenticationFailed
	}

	return nil
}

// GetHTTPClient returns the underlying HTTP client (useful for subsequent API calls)
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
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
	url := baseURL + "/proxy/accounting/dashboard/summary"
	if year > 0 {
		url = fmt.Sprintf("%s?year=%d", url, year)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", baseURL+"/dashboard")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get summary: status %d", resp.StatusCode)
	}

	var summary Summary
	if err := json.Unmarshal(body, &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}
