package client

import "fmt"

// ViesProfile is the intra community VAT (VIES) state of the company. Only
// VatRegistered and None were seen live, so both statuses stay raw strings
type ViesProfile struct {
	Code3              *string `json:"Code3"`
	ViesProfileStatus  string  `json:"ViesProfileStatus"`
	ViesWorkflowStatus string  `json:"ViesWorkflowStatus"`
}

// GetViesProfile fetches the VIES profile. Ok=false is the web's "profile
// unavailable" state and comes back as *APIError
func (c *Client) GetViesProfile(companyID string) (*ViesProfile, error) {
	if companyID == "" {
		return nil, fmt.Errorf("company ID not available")
	}
	var profile *ViesProfile
	if err := c.doOk("GET", "/proxy/accounting/company/vies-profile/company_"+companyID, "/settings", nil, &profile); err != nil {
		return nil, fmt.Errorf("failed to get VIES profile: %w", err)
	}
	if profile == nil {
		return nil, fmt.Errorf("VIES profile response has no data")
	}
	return profile, nil
}
