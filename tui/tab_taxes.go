package tui

import (
	"fmt"
	"strings"

	"solo-cli/taxes"
)

// renderTaxesViewport wraps renderTaxes in a manual scroll window sized to
// the terminal height
func (m Model) renderTaxesViewport() string {
	content := m.renderTaxes()
	lines := strings.Split(content, "\n")
	// The scroll hint line is the taxes tab's only chrome inside the body
	availHeight := m.bodyHeight() - 1
	// Clamp scroll
	maxScroll := len(lines) - availHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.taxesScroll > maxScroll {
		m.taxesScroll = maxScroll
	}
	// Slice visible lines
	end := m.taxesScroll + availHeight
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	b.WriteString(strings.Join(lines[m.taxesScroll:end], "\n"))
	if m.taxesScroll > 0 || end < len(lines) {
		b.WriteString("\n")
		b.WriteString(SummaryLabelStyle.Render("↑↓ scroll to see more"))
	}
	return b.String()
}

func (m Model) renderTaxes() string {
	if m.taxBreakdown == nil {
		if m.taxConfig == nil {
			return ErrorStyle.Render("Could not load taxes.json config")
		}
		return LoadingStyle.Render("Loading tax data...")
	}

	t := m.taxBreakdown
	var b strings.Builder

	// Header
	yearStr := ""
	if m.summary != nil {
		yearStr = fmt.Sprintf(" (%d)", m.summary.Year)
	}
	b.WriteString(TitleStyle.Render(fmt.Sprintf("Tax Breakdown%s", yearStr)))
	b.WriteString("\n")

	// Net income summary
	incomeContent := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s (%.1f @ %.0f RON salariu minim brut)",
		SummaryLabelStyle.Render("Total Revenues:"),
		SummaryValueStyle.Render(taxes.FormatRON(m.summary.TotalRevenues)),
		SummaryLabelStyle.Render("Deductible Expenses:"),
		SummaryValueStyle.Render(taxes.FormatRON(m.summary.TotalDeductibleExpenses)),
		SummaryLabelStyle.Render("Net Income:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.NetIncome)),
		t.SalariesCount,
		t.SalariuMinimBrut,
	)
	b.WriteString(CompactBoxStyle.Render(incomeContent))
	b.WriteString("\n")

	// CAS
	casContent := fmt.Sprintf(
		"%s %s\n%s %s → %s %s",
		SummaryLabelStyle.Render("Bracket:"),
		t.CAS.Label,
		SummaryLabelStyle.Render("Base:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.CAS.Base)),
		SummaryLabelStyle.Render("Amount:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.CAS.Amount)),
	)
	casContent += renderThresholdHint(t.CAS)
	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("CAS (%.0f%%)", t.CAS.Percentage)))
	b.WriteString("\n")
	b.WriteString(CompactBoxStyle.Render(casContent))
	b.WriteString("\n")

	// CASS
	cassContent := fmt.Sprintf(
		"%s %s\n%s %s → %s %s",
		SummaryLabelStyle.Render("Bracket:"),
		t.CASS.Label,
		SummaryLabelStyle.Render("Base:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.CASS.Base)),
		SummaryLabelStyle.Render("Amount:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.CASS.Amount)),
	)
	cassContent += renderThresholdHint(t.CASS)
	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("CASS (%.0f%%)", t.CASS.Percentage)))
	b.WriteString("\n")
	b.WriteString(CompactBoxStyle.Render(cassContent))
	b.WriteString("\n")

	// Income Tax
	taxableIncome := t.NetIncome - t.CAS.Amount - t.CASS.Amount
	if taxableIncome < 0 {
		taxableIncome = 0
	}
	itContent := fmt.Sprintf(
		"%s %s (Net Income - CAS - CASS)\n%s %s",
		SummaryLabelStyle.Render("Taxable Income:"),
		SummaryValueStyle.Render(taxes.FormatRON(taxableIncome)),
		SummaryLabelStyle.Render("Amount:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.IncomeTax)),
	)
	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Income Tax (%.0f%%)", m.taxConfig.IncomeTaxPercent)))
	b.WriteString("\n")
	b.WriteString(CompactBoxStyle.Render(itContent))
	b.WriteString("\n")

	// Totals
	totalsContent := fmt.Sprintf(
		"%s %s\n%s %s\n%s %.1f%%",
		SummaryLabelStyle.Render("Total Taxes:"),
		ErrorStyle.Render(taxes.FormatRON(t.TotalTaxes)),
		SummaryLabelStyle.Render("Net After Tax:"),
		PaidStyle.Render(taxes.FormatRON(t.NetAfterTax)),
		SummaryLabelStyle.Render("Effective Tax Rate:"),
		t.EffectiveRate,
	)
	b.WriteString(CompactBoxStyle.Render(totalsContent))

	return b.String()
}

// renderThresholdHint shows a "buffer to next" line if still in the lowest
// bracket, or an actionable "add expenses to drop a bracket" line once a
// threshold has been crossed. Returns "" if no hint applies.
func renderThresholdHint(t taxes.ThresholdResult) string {
	if t.PrevLabel != "" {
		hintStyle := dangerStyle
		if t.ExpensesToPrev < 5000 {
			hintStyle = secondaryStyle
		} else if t.ExpensesToPrev < 15000 {
			hintStyle = warningStyle
		}
		return fmt.Sprintf("\n%s %s → %s",
			SummaryLabelStyle.Render("Surplus:"),
			hintStyle.Render(taxes.FormatRON(t.ExpensesToPrev)),
			SummaryLabelStyle.Render(t.PrevLabel),
		)
	}
	if t.NextLabel != "" {
		bufferStyle := secondaryStyle
		if t.BufferToNext < 5000 {
			bufferStyle = dangerStyle
		} else if t.BufferToNext < 15000 {
			bufferStyle = warningStyle
		}
		return fmt.Sprintf("\n%s %s → %s",
			SummaryLabelStyle.Render("Buffer:"),
			bufferStyle.Render(taxes.FormatRON(t.BufferToNext)),
			SummaryLabelStyle.Render(t.NextLabel),
		)
	}
	return ""
}
