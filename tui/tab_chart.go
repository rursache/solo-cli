package tui

import (
	"fmt"
	"strconv"
	"strings"

	"solo-cli/taxes"
)

var monthLabels = [12]string{"Ian", "Feb", "Mar", "Apr", "Mai", "Iun", "Iul", "Aug", "Sep", "Oct", "Noi", "Dec"}

// revenueRON returns the invoice value in RON, preferring the local amount
// for foreign currency invoices
func revenueRON(total float64, localTotal *float64) float64 {
	if localTotal != nil {
		return *localTotal
	}
	return total
}

// monthlyRevenues aggregates the loaded invoices of the given year by issue
// month. IssueDate is ISO formatted so the year and month are a prefix
func (m Model) monthlyRevenues(year int) [12]float64 {
	var months [12]float64
	if m.revenues == nil {
		return months
	}
	prefix := strconv.Itoa(year) + "-"
	for _, r := range m.revenues.Items {
		if !strings.HasPrefix(r.IssueDate, prefix) || len(r.IssueDate) < 7 {
			continue
		}
		mo, err := strconv.Atoi(r.IssueDate[5:7])
		if err != nil || mo < 1 || mo > 12 {
			continue
		}
		var local *float64
		if r.InvoiceLocalAmount != nil {
			local = &r.InvoiceLocalAmount.Total
		}
		months[mo-1] += revenueRON(r.Total, local)
	}
	return months
}

func (m Model) renderChart() string {
	var b strings.Builder

	year := m.year
	if year == 0 && m.summary != nil {
		year = m.summary.Year
	}

	b.WriteString(TitleStyle.Render(fmt.Sprintf("Monthly Revenues (%d)", year)))
	b.WriteString("\n")

	loaded, available := m.chartCoverage()
	if loaded < available {
		b.WriteString(LoadingStyle.Render(fmt.Sprintf("Loading invoices... %d of %d", loaded, available)))
		b.WriteString("\n\n")
	}

	months := m.monthlyRevenues(year)
	maxVal := 0.0
	total := 0.0
	for _, v := range months {
		if v > maxVal {
			maxVal = v
		}
		total += v
	}

	if maxVal == 0 {
		b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("No invoices issued in %d", year)))
		return b.String()
	}

	// Layout: "Ian " (4) + bar + " " + amount (13). The bar flexes
	valueWidth := 13
	barWidth := m.fillWidth(4+1+valueWidth, 20)

	for i, v := range months {
		filled := 0
		if maxVal > 0 {
			filled = int(v / maxVal * float64(barWidth))
		}
		if v > 0 && filled == 0 {
			filled = 1 // Non-zero months always show something
		}

		bar := secondaryStyle.Render(strings.Repeat("█", filled)) + SummaryLabelStyle.Render(strings.Repeat("░", barWidth-filled))
		b.WriteString(fmt.Sprintf("%s %s %s\n",
			SummaryLabelStyle.Render(fmt.Sprintf("%-3s", monthLabels[i])),
			bar,
			SummaryValueStyle.Render(fmt.Sprintf("%*.2f", valueWidth-1, v)),
		))
	}

	b.WriteString("\n")
	b.WriteString(SummaryLabelStyle.Render("Total: "))
	b.WriteString(SummaryValueStyle.Render(taxes.FormatRON(total)))

	return b.String()
}

// chartCoverage reports how many invoices are loaded vs available
func (m Model) chartCoverage() (int, int) {
	if m.revenues == nil {
		return 0, 0
	}
	loaded := len(m.revenues.Items)
	if m.revenues.TotalResults != nil && *m.revenues.TotalResults > loaded {
		return loaded, *m.revenues.TotalResults
	}
	return loaded, loaded
}
