package tui

import "fmt"

func (m Model) renderQueue() string {
	if m.queue == nil || len(m.queue.Items) == 0 {
		return m.emptyList("No documents in queue")
	}

	// Fixed columns: Days(4) + separator. Everything here is pending by
	// definition so there is no status column. Document fills the width
	documentWidth := m.fillWidth(5, 20)
	header := fmt.Sprintf("%-4s %s", "Days", padTruncate("Document", documentWidth))

	return m.renderList(len(m.queue.Items), header, func(i int) string {
		q := m.queue.Items[i]
		return fmt.Sprintf("%-4d %s",
			q.DaysPassed,
			m.cell(q.DocumentName, documentWidth, i == m.cursor),
		)
	})
}
