package client

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGetSubscription(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.Header.Get("Referer"), "/settings/subscription") {
			t.Errorf("referer = %s", r.Header.Get("Referer"))
		}
		switch r.URL.Path {
		case "/proxy/accounting/subscription/summary":
			w.Write([]byte(`{
				"SubscriptionSummary":{
					"Subscription":{"Name":"Test Plan","Status":"Overdue","IsCollectionPaused":true,"NextBillingDate":"2027-01-01","ProductPrice":1000,"HasUpcomingPriceChange":false,"UpcomingProductPrice":0,"UpcomingProductName":null,"UpcomingPriceCode":null,"UpcomingChangeOn":null,"UpcomingInvoiceTotal":0,"DiscountedAmount":0,"CreatedOn":"2024-01-01","CancelledOn":null},
					"OverduePaymentSummary":{"Total":100,"PaymentCount":2},
					"DefaultCard":{"ExpiringMonth":8,"ExpiringYear":2030,"IsExpired":false,"LastDigits":"0000","CardBrand":"visa"}
				},
				"RestartOptions":null,
				"ChangeOptions":{"IsChangeable":true,"CurrentSubscription":{"ProductId":1,"Name":"Test Plan","BillingPeriod":"yearly","PriceDescription":"1000 lei/an"},"Products":[]},
				"ScheduledChange":{"TargetBillingPeriod":"monthly","IsWithdrawable":true}
			}`))
		case "/proxy/accounting/subscription/benefits":
			w.Write([]byte(`{"HasProviderHostedHeadquarter":true}`))
		default:
			http.NotFound(w, r)
		}
	}))

	resp, err := c.GetSubscription()
	if err != nil {
		t.Fatal(err)
	}
	sub := resp.SubscriptionSummary.Subscription
	if sub.Name != "Test Plan" || sub.ProductPrice != 1000 || sub.DisplayStatus() != "Paused" {
		t.Errorf("subscription = %+v, status %s", sub, sub.DisplayStatus())
	}
	if d := resp.SubscriptionSummary.OverduePaymentSummary; d == nil || d.Total != 100 || d.PaymentCount != 2 {
		t.Errorf("overdue = %+v", d)
	}
	if resp.ChangeOptions.CurrentSubscription.BillingPeriod != "yearly" || resp.ScheduledChange.TargetBillingPeriod != "monthly" {
		t.Errorf("options = %+v, scheduled %+v", resp.ChangeOptions.CurrentSubscription, resp.ScheduledChange)
	}
	card := resp.SubscriptionSummary.DefaultCard
	if card.BrandLabel() != "Visa" || card.Expiry() != "08/2030" || card.Expired(time.Date(2030, 8, 31, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("card = %+v", card)
	}

	b, err := c.GetSubscriptionBenefits()
	if err != nil || !b.HasProviderHostedHeadquarter {
		t.Errorf("benefits = %+v, %v", b, err)
	}
}

func TestSubscriptionDisplayStatus(t *testing.T) {
	for _, tc := range []struct {
		status string
		paused bool
		want   string
	}{
		{"Active", false, "Active"},
		{"Overdue", false, "Overdue"},
		{"Overdue", true, "Paused"},
		{"Cancelled", false, "Cancelled"},
		{"Trial", false, "Trial"},
		{"", false, "Unknown"},
	} {
		s := Subscription{Status: tc.status, IsCollectionPaused: tc.paused}
		if got := s.DisplayStatus(); got != tc.want {
			t.Errorf("DisplayStatus(%q, %v) = %q, want %q", tc.status, tc.paused, got, tc.want)
		}
	}
}

func TestPaymentCardExpiry(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		card    PaymentCard
		expiry  string
		expired bool
	}{
		{PaymentCard{ExpiringMonth: 9, ExpiringYear: 2026}, "09/2026", true},
		{PaymentCard{ExpiringMonth: 10, ExpiringYear: 2026}, "10/2026", false},
		{PaymentCard{ExpiringMonth: 1, ExpiringYear: 2030, IsExpired: true}, "01/2030", true},
		{PaymentCard{}, "", false},
		{PaymentCard{ExpiringMonth: 13, ExpiringYear: 2020}, "", false},
	} {
		if tc.card.Expiry() != tc.expiry || tc.card.Expired(now) != tc.expired {
			t.Errorf("%+v: expiry %q expired %v", tc.card, tc.card.Expiry(), tc.card.Expired(now))
		}
	}
	if (&PaymentCard{CardBrand: "amex"}).BrandLabel() != "Card" || (&PaymentCard{CardBrand: "mastercard"}).BrandLabel() != "Mastercard" {
		t.Error("brand labels")
	}
}
