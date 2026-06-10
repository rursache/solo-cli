package tui

import "fmt"

func (m Model) renderQueue() string {
	if m.queue == nil || len(m.queue.Items) == 0 {
		return "No documents in queue"
	}

	// Fixed columns: Days(5) + Status(8) + separators. Document goes last so
	// the long filename is the column that fills the remaining width
	documentWidth := m.fillWidth(15, 20)
	header := fmt.Sprintf("%5s %-8s %s", "Days", "Status", padTruncate("Document", documentWidth))

	return m.renderList(len(m.queue.Items), header, func(i int) string {
		q := m.queue.Items[i]
		status := "Pending"
		if q.IsOverdue {
			status = "OVERDUE"
		}
		return fmt.Sprintf("%5d %-8s %s",
			q.DaysPassed,
			status,
			m.cell(q.DocumentName, documentWidth, i == m.cursor),
		)
	})
}
