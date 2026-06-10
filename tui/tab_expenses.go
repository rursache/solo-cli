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
		b.WriteString(m.emptyList("No expenses found"))
		return b.String()
	}

	// Fixed columns: Amount(12) + Curr(4) + Category(30) + separators
	supplierWidth := m.fillWidth(49, 20)
	header := fmt.Sprintf("%-12s %-4s %-30s %s", "Amount", "Curr", "Category", padTruncate("Supplier", supplierWidth))

	b.WriteString(m.renderList(len(m.expenses.Items), header, func(i int) string {
		e := m.expenses.Items[i]
		focused := i == m.cursor
		return fmt.Sprintf("%-12.2f %-4s %-30s %s",
			e.Total,
			strings.ToUpper(e.Currency.ShortName),
			m.cell(e.Category, 30, focused),
			m.cell(e.SupplierName, supplierWidth, focused),
		)
	}))

	return b.String()
}
