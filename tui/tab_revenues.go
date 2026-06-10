package tui

import (
	"fmt"
	"strings"
)

func (m Model) renderRevenues() string {
	if m.revenues == nil || len(m.revenues.Items) == 0 {
		return "No revenues found"
	}

	var b strings.Builder
	total := len(m.revenues.Items)

	// Show scroll position indicator
	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Showing %d-%d of %d", m.viewportOffset+1, min(m.viewportOffset+m.tabViewportSize(), total), total)))
	b.WriteString("\n\n")

	// Fixed columns: Invoice(18) + Amount(12) + Curr(4) + separators
	clientWidth := m.fillWidth(37, 20)

	b.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%-18s %12s %-4s %s", "Invoice", "Amount", "Curr", padTruncate("Client", clientWidth))))
	b.WriteString("\n")

	// Calculate visible range
	endIdx := min(m.viewportOffset+m.tabViewportSize(), total)

	for i := m.viewportOffset; i < endIdx; i++ {
		r := m.revenues.Items[i]

		row := fmt.Sprintf("%-18s %12.2f %-4s %s",
			truncate(r.SerialCode, 18),
			r.Total,
			strings.ToUpper(r.Currency.ShortName),
			padTruncate(r.ClientName, clientWidth),
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
