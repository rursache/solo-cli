package client

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
)

var companyCodeRe = regexp.MustCompile(`CompanyCode:\s*"company_([0-9a-fA-F]{32})"`)

// DiscoverCompanyID fetches an authenticated HTML page and extracts the company ID
// from the server-injected Angular Principal constant.
func (c *Client) DiscoverCompanyID() (string, error) {
	req, err := http.NewRequest("GET", baseURL+"/dashboard", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch dashboard: status %d", resp.StatusCode)
	}

	matches := companyCodeRe.FindSubmatch(body)
	if matches == nil {
		return "", fmt.Errorf("company ID not found in dashboard response")
	}

	return string(matches[1]), nil
}
