package tui

import (
	"fmt"
	"strings"
)

func (m Model) renderExpenses() string {
	var b strings.Builder

	// Show rejected expenses warning if any
	if m.rejected != nil && len(m.rejected.Items) > 0 {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("⚠️  %d rejected expense(s):", len(m.rejected.Items))))
		b.WriteString("\n")
		for _, r := range m.rejected.Items {
			docName := truncate(r.DocumentName, 30)
			reason := truncate(r.Reason, 50)
			b.WriteString(WarningStyle().Render(fmt.Sprintf("   • %s - %s", docName, reason)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if m.expenses == nil || len(m.expenses.Items) == 0 {
		b.WriteString("No expenses found")
		return b.String()
	}

	total := len(m.expenses.Items)

	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Showing %d-%d of %d", m.viewportOffset+1, min(m.viewportOffset+m.tabViewportSize(), total), total)))
	b.WriteString("\n\n")

	// Fixed columns: Amount(12) + Curr(4) + Category(30) + separators
	supplierWidth := m.fillWidth(49, 20)

	b.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%-12s %-4s %-30s %s", "Amount", "Curr", "Category", padTruncate("Supplier", supplierWidth))))
	b.WriteString("\n")

	endIdx := min(m.viewportOffset+m.tabViewportSize(), total)

	for i := m.viewportOffset; i < endIdx; i++ {
		e := m.expenses.Items[i]

		row := fmt.Sprintf("%-12.2f %-4s %-30s %s",
			e.Total,
			strings.ToUpper(e.Currency.ShortName),
			truncate(e.Category, 30),
			padTruncate(e.SupplierName, supplierWidth),
		)

		if i == m.cursor {
			b.WriteString(TableSelectedStyle.Render(row))
		} else {
			b.WriteString(TableRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	return b.String()
}
