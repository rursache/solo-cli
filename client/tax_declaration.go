package client

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// WebBaseURL is the SOLO.ro web app, for pointing users at pages the CLI
// does not cover
const WebBaseURL = "https://falcon.solo.ro"

// TaxDeclarationWebURL is where the Declaratia Unica is filed and rectified
const TaxDeclarationWebURL = WebBaseURL + "/tax-declaration/dashboard"

// DeclarationStatus is the Declaratia Unica (D212) filing state of a year
type DeclarationStatus string

const (
	DeclarationNone                  DeclarationStatus = "None"
	DeclarationFormAutoFiling        DeclarationStatus = "FormCompletedAutoFiling"
	DeclarationFormManualFiling      DeclarationStatus = "FormCompletedManualFiling"
	DeclarationFormNoFiling          DeclarationStatus = "FormCompletedNoFiling"
	DeclarationFormPnlExpired        DeclarationStatus = "FormCompletedPnlExpired"
	DeclarationAutoFilingSoon        DeclarationStatus = "DeclarationAutoFilingSoon"
	DeclarationAutoFilingError       DeclarationStatus = "DeclarationAutoFilingError"
	DeclarationAutoFiledOk           DeclarationStatus = "DeclarationAutoFiledOk"
	DeclarationManualFilingStarted   DeclarationStatus = "DeclarationManualFilingStarted"
	DeclarationManualFilingCompleted DeclarationStatus = "DeclarationManualFilingCompleted"
	DeclarationPnlExpired            DeclarationStatus = "DeclarationPnlExpired"
	DeclarationCustom                DeclarationStatus = "CustomDeclaration"
)

var declarationLabels = map[DeclarationStatus]string{
	DeclarationNone:                  "Not started",
	DeclarationFormAutoFiling:        "Form completed, SOLO will file it",
	DeclarationFormManualFiling:      "Form completed, you file it at ANAF",
	DeclarationFormNoFiling:          "Form completed, nothing to file",
	DeclarationFormPnlExpired:        "Form completed but the power of attorney expired",
	DeclarationAutoFilingSoon:        "SOLO will file it soon",
	DeclarationAutoFilingError:       "Automatic filing failed",
	DeclarationAutoFiledOk:           "Filed at ANAF by SOLO",
	DeclarationManualFilingStarted:   "Filing by yourself, not finished",
	DeclarationManualFilingCompleted: "Filed at ANAF by you",
	DeclarationPnlExpired:            "Power of attorney expired before filing",
	DeclarationCustom:                "Custom declaration handled by SOLO",
}

// normalizeDeclarationStatus maps a server value onto the known constants
// ignoring case, unknown values are kept as sent
func normalizeDeclarationStatus(raw string) DeclarationStatus {
	for s := range declarationLabels {
		if strings.EqualFold(string(s), raw) {
			return s
		}
	}
	return DeclarationStatus(raw)
}

// Label is a short human description, "Unknown status" for values the CLI
// does not know
func (s DeclarationStatus) Label() string {
	if l, ok := declarationLabels[s]; ok {
		return l
	}
	return "Unknown status"
}

// Known reports whether s is one of the documented statuses
func (s DeclarationStatus) Known() bool {
	_, ok := declarationLabels[s]
	return ok
}

// Filed is true once the declaration reached ANAF or needs no filing
func (s DeclarationStatus) Filed() bool {
	switch s {
	case DeclarationAutoFiledOk, DeclarationManualFilingCompleted, DeclarationFormNoFiling, DeclarationCustom:
		return true
	}
	return false
}

// NeedsAttention is true for failed or blocked filings
func (s DeclarationStatus) NeedsAttention() bool {
	switch s {
	case DeclarationAutoFilingError, DeclarationFormPnlExpired, DeclarationPnlExpired:
		return true
	}
	return false
}

// TaxDeclarationDashboard is the Declaratia Unica status of one fiscal year
// plus the taxes SOLO computed for it. An unfiled year comes back with
// DeclarationStatus None and nil taxes
type TaxDeclarationDashboard struct {
	FiscalYear          int               `json:"FiscalYear"`
	ReportingFiscalYear int               `json:"ReportingFiscalYear"` // filing year, 0 when missing
	DeclarationStatus   DeclarationStatus `json:"DeclarationStatus"`
	// LastCompletedWorkflow is an opaque workflow id like "wfc_..."
	LastCompletedWorkflow string                   `json:"LastCompletedWorkflow"`
	LastCompletedOn       string                   `json:"LastCompletedOn"` // YYYY-MM-DD
	HasDueInFiscalYear    bool                     `json:"HasDueInFiscalYear"`
	TaxesSummary          *DeclarationTaxesSummary `json:"TaxesSummary"`
}

