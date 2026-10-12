package client

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

var (
	companyCodeRe      = regexp.MustCompile(`CompanyCode:\s*"company_([0-9a-fA-F]{32})"`)
	companyStartYearRe = regexp.MustCompile(`CompanyStartYear:\s*"?(\d{4})`)
)

// DiscoverCompanyID fetches an authenticated HTML page and extracts the company ID
// from the server-injected Angular Principal constant. It also records the
// company start year in CompanyStartYear when the page has one
func (c *Client) DiscoverCompanyID() (string, error) {
	id, start, err := c.discoverCompany()
	if err != nil {
		return "", err
	}
	c.CompanyStartYear = start
	return id, nil
}

func (c *Client) discoverCompany() (string, int, error) {
	req, err := http.NewRequest("GET", baseURL+"/dashboard", nil)
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("failed to fetch dashboard: status %d", resp.StatusCode)
	}

	matches := companyCodeRe.FindSubmatch(body)
	if matches == nil {
		return "", 0, fmt.Errorf("company ID not found in dashboard response")
	}

	return string(matches[1]), parseStartYear(body, time.Now().Year()), nil
}

// parseStartYear returns the CompanyStartYear constant, 0 when missing or
// outside 1990..currentYear
func parseStartYear(body []byte, currentYear int) int {
	m := companyStartYearRe.FindSubmatch(body)
	if m == nil {
		return 0
	}
	year, err := strconv.Atoi(string(m[1]))
	if err != nil || year < 1990 || year > currentYear {
		return 0
	}
	return year
}
