package tui

import (
	"fmt"
	"strings"
)

func (m Model) renderQueue() string {
	if m.queue == nil || len(m.queue.Items) == 0 {
		return "No documents in queue"
	}

	var b strings.Builder
	total := len(m.queue.Items)

	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Showing %d-%d of %d", m.viewportOffset+1, min(m.viewportOffset+m.tabViewportSize(), total), total)))
	b.WriteString("\n\n")

	// Fixed columns: Days(5) + Status(8) + separators. Document goes last so
	// the long filename is the column that fills the remaining width
	documentWidth := m.fillWidth(15, 20)

	b.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%5s %-8s %s", "Days", "Status", padTruncate("Document", documentWidth))))
	b.WriteString("\n")

	endIdx := min(m.viewportOffset+m.tabViewportSize(), total)

	for i := m.viewportOffset; i < endIdx; i++ {
		q := m.queue.Items[i]
		status := "Pending"
		if q.IsOverdue {
			status = "OVERDUE"
		}

		row := fmt.Sprintf("%5d %-8s %s",
			q.DaysPassed,
			status,
			padTruncate(q.DocumentName, documentWidth),
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