func (d *TaxDeclarationDashboard) UnmarshalJSON(b []byte) error {
	var raw struct {
		FiscalYear            lenientNumber
		ReportingFiscalYear   lenientNumber
		DeclarationStatus     lenientString
		LastCompletedWorkflow lenientString
		LastCompletedOn       lenientString
		HasDueInFiscalYear    lenientBool
		TaxesSummary          json.RawMessage
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*d = TaxDeclarationDashboard{
		FiscalYear:            raw.FiscalYear.int(),
		ReportingFiscalYear:   raw.ReportingFiscalYear.int(),
		DeclarationStatus:     normalizeDeclarationStatus(raw.DeclarationStatus.v),
		LastCompletedWorkflow: raw.LastCompletedWorkflow.v,
		LastCompletedOn:       raw.LastCompletedOn.v,
		HasDueInFiscalYear:    raw.HasDueInFiscalYear.v,
	}
	// A malformed summary must not hide the filing status
	if len(raw.TaxesSummary) > 0 && string(raw.TaxesSummary) != "null" {
		var s DeclarationTaxesSummary
		if json.Unmarshal(raw.TaxesSummary, &s) == nil {
			d.TaxesSummary = &s
		}
	}
	return nil
}

// FilingYear is the year the declaration is filed in, the one after the
// fiscal year it covers
func (d *TaxDeclarationDashboard) FilingYear() int {
	if d.ReportingFiscalYear > 0 {
		return d.ReportingFiscalYear
	}
	return d.FiscalYear + 1
}

// Deadline is 25 May of the filing year, Bucharest time. Filing and payment
// are due by the end of that day
func (d *TaxDeclarationDashboard) Deadline() time.Time {
	return time.Date(d.FilingYear(), time.May, 25, 0, 0, 0, 0, bucharest())
}

// HasTaxes is false for unfiled years, whose summary carries only nulls
func (d *TaxDeclarationDashboard) HasTaxes() bool {
	return d.TaxesSummary != nil && d.TaxesSummary.HasTaxes()
}

func bucharest() *time.Location {
	if loc, err := time.LoadLocation("Europe/Bucharest"); err == nil {
		return loc
	}
	// No tzdata on this system, May is always summer time (UTC+3)
	return time.FixedZone("EEST", 3*3600)
}

// DeclarationTaxesSummary holds the taxes SOLO computed for a declaration.
// Amounts are nil when missing
type DeclarationTaxesSummary struct {
	IncomeTax              *float64                   `json:"IncomeTax"`
	HealthTax              *float64                   `json:"HealthTax"`              // CASS on the PFA income
	HealthOtherRevenuesTax *float64                   `json:"HealthOtherRevenuesTax"` // CASS on other income
	TotalHealthTaxes       *float64                   `json:"TotalHealthTaxes"`
	RetirementTax          *float64                   `json:"RetirementTax"` // CAS
	TotalTaxes             *float64                   `json:"TotalTaxes"`
	IncomeTaxDiscount      *float64                   `json:"IncomeTaxDiscount"`
	TotalTaxesDiscounted   *float64                   `json:"TotalTaxesDiscounted"`
	TotalRevenues          *float64                   `json:"TotalRevenues"`
	TotalOtherRevenues     *float64                   `json:"TotalOtherRevenues"`
	HasCompanyRevenues     bool                       `json:"HasCompanyRevenues"`
	CompanyRevenue         *DeclarationCompanyRevenue `json:"CompanyRevenue"`
	RevenueTaxes           []DeclarationRevenueTax    `json:"RevenueTaxes"`
}

// DeclarationCompanyRevenue is the PFA income the declaration is based on
type DeclarationCompanyRevenue struct {
	TotalRevenues      *float64 `json:"TotalRevenues"`
	NetRevenues        *float64 `json:"NetRevenues"`
	DeductibleExpenses *float64 `json:"DeductibleExpenses"`
}

// DeclarationRevenueTax is the income tax for one revenue type
type DeclarationRevenueTax struct {
	RevenueTypeID int      `json:"RevenueTypeId"`
	Name          string   `json:"Name"`
	Sequence      int      `json:"Sequence"`
	Tax           *float64 `json:"Tax"`
}

func (s *DeclarationTaxesSummary) UnmarshalJSON(b []byte) error {
	var raw struct {
		IncomeTax              lenientNumber
		HealthTax              lenientNumber
		HealthOtherRevenuesTax lenientNumber
		TotalHealthTaxes       lenientNumber
		RetirementTax          lenientNumber
		TotalTaxes             lenientNumber
		IncomeTaxDiscount      lenientNumber
		TotalTaxesDiscounted   lenientNumber
		TotalRevenues          lenientNumber
		TotalOtherRevenues     lenientNumber
		HasCompanyRevenues     lenientBool
		CompanyRevenue         *struct {
			TotalRevenues      lenientNumber
			NetRevenues        lenientNumber
			DeductibleExpenses lenientNumber
		}
		RevenueTaxes []struct {
			RevenueTypeId lenientNumber
			Name          lenientString
			Sequence      lenientNumber
			Tax           lenientNumber
		}
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*s = DeclarationTaxesSummary{
		IncomeTax:              raw.IncomeTax.v,
		HealthTax:              raw.HealthTax.v,
		HealthOtherRevenuesTax: raw.HealthOtherRevenuesTax.v,
		TotalHealthTaxes:       raw.TotalHealthTaxes.v,
		RetirementTax:          raw.RetirementTax.v,
		TotalTaxes:             raw.TotalTaxes.v,
		IncomeTaxDiscount:      raw.IncomeTaxDiscount.v,
		TotalTaxesDiscounted:   raw.TotalTaxesDiscounted.v,
		TotalRevenues:          raw.TotalRevenues.v,
		TotalOtherRevenues:     raw.TotalOtherRevenues.v,
		HasCompanyRevenues:     raw.HasCompanyRevenues.v,
	}
	if cr := raw.CompanyRevenue; cr != nil {
		s.CompanyRevenue = &DeclarationCompanyRevenue{
			TotalRevenues:      cr.TotalRevenues.v,
			NetRevenues:        cr.NetRevenues.v,
			DeductibleExpenses: cr.DeductibleExpenses.v,
		}
	}
	for _, rt := range raw.RevenueTaxes {
		s.RevenueTaxes = append(s.RevenueTaxes, DeclarationRevenueTax{
			RevenueTypeID: rt.RevenueTypeId.int(),
			Name:          rt.Name.v,
			Sequence:      rt.Sequence.int(),
			Tax:           rt.Tax.v,
		})
	}
	return nil
}

// HasTaxes is false when all four taxes are null (an unfiled year)
func (s *DeclarationTaxesSummary) HasTaxes() bool {
	return s.IncomeTax != nil || s.HealthTax != nil || s.HealthOtherRevenuesTax != nil || s.RetirementTax != nil
}

// HealthTaxes is CASS on the PFA income plus CASS on other income
func (s *DeclarationTaxesSummary) HealthTaxes() float64 {
	return valueOr0(s.HealthTax) + valueOr0(s.HealthOtherRevenuesTax)
}

// Total sums the four taxes with nulls as zero, like the web (which ignores
// TotalTaxes)
func (s *DeclarationTaxesSummary) Total() float64 {
	return valueOr0(s.IncomeTax) + valueOr0(s.RetirementTax) + s.HealthTaxes()
}

// Discount is the early filing income tax discount, never negative
func (s *DeclarationTaxesSummary) Discount() float64 {
	return max(0, valueOr0(s.IncomeTaxDiscount))
}

// DiscountedTotal is TotalTaxesDiscounted when sent, else Total minus Discount
func (s *DeclarationTaxesSummary) DiscountedTotal() float64 {
	if s.TotalTaxesDiscounted != nil && *s.TotalTaxesDiscounted > 0 {
		return *s.TotalTaxesDiscounted
	}
	return s.Total() - s.Discount()
}

// GetTaxDeclarationDashboard fetches the Declaratia Unica status and taxes
// for a fiscal year
func (c *Client) GetTaxDeclarationDashboard(fiscalYear int) (*TaxDeclarationDashboard, error) {
	var d TaxDeclarationDashboard
	path := fmt.Sprintf("/proxy/accounting/tax-declaration/tax-dashboard/%d", fiscalYear)
	if err := c.doJSON("GET", path, "/tax-declaration/dashboard", nil, &d); err != nil {
		return nil, fmt.Errorf("failed to get tax declaration: %w", err)
	}
	return &d, nil
}
