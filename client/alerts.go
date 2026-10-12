package client

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// AlertKind names the dashboard alert source
type AlertKind string

const (
	AlertVAT             AlertKind = "vat"
	AlertCAEN            AlertKind = "caen"
	AlertPowerOfAttorney AlertKind = "procura"
)

// AlertSeverity ranks an alert, critical ones need action now
type AlertSeverity string

const (
	AlertInfo     AlertSeverity = "info"
	AlertWarning  AlertSeverity = "warning"
	AlertCritical AlertSeverity = "critical"
)

// Alert is an active dashboard notice that continues on the web
type Alert struct {
	Kind     AlertKind
	Severity AlertSeverity
	Title    string
	Message  string
	URL      string // falcon.solo.ro page to open
}

// VATAlertStatus is the VAT exemption threshold warning
type VATAlertStatus struct {
	Status                 string  `json:"Status"`
	CriticalThresholdValue float64 `json:"CriticalThresholdValue"` // likely the threshold in RON, 0 when missing
}

func (v *VATAlertStatus) UnmarshalJSON(b []byte) error {
	var raw struct {
		Status                 lenientString
		CriticalThresholdValue lenientNumber
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*v = VATAlertStatus{Status: raw.Status.v, CriticalThresholdValue: valueOr0(raw.CriticalThresholdValue.v)}
	return nil
}

// Alert is nil unless the status is DangerPending or CriticalPending, the
// only two the web renders
func (v *VATAlertStatus) Alert() *Alert {
	threshold := "the VAT exemption threshold"
	if v.CriticalThresholdValue > 0 {
		threshold = fmt.Sprintf("the VAT exemption threshold of %.0f RON", v.CriticalThresholdValue)
	}
	switch strings.ToLower(v.Status) {
	case "dangerpending":
		return &Alert{
			Kind:     AlertVAT,
			Severity: AlertWarning,
			Title:    "Close to the VAT threshold",
			Message:  "Revenues are getting close to " + threshold,
			URL:      WebBaseURL + "/vat",
		}
	case "criticalpending":
		return &Alert{
			Kind:     AlertVAT,
			Severity: AlertCritical,
			Title:    "VAT threshold reached",
			Message:  "Revenues reached " + threshold + ", VAT registration is needed",
			URL:      WebBaseURL + "/vat",
		}
	}
	return nil
}

// CAENAlertStatus is a pending CAEN code update workflow
type CAENAlertStatus struct {
	Status string `json:"Status"`
	IsPfi  bool   `json:"IsPfi"`
}

func (c *CAENAlertStatus) UnmarshalJSON(b []byte) error {
	var raw struct {
		Status lenientString
		IsPfi  lenientBool
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*c = CAENAlertStatus{Status: raw.Status.v, IsPfi: raw.IsPfi.v}
	return nil
}

// Alert is nil unless the status is Requested
func (c *CAENAlertStatus) Alert() *Alert {
	if !strings.EqualFold(c.Status, "Requested") {
		return nil
	}
	return &Alert{
		Kind:     AlertCAEN,
		Severity: AlertWarning,
		Title:    "CAEN code update",
		Message:  "The CAEN codes of your activity need updating, continue on the web",
		URL:      WebBaseURL + "/caen",
	}
}

// PowerOfAttorneyStatus is the notarial power of attorney (procura)
// workflow. The address and shipping fields are left out, their shape was
// never seen with data
type PowerOfAttorneyStatus struct {
	Status                 string `json:"Status"`
	Template               string `json:"Template"`
	IsTransitioning        bool   `json:"IsTransitioning"`
	RejectionReasonCode    string `json:"RejectionReasonCode"`
	IsRejectionRecoverable bool   `json:"IsRejectionRecoverable"`
}

func (p *PowerOfAttorneyStatus) UnmarshalJSON(b []byte) error {
	var raw struct {
		Status                 lenientString
		Template               lenientString
		IsTransitioning        lenientBool
		RejectionReasonCode    lenientString
		IsRejectionRecoverable lenientBool
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*p = PowerOfAttorneyStatus{
		Status:                 raw.Status.v,
		Template:               raw.Template.v,
		IsTransitioning:        raw.IsTransitioning.v,
		RejectionReasonCode:    raw.RejectionReasonCode.v,
		IsRejectionRecoverable: raw.IsRejectionRecoverable.v,
	}
	return nil
}

// Alert is nil for an empty status or NotAvailable, the web shows the
// banner for anything else
func (p *PowerOfAttorneyStatus) Alert() *Alert {
	if p.Status == "" || strings.EqualFold(p.Status, "NotAvailable") {
		return nil
	}
	if p.RejectionReasonCode != "" {
		msg := "See the reason and the next steps on the web"
		if p.IsRejectionRecoverable {
			msg = "Fix the details and send the request again on the web"
		}
		return &Alert{
			Kind:     AlertPowerOfAttorney,
			Severity: AlertCritical,
			Title:    "Power of attorney rejected",
			Message:  msg + " (" + p.RejectionReasonCode + ")",
			URL:      WebBaseURL + "/procura",
		}
	}
	return &Alert{
		Kind:     AlertPowerOfAttorney,
		Severity: AlertInfo,
		Title:    "Notarial power of attorney",
		Message:  "A power of attorney request is in progress (" + p.Status + ")",
		URL:      WebBaseURL + "/procura",
	}
}

// GetVATAlertStatus fetches the VAT threshold alert status
func (c *Client) GetVATAlertStatus() (*VATAlertStatus, error) {
	var s VATAlertStatus
	if err := c.doJSON("GET", "/proxy/accounting/vat/alert-status", "/dashboard", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetCAENAlertStatus fetches the CAEN code update alert status
func (c *Client) GetCAENAlertStatus() (*CAENAlertStatus, error) {
	var s CAENAlertStatus
	if err := c.doJSON("GET", "/proxy/accounting/caen-update/alert-status", "/dashboard", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetPowerOfAttorneyStatus fetches the notarial power of attorney status
func (c *Client) GetPowerOfAttorneyStatus() (*PowerOfAttorneyStatus, error) {
	var s PowerOfAttorneyStatus
	if err := c.doJSON("GET", "/proxy/accounting/document-workflows/notarial-power-of-attorney/status", "/dashboard", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// AlertError is one alert source that could not be loaded
type AlertError struct {
	Kind AlertKind
	Err  error
}

func (e *AlertError) Error() string {
	return fmt.Sprintf("%s alert: %v", e.Kind, e.Err)
}

func (e *AlertError) Unwrap() error { return e.Err }

// Alerts is the combined dashboard alert state. Each source is best effort:
// a failed one is nil and listed in Errors
type Alerts struct {
	VAT             *VATAlertStatus
	CAEN            *CAENAlertStatus
	PowerOfAttorney *PowerOfAttorneyStatus
	Errors          []*AlertError
}

// Active lists the alerts to show, in VAT, CAEN, procura order
func (a *Alerts) Active() []Alert {
	var out []Alert
	if a.VAT != nil {
		if al := a.VAT.Alert(); al != nil {
			out = append(out, *al)
		}
	}
	if a.CAEN != nil {
		if al := a.CAEN.Alert(); al != nil {
			out = append(out, *al)
		}
	}
	if a.PowerOfAttorney != nil {
		if al := a.PowerOfAttorney.Alert(); al != nil {
			out = append(out, *al)
		}
	}
	return out
}

// GetAlerts loads the three dashboard alert statuses concurrently. It never
// fails as a whole, check Errors for the sources that could not be loaded
func (c *Client) GetAlerts() *Alerts {
	var (
		a                       Alerts
		wg                      sync.WaitGroup
		vatErr, caenErr, poaErr error
	)
	wg.Add(3)
	go func() { defer wg.Done(); a.VAT, vatErr = c.GetVATAlertStatus() }()
	go func() { defer wg.Done(); a.CAEN, caenErr = c.GetCAENAlertStatus() }()
	go func() { defer wg.Done(); a.PowerOfAttorney, poaErr = c.GetPowerOfAttorneyStatus() }()
	wg.Wait()

	for _, e := range []*AlertError{{AlertVAT, vatErr}, {AlertCAEN, caenErr}, {AlertPowerOfAttorney, poaErr}} {
		if e.Err != nil {
			a.Errors = append(a.Errors, e)
		}
	}
	return &a
}
