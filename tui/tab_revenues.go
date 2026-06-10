package tui

import (
	"fmt"
	"strings"
)

func (m Model) renderRevenues() string {
	if m.revenues == nil || len(m.revenues.Items) == 0 {
		return "No revenues found"
	}

	// Fixed columns: Invoice(18) + Amount(12) + Curr(4) + separators
	clientWidth := m.fillWidth(37, 20)
	header := fmt.Sprintf("%-18s %12s %-4s %s", "Invoice", "Amount", "Curr", padTruncate("Client", clientWidth))

	return m.renderList(len(m.revenues.Items), header, func(i int) string {
		r := m.revenues.Items[i]
		focused := i == m.cursor
		return fmt.Sprintf("%-18s %12.2f %-4s %s",
			m.cell(r.SerialCode, 18, focused),
			r.Total,
			strings.ToUpper(r.Currency.ShortName),
			m.cell(r.ClientName, clientWidth, focused),
		)
	})
}
