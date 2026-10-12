package client

import (
	"net/http"
	"testing"
)

func TestDiscoverCompanyStartYear(t *testing.T) {
	page := ""
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(page))
	}))

	page = `<script>var Principal = { CompanyCode: "company_0123456789abcdef0123456789abcdef", CompanyStartYear: 2021 };</script>`
	id, err := c.DiscoverCompanyID()
	if err != nil || id != "0123456789abcdef0123456789abcdef" || c.CompanyStartYear != 2021 {
		t.Errorf("DiscoverCompanyID = %q, %v, start %d", id, err, c.CompanyStartYear)
	}

	page = `<script>var Principal = { CompanyCode: "company_0123456789abcdef0123456789abcdef" };</script>`
	if _, err := c.DiscoverCompanyID(); err != nil || c.CompanyStartYear != 0 {
		t.Errorf("missing start year: %v, start %d", err, c.CompanyStartYear)
	}

	page = `<p>no principal</p>`
	c.CompanyStartYear = 2020
	if _, err := c.DiscoverCompanyID(); err == nil || c.CompanyStartYear != 2020 {
		t.Errorf("failed discovery must keep the old start year: %v, %d", err, c.CompanyStartYear)
	}
}

func TestParseStartYear(t *testing.T) {
	for _, tc := range []struct {
		body string
		want int
	}{
		{`CompanyStartYear: 2019,`, 2019},
		{`CompanyStartYear:"2022"`, 2022},
		{`CompanyStartYear: 1989`, 0},
		{`CompanyStartYear: 2031`, 0},
		{`CompanyStartYear: null`, 0},
		{``, 0},
	} {
		if got := parseStartYear([]byte(tc.body), 2030); got != tc.want {
			t.Errorf("parseStartYear(%q) = %d, want %d", tc.body, got, tc.want)
		}
	}
}
