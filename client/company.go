package client

import "fmt"

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

// CAENCode represents one CAEN activity code assigned to the company
type CAENCode struct {
	Id        int    `json:"Id"`
	IsPrimary bool   `json:"IsPrimary"`
	Code      string `json:"Code"`
	Name      string `json:"Name"`
	Display   string `json:"Display"`
}

// GetCAENCodes fetches the company's CAEN activity codes
func (c *Client) GetCAENCodes(companyID string) ([]CAENCode, error) {
	if companyID == "" {
		return nil, fmt.Errorf("company ID not available")
	}

	path := "/proxy/accounting/company/caen-codes/company_" + companyID

	var result []CAENCode
	if err := c.doJSON("GET", path, "/settings", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get CAEN codes: %w", err)
	}
	return result, nil
}

// GetCompanyInfo fetches company profile by ID
func (c *Client) GetCompanyInfo(companyID string) (*CompanyInfo, error) {
	if companyID == "" {
		return nil, fmt.Errorf("company ID not available")
	}

	path := fmt.Sprintf("/proxy/accounting/company/basic-profile/company_%s", companyID)

	var result CompanyInfoResponse
	if err := c.doJSON("GET", path, "/settings", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get company info: %w", err)
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
