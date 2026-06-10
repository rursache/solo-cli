package tui

import (
	"fmt"
	"strings"
)

func (m Model) renderEFactura() string {
	if m.efactura == nil || len(m.efactura.Items) == 0 {
		return "No e-Factura documents found"
	}

	var b strings.Builder
	total := len(m.efactura.Items)

	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Showing %d-%d of %d", m.viewportOffset+1, min(m.viewportOffset+m.tabViewportSize(), total), total)))
	b.WriteString("\n\n")

	// Fixed columns: Serial(20) + Amount(12) + Curr(4) + Date(12) + separators
	partyWidth := m.fillWidth(52, 20)

	b.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%-20s %12s %-4s %-12s %s", "Serial", "Amount", "Curr", "Date", padTruncate("Party", partyWidth))))
	b.WriteString("\n")

	endIdx := min(m.viewportOffset+m.tabViewportSize(), total)

	for i := m.viewportOffset; i < endIdx; i++ {
		e := m.efactura.Items[i]
		row := fmt.Sprintf("%-20s %12.2f %-4s %-12s %s",
			truncate(e.SerialCode, 20),
			e.TotalAmount,
			strings.ToUpper(e.CurrencyCode),
			e.InvoiceDate,
			padTruncate(e.PartyName, partyWidth),
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
