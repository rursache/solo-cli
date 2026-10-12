package client

import (
	"fmt"
	"net/url"
	"strings"
)

// ReferralCampaign is the user's referral code and the running campaign
type ReferralCampaign struct {
	ReferralCode    string        `json:"ReferralCode"`
	CurrentCampaign *CampaignInfo `json:"CurrentCampaign"`
}

// Link is the shareable referral URL, empty without a code
func (r *ReferralCampaign) Link() string {
	code := strings.TrimSpace(r.ReferralCode)
	if code == "" {
		return ""
	}
	return "https://www.solo.ro/ref/" + url.PathEscape(code)
}

// CampaignInfo holds the prizes in lei. GuestPrizeValue 0 means only the
// referrer is rewarded
type CampaignInfo struct {
	Code               string  `json:"Code"`
	ReferrerPrizeValue float64 `json:"ReferrerPrizeValue"`
	GuestPrizeValue    float64 `json:"GuestPrizeValue"`
	IsTimeLimited      bool    `json:"IsTimeLimited"`
	StartDate          *string `json:"StartDate"`
	EndDate            *string `json:"EndDate"`
}

// ReferralRewards is the lei earned so far and the lei still pending
type ReferralRewards struct {
	PendingWinnings float64 `json:"PendingWinnings"`
	EarnedWinnings  float64 `json:"EarnedWinnings"`
}

// GetReferralCampaign fetches the referral code and campaign
func (c *Client) GetReferralCampaign() (*ReferralCampaign, error) {
	var result ReferralCampaign
	if err := c.doJSON("GET", "/api/referral/campaign", "/referral", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get referral campaign: %w", err)
	}
	return &result, nil
}

// GetReferralRewards fetches the referral earnings
func (c *Client) GetReferralRewards() (*ReferralRewards, error) {
	var result ReferralRewards
	if err := c.doJSON("GET", "/api/referral/rewards", "/referral", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get referral rewards: %w", err)
	}
	return &result, nil
}
