package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"solo-cli/client"

	"github.com/charmbracelet/lipgloss"
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
		b.WriteString("\n")
		if m.company.Address != "" {
			b.WriteString(SummaryLabelStyle.Render(truncate(m.company.Address, m.fillWidth(0, 40))))
			b.WriteString("\n")
		}
		if caen := m.renderCAENLines(m.fillWidth(0, 40)); caen != "" {
			b.WriteString(caen)
		}
		b.WriteString("\n")
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
		"%s %s\n%s %.2f %s\n%s %.2f %s\n%s %.2f %s",
		SummaryLabelStyle.Render("Year:"),
		m.renderYearRow(),
		SummaryLabelStyle.Render("Total Revenues:"),
		m.summary.TotalRevenues,
		m.summary.DisplayCurrency,
		SummaryLabelStyle.Render("Total Expenses:"),
		m.summary.TotalDeductibleExpenses,
		m.summary.DisplayCurrency,
		SummaryLabelStyle.Render("Net Income:"),
		m.summary.TotalRevenues-m.summary.TotalDeductibleExpenses,
		m.summary.DisplayCurrency,
	)

	if m.summary.HasTaxes {
		summaryContent += fmt.Sprintf("\n%s %.2f %s",
			SummaryLabelStyle.Render("Taxes:"),
			m.summary.Taxes,
			m.summary.DisplayCurrency,
		)
	}

	box := SummaryBoxStyle.Render(summaryContent)

	// Show pending review info if any
	var pending string
	if m.queue != nil && len(m.queue.Items) > 0 {
		pending = "\n\n" + InfoStyle().Render(fmt.Sprintf("ℹ️  %d documents pending review", len(m.queue.Items)))
	}

	// Notices sit right of the summary box when there is room. Narrow
	// terminals get them below it, only if that still fits the body
	if avail := m.width - lipgloss.Width(box) - 2; avail >= minNoticesWidth {
		if notices := m.renderNotices(min(avail, maxNoticesWidth) - noticesChrome); notices != "" {
			box = lipgloss.JoinHorizontal(lipgloss.Top, box, "  ", NoticesBoxStyle.Render(notices))
		}
	} else if notices := m.renderNotices(min(m.width-1, maxNoticesWidth)); notices != "" {
		stacked := b.String() + box + "\n" + notices + pending
		if m.height == 0 || lipgloss.Height(stacked) <= m.bodyHeight() {
			box += "\n" + notices
		}
	}

	b.WriteString(box)
	b.WriteString(pending)
	return b.String()
}

// yearOptions is how many years the dashboard offers, newest first
const yearOptions = 4

// renderYearRow shows the selectable years with the displayed one
// highlighted. The years are clickable (see clickYear in mouse.go)
func (m Model) renderYearRow() string {
	if m.maxYear == 0 {
		return SummaryValueStyle.Render(fmt.Sprintf("%d", m.summary.Year))
	}

	var parts []string
	for y := m.maxYear; y > m.maxYear-yearOptions; y-- {
		s := fmt.Sprintf("%d", y)
		if y == m.summary.Year {
			parts = append(parts, SummaryValueStyle.Render(s))
		} else {
			parts = append(parts, SummaryLabelStyle.Render(s))
		}
	}
	return strings.Join(parts, SummaryLabelStyle.Render(" • "))
}

// renderCAENLines shows the principal CAEN code with its full name and the
// secondary ones as a compact code list. The principal line marquees when
// it overflows since the full activity name matters
func (m Model) renderCAENLines(width int) string {
	if len(m.caenCodes) == 0 {
		return ""
	}

	var primary string
	var secondary []string
	for _, c := range m.caenCodes {
		if c.IsPrimary {
			primary = fmt.Sprintf("%s - %s", c.Code, c.Name)
		} else {
			secondary = append(secondary, c.Code)
		}
	}

	var b strings.Builder
	if primary != "" {
		// The label stays static, only the value marquees
		const label = "CAEN principal: "
		valueWidth := width - len(label)
		if valueWidth < 10 {
			valueWidth = 10
		}
		b.WriteString(SummaryLabelStyle.Render(label + marquee(primary, valueWidth, m.marqueeOffset)))
		b.WriteString("\n")
	}
	if len(secondary) > 0 {
		b.WriteString(SummaryLabelStyle.Render(truncate("CAEN secundare: "+strings.Join(secondary, ", "), width)))
		b.WriteString("\n")
	}
	return b.String()
}

// Notices panel width bounds, the chrome is its border and padding
const (
	minNoticesWidth = 36
	maxNoticesWidth = 72
	noticesChrome   = 6
)

// renderNotices shows the Declaratia Unica status of the last fiscal year
// and the active alerts, one line each truncated to width. "" when there
// is nothing to show
func (m Model) renderNotices(width int) string {
	if width < 20 {
		return ""
	}
	var lines []string
	if d := m.declaration; d != nil && d.DeclarationStatus != "" {
		lines = append(lines,
			SummaryValueStyle.Render(truncate(fmt.Sprintf("Declaratia Unica %d", d.FiscalYear), width)),
			declarationStyle(d).Render(truncate(d.DeclarationStatus.Label(), width)),
			SummaryLabelStyle.Render(truncate("Deadline: "+client.DeadlineText(d.Deadline(), time.Now()), width)),
		)
	}
	if len(m.alerts) > 0 && len(lines) > 0 {
		lines = append(lines, "")
	}
	for _, a := range m.alerts {
		lines = append(lines, alertLine(a, width))
	}
	return strings.Join(lines, "\n")
}

// declarationStyle is green once filed, red when the filing failed or the
// deadline passed without it and amber while still pending
func declarationStyle(d *client.TaxDeclarationDashboard) lipgloss.Style {
	switch {
	case d.DeclarationStatus.Filed():
		return PaidStyle
	case d.DeclarationStatus.NeedsAttention(), time.Now().After(d.Deadline().AddDate(0, 0, 1)):
		return dangerStyle
	}
	return warningStyle
}

// alertLine renders "● Title - message" with the title colored by severity
// and the message muted, truncated as a whole to width
func alertLine(a client.Alert, width int) string {
	style := InfoStyle()
	switch a.Severity {
	case client.AlertCritical:
		style = dangerStyle
	case client.AlertWarning:
		style = warningStyle
	}
	title := truncate("● "+a.Title, width)
	rest := width - utf8.RuneCountInString(title)
	if a.Message == "" || rest < 8 {
		return style.Render(title)
	}
	return style.Render(title) + SummaryLabelStyle.Render(truncate(" - "+a.Message, rest))
}
