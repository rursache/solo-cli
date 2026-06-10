package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// dateOnly trims an ISO timestamp to its date part
func dateOnly(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

type detailField struct {
	label string
	value string
}

// detailFields returns the full field list for the selected row of the
// active list tab, nil when there is nothing to show
func (m Model) detailFields() (string, []detailField) {
	i := m.cursor
	switch m.activeTab {
	case TabRevenues:
		if m.revenues == nil || i >= len(m.revenues.Items) {
			return "", nil
		}
		r := m.revenues.Items[i]
		paid := "no"
		if r.IsPaid {
			paid = "yes (" + dateOnly(r.PaymentDate) + ")"
		}
		fields := []detailField{
			{"Invoice", r.SerialCode},
			{"Client", r.ClientName},
			{"Issued", dateOnly(r.IssueDate)},
			{"Paid", paid},
			{"Total", fmt.Sprintf("%.2f %s", r.Total, strings.ToUpper(r.Currency.ShortName))},
		}
		if r.InvoiceLocalAmount != nil {
			la := r.InvoiceLocalAmount
			fields = append(fields,
				detailField{"Local amount", fmt.Sprintf("%.2f %s", la.Amount, strings.ToUpper(la.Currency.ShortName))},
				detailField{"Local VAT", fmt.Sprintf("%.2f", la.VAT)},
				detailField{"Local total", fmt.Sprintf("%.2f", la.Total)},
			)
		}
		if r.Status != nil {
			fields = append(fields, detailField{"Status", r.Status.Name})
		}
		if r.EInvoiceStatus != nil {
			fields = append(fields, detailField{"e-Factura", r.EInvoiceStatus.Name})
		}
		return "Invoice " + r.SerialCode, fields

	case TabExpenses:
		if m.expenses == nil || i >= len(m.expenses.Items) {
			return "", nil
		}
		e := m.expenses.Items[i]
		fields := []detailField{
			{"Supplier", e.SupplierName},
			{"Date", dateOnly(e.PurchaseDate)},
			{"Category", e.Category},
			{"Primary category", e.PrimaryCategory},
			{"Deductibility", e.Deductibility},
			{"Total", fmt.Sprintf("%.2f %s", e.Total, strings.ToUpper(e.Currency.ShortName))},
		}
		if e.ExpenseLocalAmount != nil {
			fields = append(fields, detailField{"Local total", fmt.Sprintf("%.2f %s", e.ExpenseLocalAmount.Total, strings.ToUpper(e.ExpenseLocalAmount.Currency.ShortName))})
		}
		return "Expense", fields

	case TabEFactura:
		if m.efactura == nil || i >= len(m.efactura.Items) {
			return "", nil
		}
		e := m.efactura.Items[i]
		return "e-Factura " + e.SerialCode, []detailField{
			{"Serial", e.SerialCode},
			{"Party", e.PartyName},
			{"Party CUI", e.PartyCode1},
			{"Date", dateOnly(e.InvoiceDate)},
			{"Amount", fmt.Sprintf("%.2f %s", e.TotalAmount, strings.ToUpper(e.CurrencyCode))},
		}

	case TabQueue:
		if m.queue == nil || i >= len(m.queue.Items) {
			return "", nil
		}
		q := m.queue.Items[i]
		overdue := "no"
		if q.IsOverdue {
			overdue = "YES"
		}
		return "Queued document", []detailField{
			{"Document", q.DocumentName},
			{"Code", q.DocumentCode},
			{"Type", q.DocumentMimeType},
			{"Uploaded", dateOnly(q.CreatedOn)},
			{"Days in queue", fmt.Sprintf("%d", q.DaysPassed)},
			{"Deadline", dateOnly(q.ProcessingDeadline)},
			{"Overdue", overdue},
		}
	}
	return "", nil
}

// renderDetail shows the selected row as a centered modal box
func (m Model) renderDetail() string {
	title, fields := m.detailFields()
	if fields == nil {
		return "Nothing selected"
	}

	labelWidth := 0
	for _, f := range fields {
		if len(f.label) > labelWidth {
			labelWidth = len(f.label)
		}
	}

	var b strings.Builder
	b.WriteString(AppTitleStyle.Render(title))
	b.WriteString("\n\n")
	for _, f := range fields {
		b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("%-*s ", labelWidth+1, f.label+":")))
		b.WriteString(SummaryValueStyle.Render(f.value))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(SummaryLabelStyle.Render("↑/↓ browse items • esc close"))

	box := SummaryBoxStyle.Render(b.String())
	return lipgloss.Place(m.width-1, m.bodyHeight(), lipgloss.Center, lipgloss.Center, box)
}
