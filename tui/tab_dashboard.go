package tui

import (
	"fmt"
	"strings"
)

func (m Model) renderDashboard() string {
	if m.summary == nil {
		return LoadingStyle.Render("Loading summary...")
	}

	var b strings.Builder

	// Company info header
	if m.company != nil {
		b.WriteString(TitleStyle.Render(m.company.Name))
		b.WriteString("\n")
		b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("CUI: %s • Reg: %s", m.company.Code1, m.company.Code2)))
		b.WriteString("\n\n")
	} else if m.client.CompanyID == "" {
		b.WriteString(ErrorStyle.Render("‼️  Could not determine company ID"))
		b.WriteString("\n")
		b.WriteString(SummaryLabelStyle.Render("Visit https://falcon.solo.ro/settings#!/company and check Network tab for company_ID"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(ErrorStyle.Render("‼️  Could not load company info"))
		b.WriteString("\n\n")
	}

	// Summary box
	summaryContent := fmt.Sprintf(
		"%s %d\n%s %.2f %s\n%s %.2f %s",
		SummaryLabelStyle.Render("Year:"),
		m.summary.Year,
		SummaryLabelStyle.Render("Total Revenues:"),
		m.summary.TotalRevenues,
		m.summary.DisplayCurrency,
		SummaryLabelStyle.Render("Total Expenses:"),
		m.summary.TotalDeductibleExpenses,
		m.summary.DisplayCurrency,
	)

	if m.summary.HasTaxes {
		summaryContent += fmt.Sprintf("\n%s %.2f %s",
			SummaryLabelStyle.Render("Taxes:"),
			m.summary.Taxes,
			m.summary.DisplayCurrency,
		)
	}

	b.WriteString(SummaryBoxStyle.Render(summaryContent))

	// Show pending review info if any
	if m.queue != nil && len(m.queue.Items) > 0 {
		b.WriteString("\n\n")
		b.WriteString(InfoStyle().Render(fmt.Sprintf("ℹ️  %d documents pending review", len(m.queue.Items))))
	}

	return b.String()
}
