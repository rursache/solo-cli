package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CompanyInfo represents the company profile
type CompanyInfo struct {
	Name            string `json:"Name"`
	Code1           string `json:"Code1"` // CUI/CIF
	Code2           string `json:"Code2"` // Registration number
	Address         string `json:"Address"`
	InvoiceMentions string `json:"InvoiceMentions"`
}

// CompanyInfoResponse represents the API response
type CompanyInfoResponse struct {
	Data         *CompanyInfo `json:"Data"`
	Ok           bool         `json:"Ok"`
	ErrorCode    *string      `json:"ErrorCode"`
	ErrorMessage *string      `json:"ErrorMessage"`
}

// GetCompanyInfo fetches company profile by ID
func (c *Client) GetCompanyInfo(companyID string) (*CompanyInfo, error) {
	if companyID == "" {
		return nil, fmt.Errorf("company_id not configured")
	}

	url := fmt.Sprintf("%s/proxy/accounting/company/basic-profile/company_%s", baseURL, companyID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", baseURL+"/settings")

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
		return nil, fmt.Errorf("failed to get company info: status %d", resp.StatusCode)
	}

	var result CompanyInfoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.Ok {
		errMsg := "unknown error"
		if result.ErrorMessage != nil {
			errMsg = *result.ErrorMessage
		}
		return nil, fmt.Errorf("failed to get company info: %s", errMsg)
	}

	return result.Data, nil
}
