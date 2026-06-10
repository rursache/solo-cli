package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Render every tab of the demo model at a fixed terminal size and check that
// table rows fill the width without ever exceeding it
func TestTabsRenderWithinWidth(t *testing.T) {
	const width, height = 120, 40

	m := NewDemoModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m = updated.(Model)

	for tab := TabDashboard; tab < tabCount; tab++ {
		m.activeTab = tab
		t.Run(tab.String(), func(t *testing.T) {
			view := m.View()
			if view == "" {
				t.Fatal("empty view")
			}
			for _, line := range strings.Split(view, "\n") {
				if w := lipgloss.Width(line); w > width {
					t.Errorf("line exceeds width %d (got %d): %q", width, w, line)
				}
			}
		})
	}
}

// Fill columns must stretch rows to the full available width
func TestTableRowsFillWidth(t *testing.T) {
	const width = 100

	m := NewDemoModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 40})
	m = updated.(Model)

	for _, tab := range []Tab{TabRevenues, TabExpenses, TabEFactura, TabQueue} {
		m.activeTab = tab
		t.Run(tab.String(), func(t *testing.T) {
			view := m.View()
			// The selected row is padded to the fill width, so at least one
			// line must reach width-1
			maxW := 0
			for _, line := range strings.Split(view, "\n") {
				if w := lipgloss.Width(line); w > maxW {
					maxW = w
				}
			}
			if maxW < width-1 {
				t.Errorf("widest line is %d, want %d (fill column not stretching)", maxW, width-1)
			}
		})
	}
}

func TestPadTruncate(t *testing.T) {
	if got := padTruncate("abc", 6); got != "abc   " {
		t.Errorf("pad: %q", got)
	}
	if got := padTruncate("abcdefghij", 6); got != "abc..." {
		t.Errorf("truncate: %q", got)
	}
	// Diacritics must count as one cell each, not bytes
	if got := padTruncate("PLĂMĂDEALĂ", 12); lipgloss.Width(got) != 12 {
		t.Errorf("diacritics width = %d, want 12 (%q)", lipgloss.Width(got), got)
	}
}
