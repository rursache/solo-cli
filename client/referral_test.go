package client

import (
	"net/http"
	"strings"
	"testing"
)

func TestReferral(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || !strings.HasSuffix(r.Header.Get("Referer"), "/referral") {
			t.Errorf("unexpected request %s referer %s", r.Method, r.Header.Get("Referer"))
		}
		switch r.URL.Path {
		case "/api/referral/campaign":
			w.Write([]byte(`{"ReferralCode":"test123","CurrentCampaign":{"Code":"default","ReferrerPrizeValue":100,"GuestPrizeValue":50,"IsTimeLimited":false,"StartDate":"2026-01-01T00:00:00","EndDate":null}}`))
		case "/api/referral/rewards":
			w.Write([]byte(`{"PendingWinnings":50,"EarnedWinnings":200}`))
		default:
			http.NotFound(w, r)
		}
	}))

	campaign, err := c.GetReferralCampaign()
	if err != nil {
		t.Fatal(err)
	}
	if campaign.Link() != "https://www.solo.ro/ref/test123" {
		t.Errorf("link = %q", campaign.Link())
	}
	cc := campaign.CurrentCampaign
	if cc == nil || cc.ReferrerPrizeValue != 100 || cc.GuestPrizeValue != 50 || cc.EndDate != nil || cc.StartDate == nil {
		t.Errorf("campaign = %+v", cc)
	}

	rewards, err := c.GetReferralRewards()
	if err != nil || rewards.EarnedWinnings != 200 || rewards.PendingWinnings != 50 {
		t.Errorf("rewards = %+v, %v", rewards, err)
	}

	if (&ReferralCampaign{ReferralCode: " "}).Link() != "" {
		t.Error("blank code must give no link")
	}
	if got := (&ReferralCampaign{ReferralCode: "a b"}).Link(); got != "https://www.solo.ro/ref/a%20b" {
		t.Errorf("code must be escaped, got %q", got)
	}
}
