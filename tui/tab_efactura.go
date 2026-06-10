package tui

import (
	"fmt"
	"strings"
)

func (m Model) renderEFactura() string {
	if m.efactura == nil || len(m.efactura.Items) == 0 {
		return "No e-Factura documents found"
	}

	// Fixed columns: Serial(20) + Amount(12) + Curr(4) + Date(12) + separators
	partyWidth := m.fillWidth(52, 20)
	header := fmt.Sprintf("%-20s %12s %-4s %-12s %s", "Serial", "Amount", "Curr", "Date", padTruncate("Party", partyWidth))

	return m.renderList(len(m.efactura.Items), header, func(i int) string {
		e := m.efactura.Items[i]
		return fmt.Sprintf("%-20s %12.2f %-4s %-12s %s",
			truncate(e.SerialCode, 20),
			e.TotalAmount,
			strings.ToUpper(e.CurrencyCode),
			e.InvoiceDate,
			padTruncate(e.PartyName, partyWidth),
		)
	})
}
