//go:build live

package client

import (
	"testing"

	"solo-cli/config"
)

// Live integration tests against the real SOLO.ro API using the developer's
// own credentials from ~/.config/solo-cli. Strictly read-only: no uploads,
// no deletions. Run with:
//
//	go test -tags live ./client -v
func liveClient(t *testing.T) *Client {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("no usable config, skipping live tests: %v", err)
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = config.DefaultUserAgent
	}
	c, err := New(userAgent)
	if err != nil {
		t.Fatal(err)
	}

	// Reuse the saved session like the CLI does, login only if stale
	if loaded, _ := c.LoadCookies(); loaded {
		if _, err := c.GetSummary(); err == nil {
			return c
		}
	}
	if err := c.Login(cfg.Username, cfg.Password); err != nil {
		t.Fatalf("live login failed: %v", err)
	}
	c.SaveCookies()
	return c
}

func TestLiveSummary(t *testing.T) {
	c := liveClient(t)

	s, err := c.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if s.Year < 2020 {
		t.Errorf("suspicious year: %d", s.Year)
	}
	if s.DisplayCurrency == "" {
		t.Error("DisplayCurrency empty")
	}
	if s.TotalRevenues < 0 || s.TotalDeductibleExpenses < 0 {
		t.Errorf("negative totals: %+v", s)
	}
	t.Logf("summary %d: revenues %.2f, expenses %.2f", s.Year, s.TotalRevenues, s.TotalDeductibleExpenses)
}

func TestLiveListEndpoints(t *testing.T) {
	c := liveClient(t)

	if resp, err := c.ListRevenues(0, 5); err != nil {
		t.Errorf("ListRevenues: %v", err)
	} else {
		t.Logf("revenues: %d items", len(resp.Items))
	}

	if resp, err := c.ListExpenses(0, 5); err != nil {
		t.Errorf("ListExpenses: %v", err)
	} else {
		t.Logf("expenses: %d items", len(resp.Items))
	}

	if resp, err := c.ListQueuedExpenses(0, 5); err != nil {
		t.Errorf("ListQueuedExpenses: %v", err)
	} else {
		t.Logf("queue: %d items", len(resp.Items))
	}

	if resp, err := c.ListRejectedExpenses(0, 5); err != nil {
		t.Errorf("ListRejectedExpenses: %v", err)
	} else {
		t.Logf("rejected: %d items", len(resp.Items))
	}

	if resp, err := c.ListEFactura(0, 5); err != nil {
		t.Errorf("ListEFactura: %v", err)
	} else {
		t.Logf("efactura: %d items", len(resp.Items))
	}
}

func TestLiveCompanyDiscoveryAndProfile(t *testing.T) {
	c := liveClient(t)

	id, err := c.DiscoverCompanyID()
	if err != nil {
		t.Fatalf("DiscoverCompanyID: %v", err)
	}
	if len(id) != 32 {
		t.Errorf("company ID length = %d, want 32", len(id))
	}

	info, err := c.GetCompanyInfo(id)
	if err != nil {
		t.Fatalf("GetCompanyInfo: %v", err)
	}
	if info.Name == "" || info.Code1 == "" {
		t.Errorf("incomplete company profile: %+v", info)
	}
	t.Logf("company: %s (CUI %s)", info.Name, info.Code1)
}
