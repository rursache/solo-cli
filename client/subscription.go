package client

import (
	"fmt"
	"strings"
	"time"
)

// SubscriptionResponse is the Setari > Abonament payload. RestartOptions and
// the product list are left out since the CLI never changes the subscription
type SubscriptionResponse struct {
	SubscriptionSummary *SubscriptionSummary `json:"SubscriptionSummary"`
	ChangeOptions       *SubscriptionOptions `json:"ChangeOptions"`
	ScheduledChange     *ScheduledChange     `json:"ScheduledChange"`
}

// SubscriptionSummary holds the plan, the overdue debt and the default card
type SubscriptionSummary struct {
	Subscription          *Subscription   `json:"Subscription"`
	OverduePaymentSummary *OverduePayment `json:"OverduePaymentSummary"`
	DefaultCard           *PaymentCard    `json:"DefaultCard"`
}

// Subscription is the current plan
type Subscription struct {
	Name                   string  `json:"Name"`
	Status                 string  `json:"Status"`
	IsCollectionPaused     bool    `json:"IsCollectionPaused"`
	NextBillingDate        string  `json:"NextBillingDate"`
	ProductPrice           float64 `json:"ProductPrice"`
	HasUpcomingPriceChange bool    `json:"HasUpcomingPriceChange"`
	UpcomingProductPrice   float64 `json:"UpcomingProductPrice"`
	UpcomingProductName    *string `json:"UpcomingProductName"`
	UpcomingChangeOn       *string `json:"UpcomingChangeOn"`
	UpcomingInvoiceTotal   float64 `json:"UpcomingInvoiceTotal"`
	DiscountedAmount       float64 `json:"DiscountedAmount"`
	CreatedOn              string  `json:"CreatedOn"`
	CancelledOn            *string `json:"CancelledOn"`
}

// DisplayStatus reads Status like the web, an Overdue subscription with
// paused collection is shown as paused
func (s *Subscription) DisplayStatus() string {
	switch strings.ToLower(strings.TrimSpace(s.Status)) {
	case "active":
		return "Active"
	case "overdue":
		if s.IsCollectionPaused {
			return "Paused"
		}
		return "Overdue"
	case "cancelled", "canceled":
		return "Cancelled"
	case "":
		return "Unknown"
	}
	return s.Status
}

// OverduePayment is the unpaid subscription debt
type OverduePayment struct {
	Total        float64 `json:"Total"`
	PaymentCount int     `json:"PaymentCount"`
}

// PaymentCard is the default card on file
type PaymentCard struct {
	ExpiringMonth int    `json:"ExpiringMonth"`
	ExpiringYear  int    `json:"ExpiringYear"`
	IsExpired     bool   `json:"IsExpired"`
	LastDigits    string `json:"LastDigits"`
	CardBrand     string `json:"CardBrand"`
}

// BrandLabel names the card brand like the web
func (p *PaymentCard) BrandLabel() string {
	switch strings.ToLower(p.CardBrand) {
	case "visa":
		return "Visa"
	case "mastercard":
		return "Mastercard"
	}
	return "Card"
}

// Expiry is "MM/YYYY", empty when the server sent no valid expiry
func (p *PaymentCard) Expiry() string {
	if p.ExpiringMonth < 1 || p.ExpiringMonth > 12 || p.ExpiringYear == 0 {
		return ""
	}
	return fmt.Sprintf("%02d/%d", p.ExpiringMonth, p.ExpiringYear)
}

// Expired trusts the server flag or a past expiry month in case it lags
func (p *PaymentCard) Expired(now time.Time) bool {
	if p.IsExpired {
		return true
	}
	if p.Expiry() == "" {
		return false
	}
	return p.ExpiringYear < now.Year() || (p.ExpiringYear == now.Year() && p.ExpiringMonth < int(now.Month()))
}

// SubscriptionOptions is ChangeOptions, read only for the current plan
type SubscriptionOptions struct {
	CurrentSubscription *SubscriptionPlan `json:"CurrentSubscription"`
}

// SubscriptionPlan is the current product with its billing period
type SubscriptionPlan struct {
	Name             string `json:"Name"`
	BillingPeriod    string `json:"BillingPeriod"`
	PriceDescription string `json:"PriceDescription"`
}

// ScheduledChange is a plan change programmed for the next billing period
type ScheduledChange struct {
	TargetBillingPeriod string `json:"TargetBillingPeriod"`
	IsWithdrawable      bool   `json:"IsWithdrawable"`
}

// SubscriptionBenefits says whether SOLO hosts the PFA registered office
type SubscriptionBenefits struct {
	HasProviderHostedHeadquarter bool `json:"HasProviderHostedHeadquarter"`
}

// SubscriptionURL is the web page where subscription changes are made
func SubscriptionURL() string {
	return baseURL + "/settings/subscription"
}

// GetSubscription fetches the subscription summary
func (c *Client) GetSubscription() (*SubscriptionResponse, error) {
	var result SubscriptionResponse
	if err := c.doJSON("GET", "/proxy/accounting/subscription/summary", "/settings/subscription", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return &result, nil
}

// GetSubscriptionBenefits fetches the subscription benefits
func (c *Client) GetSubscriptionBenefits() (*SubscriptionBenefits, error) {
	var result SubscriptionBenefits
	if err := c.doJSON("GET", "/proxy/accounting/subscription/benefits", "/settings/subscription", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get subscription benefits: %w", err)
	}
	return &result, nil
}
