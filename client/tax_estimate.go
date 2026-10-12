package client

import (
	"encoding/json"
	"fmt"
)

// TaxEstimate is SOLO's own yearly CAS, CASS and income tax estimate, the
// breakdown behind the dashboard taxes figure. Amounts are nil when missing
type TaxEstimate struct {
	FiscalYear              int      `json:"FiscalYear"`
	TotalRevenues           *float64 `json:"TotalRevenues"`
	HasDeductibleExpenses   bool     `json:"HasDeductibleExpenses"`
	TotalDeductibleExpenses *float64 `json:"TotalDeductibleExpenses"`
	HasTaxableRevenue       bool     `json:"HasTaxableRevenue"`
	TaxableRevenue          *float64 `json:"TaxableRevenue"`
	IncomeTax               *float64 `json:"IncomeTax"`
	RetirementTax           *float64 `json:"RetirementTax"` // CAS
	HealthTax               *float64 `json:"HealthTax"`     // CASS
	TotalTaxes              *float64 `json:"TotalTaxes"`
}

func (e *TaxEstimate) UnmarshalJSON(b []byte) error {
	var raw struct {
		FiscalYear              lenientNumber
		TotalRevenues           lenientNumber
		HasDeductibleExpenses   lenientBool
		TotalDeductibleExpenses lenientNumber
		HasTaxableRevenue       lenientBool
		TaxableRevenue          lenientNumber
		IncomeTax               lenientNumber
		RetirementTax           lenientNumber
		HealthTax               lenientNumber
		TotalTaxes              lenientNumber
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*e = TaxEstimate{
		FiscalYear:              raw.FiscalYear.int(),
		TotalRevenues:           raw.TotalRevenues.v,
		HasDeductibleExpenses:   raw.HasDeductibleExpenses.v,
		TotalDeductibleExpenses: raw.TotalDeductibleExpenses.v,
		HasTaxableRevenue:       raw.HasTaxableRevenue.v,
		TaxableRevenue:          raw.TaxableRevenue.v,
		IncomeTax:               raw.IncomeTax.v,
		RetirementTax:           raw.RetirementTax.v,
		HealthTax:               raw.HealthTax.v,
		TotalTaxes:              raw.TotalTaxes.v,
	}
	return nil
}

// HasFigures is false when SOLO sent no tax amount at all
func (e *TaxEstimate) HasFigures() bool {
	return e.IncomeTax != nil || e.RetirementTax != nil || e.HealthTax != nil || e.TotalTaxes != nil
}

// Total is TotalTaxes, falling back to the sum of the three taxes like the web
func (e *TaxEstimate) Total() float64 {
	if e.TotalTaxes != nil {
		return *e.TotalTaxes
	}
	return valueOr0(e.IncomeTax) + valueOr0(e.RetirementTax) + valueOr0(e.HealthTax)
}

// GetTaxEstimate fetches SOLO's tax estimate for a fiscal year
func (c *Client) GetTaxEstimate(fiscalYear int) (*TaxEstimate, error) {
	var e TaxEstimate
	path := fmt.Sprintf("/proxy/accounting/tax-declaration/tax-estimate/%d", fiscalYear)
	if err := c.doJSON("GET", path, "/dashboard", nil, &e); err != nil {
		return nil, fmt.Errorf("failed to get tax estimate: %w", err)
	}
	return &e, nil
}
