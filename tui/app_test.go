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

// The list viewport must grow and shrink with the terminal height and the
// rendered view must never exceed it
func TestViewportAdaptsToHeight(t *testing.T) {
	// 24 is the practical minimum: the Dashboard's fixed content (company
	// header + summary box) needs ~22 rows before padding
	for _, height := range []int{24, 30, 50} {
		m := NewDemoModel()
		updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: height})
		m = updated.(Model)

		want := height - 13
		if want < 3 {
			want = 3
		}
		if m.viewportSize != want {
			t.Errorf("height %d: viewportSize = %d, want %d", height, m.viewportSize, want)
		}

		for tab := TabDashboard; tab < tabCount; tab++ {
			m.activeTab = tab
			view := m.View()
			lines := strings.Split(view, "\n")
			if len(lines) > height {
				t.Errorf("height %d, tab %s: view has %d lines, must fit %d", height, tab, len(lines), height)
			}
			// Help must be pinned to the very last row
			if last := lines[len(lines)-1]; !strings.Contains(last, "quit") {
				t.Errorf("height %d, tab %s: last line is not the help bar: %q", height, tab, last)
			}
			if len(lines) != height {
				t.Errorf("height %d, tab %s: view has %d lines, help not pinned to bottom", height, tab, len(lines))
			}
		}
	}
}

// The Expenses tab gives up rows to the rejected warning block
func TestExpensesViewportShrinksForRejected(t *testing.T) {
	m := NewDemoModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)

	m.activeTab = TabExpenses
	if m.rejected == nil || len(m.rejected.Items) == 0 {
		t.Skip("demo data has no rejected expenses")
	}
	want := m.viewportSize - len(m.rejected.Items) - 2
	if got := m.tabViewportSize(); got != want {
		t.Errorf("tabViewportSize = %d, want %d", got, want)
	}
}

// When the Taxes content is taller than the screen, the scroll viewport must
// use all available rows: no dead gap between the scroll hint and the help bar
func TestTaxesViewportUsesFullHeight(t *testing.T) {
	const height = 24

	m := NewDemoModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: height})
	m = updated.(Model)
	m.activeTab = TabTaxes

	lines := strings.Split(m.View(), "\n")
	hintIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "scroll to see more") {
			hintIdx = i
		}
	}
	if hintIdx == -1 {
		t.Fatal("taxes content not scrollable at height 24, cannot verify gap")
	}
	// Expected tail: hint, padding row, help margin row, help text
	if gap := len(lines) - 1 - hintIdx; gap > 3 {
		t.Errorf("%d rows between scroll hint and help bar, want at most 3 (dead space)", gap)
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
