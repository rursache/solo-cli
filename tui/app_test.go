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

		// bodyHeight (height - 7) minus the list chrome (showing + header)
		want := height - 11
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

func TestMouseNavigation(t *testing.T) {
	m := NewDemoModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	// Click on the Expenses tab label in the tab bar
	x := 0
	for _, tab := range tabOrder {
		w := lipgloss.Width(InactiveTabStyle.Render(tab.String()))
		if tab == m.activeTab {
			w = lipgloss.Width(ActiveTabStyle.Render(tab.String()))
		}
		if tab == TabExpenses {
			break
		}
		x += w
	}
	updated, _ = m.Update(tea.MouseMsg{X: x + 2, Y: tabsRowY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(Model)
	if m.activeTab != TabExpenses {
		t.Fatalf("activeTab = %s, want Expenses after tab click", m.activeTab)
	}

	// Click on the third visible row (expenses demo has a rejected block)
	rowStart := listRowsStartY
	if m.rejected != nil && len(m.rejected.Items) > 0 {
		rowStart += len(m.rejected.Items) + 2
	}
	updated, _ = m.Update(tea.MouseMsg{X: 5, Y: rowStart + 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2 after row click", m.cursor)
	}

	// Click far below the list must not move the cursor
	updated, _ = m.Update(tea.MouseMsg{X: 5, Y: 39, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want unchanged 2 after dead-space click", m.cursor)
	}

	// Wheel scrolls the cursor
	updated, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	m = updated.(Model)
	if m.cursor != 3 {
		t.Errorf("cursor = %d, want 3 after wheel down", m.cursor)
	}
	updated, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2 after wheel up", m.cursor)
	}
}

func TestMarquee(t *testing.T) {
	// Fits: plain padding, no animation regardless of offset
	if got := marquee("abc", 5, 99); got != "abc  " {
		t.Errorf("fitting string = %q, want padded", got)
	}

	long := "abcdefghij" // 10 runes, window 6, gap 3 -> cycle 13
	// During the hold the window stays at the start
	if got := marquee(long, 6, 0); got != "abcdef" {
		t.Errorf("offset 0 = %q, want %q", got, "abcdef")
	}
	if got := marquee(long, 6, marqueeHoldTicks); got != "abcdef" {
		t.Errorf("offset at hold end = %q, want still %q", got, "abcdef")
	}
	// One tick past the hold slides one rune
	if got := marquee(long, 6, marqueeHoldTicks+1); got != "bcdefg" {
		t.Errorf("first slide = %q, want %q", got, "bcdefg")
	}
	// Window wraps around through the gap back to the start
	if got := marquee(long, 6, marqueeHoldTicks+13); got != "abcdef" {
		t.Errorf("full cycle = %q, want %q", got, "abcdef")
	}
	// Output width is stable at every offset
	for off := 0; off < 30; off++ {
		if w := len([]rune(marquee(long, 6, off))); w != 6 {
			t.Fatalf("offset %d: width %d, want 6", off, w)
		}
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
