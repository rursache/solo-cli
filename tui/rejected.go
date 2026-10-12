package tui

import (
	"fmt"
	"strings"

	"solo-cli/client"
)

// maxRejectedShown caps the rejected warning block so a long rejected list
// cannot push the table off small terminals
const maxRejectedShown = 3

// rejectedDocs returns the rejected documents warned about above the active
// list tab and the noun used in the heading
func (m Model) rejectedDocs() (string, []client.RejectedExpense) {
	switch m.activeTab {
	case TabExpenses:
		if m.rejected != nil {
			return "expense", m.rejected.Items
		}
	case TabRevenues:
		if m.rejectedRevenues != nil {
			docs := make([]client.RejectedExpense, len(m.rejectedRevenues.Items))
			for i, r := range m.rejectedRevenues.Items {
				docs[i] = client.RejectedExpense(r)
			}
			return "revenue document", docs
		}
	}
	return "", nil
}

// rejectedBlockHeight is how many rows the rejected warning block takes
// above the list: heading, the shown documents, an overflow line and a
// blank. The viewport and click math both derive from it
func (m Model) rejectedBlockHeight() int {
	_, docs := m.rejectedDocs()
	if len(docs) == 0 {
		return 0
	}
	h := min(len(docs), maxRejectedShown) + 2
	if len(docs) > maxRejectedShown {
		h++
	}
	return h
}

// renderRejectedBlock renders the rejected warning block, "" when the active
// tab has no rejected documents. Its line count must match rejectedBlockHeight
func (m Model) renderRejectedBlock() string {
	noun, docs := m.rejectedDocs()
	if len(docs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(ErrorStyle.Render(fmt.Sprintf("⚠️  %d rejected %s(s):", len(docs), noun)))
	b.WriteString("\n")
	for _, r := range docs[:min(len(docs), maxRejectedShown)] {
		line := fmt.Sprintf("   • %s - %s", truncate(r.DocumentName, 30), truncate(r.Reason, 50))
		b.WriteString(WarningStyle().Render(truncate(line, m.fillWidth(0, 20))))
		b.WriteString("\n")
	}
	if extra := len(docs) - maxRejectedShown; extra > 0 {
		b.WriteString(WarningStyle().Render(fmt.Sprintf("   • and %d more", extra)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}
