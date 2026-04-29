package taxes

import (
	"fmt"
	"math"

	"solo-cli/config"
)

// ThresholdResult describes the tax computed for a specific contribution
type ThresholdResult struct {
	Label          string
	Percentage     float64
	Base           float64
	Amount         float64
	NextLabel      string  // label of the next threshold (empty if at max)
	BufferToNext   float64 // how much more net income before reaching the next threshold
	PrevLabel      string  // label of the previous (lower) bracket (empty if already at lowest)
	ExpensesToPrev float64 // extra deductible expenses needed to drop into the previous bracket
}

// TaxBreakdown holds the full tax calculation result
type TaxBreakdown struct {
	NetIncome       float64
	SalariuMinimBrut float64
	SalariesCount   float64 // net income expressed in multiples of SMB

	CAS       ThresholdResult
	CASS      ThresholdResult
	IncomeTax float64 // 10% of (net income - CAS - CASS)

	TotalTaxes float64
	NetAfterTax float64
	EffectiveRate float64 // total taxes / net income * 100
}

// Calculate computes the full tax breakdown from revenues and expenses
func Calculate(totalRevenues, totalExpenses float64, cfg *config.TaxConfig) *TaxBreakdown {
	netIncome := totalRevenues - totalExpenses
	if netIncome < 0 {
		netIncome = 0
	}

	smb := cfg.SalariuMinimBrut
	salaries := netIncome / smb

	cas := calculateContribution(netIncome, salaries, smb, cfg.CASPercent, cfg.CASThresholds)
	cass := calculateContribution(netIncome, salaries, smb, cfg.CASSPercent, cfg.CASSThresholds)

	// Income tax = percentage of (net income - CAS - CASS)
	taxableIncome := netIncome - cas.Amount - cass.Amount
	if taxableIncome < 0 {
		taxableIncome = 0
	}
	incomeTax := math.Round(taxableIncome*cfg.IncomeTaxPercent) / 100

	totalTaxes := cas.Amount + cass.Amount + incomeTax
	netAfterTax := netIncome - totalTaxes

	effectiveRate := 0.0
	if netIncome > 0 {
		effectiveRate = totalTaxes / netIncome * 100
	}

	return &TaxBreakdown{
		NetIncome:        netIncome,
		SalariuMinimBrut: smb,
		SalariesCount:    salaries,
		CAS:              cas,
		CASS:             cass,
		IncomeTax:        incomeTax,
		TotalTaxes:       totalTaxes,
		NetAfterTax:      netAfterTax,
		EffectiveRate:    effectiveRate,
	}
}

func calculateContribution(netIncome, salaries, smb, percent float64, thresholds []config.TaxThreshold) ThresholdResult {
	result := ThresholdResult{Percentage: percent}

	for i, t := range thresholds {
		maxSal := t.MaxSalaries
		if maxSal == 0 {
			maxSal = math.MaxFloat64
		}

		if salaries >= t.MinSalaries && salaries < maxSal {
			result.Label = t.Label

			switch {
			case t.BaseSalaries == 0:
				// Exempt
				result.Base = 0
				result.Amount = 0
			case t.BaseSalaries == -1:
				// Proportional: use actual net income
				result.Base = netIncome
				result.Amount = math.Round(netIncome*percent) / 100
			default:
				// Fixed: base = BaseSalaries * SMB
				result.Base = t.BaseSalaries * smb
				result.Amount = math.Round(t.BaseSalaries*smb*percent) / 100
			}

			// Calculate buffer to next threshold
			if i+1 < len(thresholds) {
				next := thresholds[i+1]
				result.NextLabel = next.Label
				nextThresholdIncome := next.MinSalaries * smb
				result.BufferToNext = nextThresholdIncome - netIncome
				if result.BufferToNext < 0 {
					result.BufferToNext = 0
				}
			}

			// Expenses needed to drop back into the previous bracket — only
			// surface the hint when the contribution saving actually beats the
			// required expense. Otherwise the suggestion is bad advice.
			if i > 0 {
				prev := thresholds[i-1]
				currentMinIncome := t.MinSalaries * smb
				expensesNeeded := math.Round((netIncome-currentMinIncome)*100)/100 + 1
				if expensesNeeded < 0 {
					expensesNeeded = 0
				}

				// Hypothetical income just below current bracket's lower bound
				hypoIncome := currentMinIncome - 1
				var prevAmount float64
				switch {
				case prev.BaseSalaries == 0:
					prevAmount = 0
				case prev.BaseSalaries == -1:
					prevAmount = math.Round(hypoIncome*percent) / 100
				default:
					prevAmount = math.Round(prev.BaseSalaries*smb*percent) / 100
				}

				if result.Amount-prevAmount > expensesNeeded {
					result.PrevLabel = prev.Label
					result.ExpensesToPrev = expensesNeeded
				}
			}

			return result
		}
	}

	// Fallback: last threshold (shouldn't normally reach here)
	if len(thresholds) > 0 {
		last := thresholds[len(thresholds)-1]
		result.Label = last.Label
		if last.BaseSalaries > 0 {
			result.Base = last.BaseSalaries * smb
			result.Amount = math.Round(last.BaseSalaries*smb*percent) / 100
		}
	}

	return result
}

// FormatRON formats a float as RON currency
func FormatRON(amount float64) string {
	return fmt.Sprintf("%.2f RON", amount)
}

// FormatBuffer returns a human-readable buffer description
func FormatBuffer(buffer float64) string {
	if buffer <= 0 {
		return "plafonul a fost atins"
	}
	return fmt.Sprintf("%.2f RON rămas până la următorul plafon", buffer)
}

// FormatExpensesHint returns a human-readable hint for dropping a bracket
func FormatExpensesHint(amount float64) string {
	return fmt.Sprintf("Surplus: %s (adaugă cheltuieli pentru a coborî sub plafon)", FormatRON(amount))
}
