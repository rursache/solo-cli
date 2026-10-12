package client

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

const (
	vatAlertPath  = "/proxy/accounting/vat/alert-status"
	caenAlertPath = "/proxy/accounting/caen-update/alert-status"
	poaAlertPath  = "/proxy/accounting/document-workflows/notarial-power-of-attorney/status"
)

func alertServer(t *testing.T, bodies map[string]string) *Client {
	return newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || !strings.HasSuffix(r.Header.Get("Referer"), "/dashboard") {
			t.Errorf("bad request: %s %s referer %q", r.Method, r.URL.Path, r.Header.Get("Referer"))
		}
		body, ok := bodies[r.URL.Path]
		if !ok {
			http.Error(w, "nope", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(body))
	}))
}

func TestGetAlertsNone(t *testing.T) {
	c := alertServer(t, map[string]string{
		vatAlertPath:  `{"Status":"NotAvailable","CriticalThresholdValue":0}`,
		caenAlertPath: `{"Status":"NotAvailable","IsPfi":false}`,
		poaAlertPath:  `{"Status":"NotAvailable","Template":"PFA_VIES","IsTransitioning":false,"RejectionReasonCode":null,"IsRejectionRecoverable":false,"PickUpAddress":null}`,
	})
	a := c.GetAlerts()
	if len(a.Errors) != 0 || a.VAT == nil || a.CAEN == nil || a.PowerOfAttorney == nil {
		t.Fatalf("alerts = %+v", a)
	}
	if a.PowerOfAttorney.Template != "PFA_VIES" {
		t.Errorf("procura = %+v", a.PowerOfAttorney)
	}
	if active := a.Active(); len(active) != 0 {
		t.Errorf("active = %+v", active)
	}
}

func TestGetAlertsActive(t *testing.T) {
	c := alertServer(t, map[string]string{
		vatAlertPath:  `{"Status":"CriticalPending","CriticalThresholdValue":"300000"}`,
		caenAlertPath: `{"Status":"requested","IsPfi":true}`,
		poaAlertPath:  `{"Status":"InProgress","Template":"PFA_VIES","RejectionReasonCode":"BadId","IsRejectionRecoverable":true}`,
	})
	active := c.GetAlerts().Active()
	if len(active) != 3 {
		t.Fatalf("active = %+v", active)
	}
	vat, caen, poa := active[0], active[1], active[2]
	if vat.Kind != AlertVAT || vat.Severity != AlertCritical || !strings.Contains(vat.Message, "300000 RON") || vat.URL != "https://falcon.solo.ro/vat" {
		t.Errorf("vat = %+v", vat)
	}
	if caen.Kind != AlertCAEN || caen.URL != "https://falcon.solo.ro/caen" {
		t.Errorf("caen = %+v", caen)
	}
	if poa.Kind != AlertPowerOfAttorney || poa.Severity != AlertCritical || !strings.Contains(poa.Message, "BadId") || poa.URL != "https://falcon.solo.ro/procura" {
		t.Errorf("procura = %+v", poa)
	}
}

// A failing source is skipped and reported, the others still load
func TestGetAlertsBestEffort(t *testing.T) {
	c := alertServer(t, map[string]string{
		vatAlertPath: `{"Status":"DangerPending","CriticalThresholdValue":0}`,
		poaAlertPath: `{"Status":"Started"}`,
	})
	a := c.GetAlerts()
	if len(a.Errors) != 1 || a.Errors[0].Kind != AlertCAEN || a.CAEN != nil {
		t.Fatalf("errors = %+v", a.Errors)
	}
	var target *AlertError
	if !errors.As(error(a.Errors[0]), &target) || !strings.Contains(a.Errors[0].Error(), "caen alert") {
		t.Errorf("error text = %v", a.Errors[0])
	}
	active := a.Active()
	if len(active) != 2 || active[0].Severity != AlertWarning || strings.Contains(active[0].Message, " of ") {
		t.Errorf("active = %+v", active)
	}
	if active[1].Severity != AlertInfo || !strings.Contains(active[1].Message, "Started") {
		t.Errorf("procura = %+v", active[1])
	}
}

func TestAlertStatusRules(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   bool
	}{{"DangerPending", true}, {"criticalpending", true}, {"NotAvailable", false}, {"Resolved", false}, {"", false}} {
		if got := (&VATAlertStatus{Status: tc.status}).Alert() != nil; got != tc.want {
			t.Errorf("VAT %q active = %v", tc.status, got)
		}
	}
	for _, tc := range []struct {
		status string
		want   bool
	}{{"Requested", true}, {"NotAvailable", false}, {"Completed", false}} {
		if got := (&CAENAlertStatus{Status: tc.status}).Alert() != nil; got != tc.want {
			t.Errorf("CAEN %q active = %v", tc.status, got)
		}
	}
	for _, tc := range []struct {
		status string
		want   bool
	}{{"NotAvailable", false}, {"notavailable", false}, {"", false}, {"Shipped", true}} {
		if got := (&PowerOfAttorneyStatus{Status: tc.status}).Alert() != nil; got != tc.want {
			t.Errorf("procura %q active = %v", tc.status, got)
		}
	}
}
