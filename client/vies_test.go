package client

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGetViesProfile(t *testing.T) {
	const id = "0123456789abcdef0123456789abcdef"
	body := ""
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/accounting/company/vies-profile/company_"+id || !strings.HasSuffix(r.Header.Get("Referer"), "/settings") {
			t.Errorf("unexpected request %s referer %s", r.URL.Path, r.Header.Get("Referer"))
		}
		w.Write([]byte(body))
	}))

	body = `{"Data":{"Code3":null,"ViesProfileStatus":"VatRegistered","ViesWorkflowStatus":"None"},"Ok":true,"ErrorCode":null,"ErrorMessage":null}`
	p, err := c.GetViesProfile(id)
	if err != nil || p.ViesProfileStatus != "VatRegistered" || p.ViesWorkflowStatus != "None" || p.Code3 != nil {
		t.Errorf("GetViesProfile = %+v, %v", p, err)
	}

	body = `{"Data":{"Code3":"RO12345678","ViesProfileStatus":"VatRegistered","ViesWorkflowStatus":"None"},"Ok":true}`
	if p, err := c.GetViesProfile(id); err != nil || p.Code3 == nil || *p.Code3 != "RO12345678" {
		t.Errorf("Code3 not decoded: %+v, %v", p, err)
	}

	body = `{"Data":null,"Ok":false,"ErrorCode":"NoProfile","ErrorMessage":null}`
	var apiErr *APIError
	if _, err := c.GetViesProfile(id); !errors.As(err, &apiErr) || apiErr.Code != "NoProfile" {
		t.Errorf("Ok=false = %v", err)
	}

	body = `{"Data":null,"Ok":true}`
	if _, err := c.GetViesProfile(id); err == nil {
		t.Error("null Data must fail")
	}
	if _, err := c.GetViesProfile(""); err == nil {
		t.Error("missing company ID must fail")
	}
}
