package tui

import (
	"fmt"
	"strings"
	"testing"

	"solo-cli/client"

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
	// 28 is the practical minimum: the Dashboard's fixed content (company
	// header with address and CAEN codes + summary box) needs ~26 rows
	// before padding
	for _, height := range []int{28, 35, 50} {
		m := NewDemoModel()
		updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: height})
		m = updated.(Model)

		// bodyHeight (height - 7) minus the list chrome (combined
		// search/showing line and table header)
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

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// The [ and ] keys switch the displayed year on Dashboard and Taxes,
// bounded by the current fiscal year, and trigger a summary refetch
func TestYearSwitcher(t *testing.T) {
	m := NewDemoModel()
	m.demoMode = false // demo mode has no API to refetch from
	m.year, m.maxYear = 2026, 2026
	m.activeTab = TabDashboard

	updated, cmd := m.Update(keyMsg("["))
	m = updated.(Model)
	if m.year != 2025 {
		t.Errorf("year after [ = %d, want 2025", m.year)
	}
	if cmd == nil {
		t.Error("[ must trigger a summary refetch")
	}

	updated, cmd = m.Update(keyMsg("]"))
	m = updated.(Model)
	if m.year != 2026 {
		t.Errorf("year after ] = %d, want 2026", m.year)
	}

	// Cannot go past the current fiscal year
	updated, cmd = m.Update(keyMsg("]"))
	m = updated.(Model)
	if m.year != 2026 || cmd != nil {
		t.Errorf("year went past maxYear: %d (cmd %v)", m.year, cmd)
	}

	// Ignored on list tabs
	m.activeTab = TabRevenues
	updated, cmd = m.Update(keyMsg("["))
	m = updated.(Model)
	if m.year != 2026 || cmd != nil {
		t.Errorf("year switch must be ignored on list tabs: %d", m.year)
	}

	// Ignored in demo mode
	m.activeTab = TabDashboard
	m.demoMode = true
	updated, _ = m.Update(keyMsg("["))
	m = updated.(Model)
	if m.year != 2026 {
		t.Errorf("year switch must be ignored in demo mode: %d", m.year)
	}
}

// Clicking a year in the dashboard summary box switches to it
func TestClickableYears(t *testing.T) {
	m := NewDemoModel()
	m.demoMode = false
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	m.activeTab = TabDashboard
	m.year, m.maxYear = 2026, 2026
	m.summary.Year = 2026

	// The year row must offer the current and previous years
	view := m.View()
	for _, yr := range []string{"2026", "2025", "2024", "2023"} {
		if !strings.Contains(view, yr) {
			t.Fatalf("year row missing %s", yr)
		}
	}

	// Locate 2024 in the rendered view the same way clickYear does
	lines := strings.Split(view, "\n")
	clickX, clickY := -1, -1
	for i, line := range lines {
		plain := stripANSI(line)
		if strings.Contains(plain, "Year:") {
			idx := strings.Index(plain, "2024")
			clickX = len([]rune(plain[:idx])) + 1
			clickY = i
			break
		}
	}
	if clickY == -1 {
		t.Fatal("year row not found in view")
	}

	updated, cmd := m.Update(tea.MouseMsg{X: clickX, Y: clickY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(Model)
	if m.year != 2024 {
		t.Fatalf("year after click = %d, want 2024", m.year)
	}
	if cmd == nil {
		t.Fatal("year click must refetch the summary")
	}

	// Clicking the already selected year does nothing
	m.summary.Year = 2024
	updated, cmd = m.Update(tea.MouseMsg{X: clickX, Y: clickY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(Model)
	if m.year != 2024 || cmd != nil {
		t.Error("clicking the selected year must be a no-op")
	}
}

// The complete [ key flow: first summary establishes the year, the key
// switches it and the refetched summary lands
func TestYearSwitcherFullFlow(t *testing.T) {
	m := NewDemoModel()
	m.demoMode = false
	m.activeTab = TabDashboard
	m.year, m.maxYear = 0, 0

	updated, _ := m.Update(summaryMsg(&client.Summary{Year: 2026, TotalRevenues: 1000}))
	m = updated.(Model)

	updated, cmd := m.Update(keyMsg("["))
	m = updated.(Model)
	if m.year != 2025 || cmd == nil {
		t.Fatalf("[ after first summary: year %d, cmd %v", m.year, cmd)
	}

	// The refetched summary for a year without data renders zeros
	updated, _ = m.Update(summaryMsg(&client.Summary{Year: 2025}))
	m = updated.(Model)
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	if view := m.View(); !strings.Contains(view, "0.00") {
		t.Error("no-data year must render zero totals")
	}
}

func stripANSI(s string) string {
	var out []rune
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEscape = true
		case inEscape && (r == 'm'):
			inEscape = false
		case !inEscape:
			out = append(out, r)
		}
	}
	return string(out)
}

// The first summary establishes maxYear and later summaries track the year
func TestSummarySetsYearBounds(t *testing.T) {
	m := NewDemoModel()
	m.summary = nil
	m.year, m.maxYear = 0, 0

	updated, _ := m.Update(summaryMsg(&client.Summary{Year: 2026, TotalRevenues: 1000}))
	m = updated.(Model)
	if m.year != 2026 || m.maxYear != 2026 {
		t.Fatalf("after first summary: year %d maxYear %d, want 2026/2026", m.year, m.maxYear)
	}

	updated, _ = m.Update(summaryMsg(&client.Summary{Year: 2024, TotalRevenues: 500}))
	m = updated.(Model)
	if m.year != 2024 || m.maxYear != 2026 {
		t.Errorf("after year switch: year %d maxYear %d, want 2024/2026", m.year, m.maxYear)
	}
}

// Scrolling near the end of the loaded items must trigger a next-page
// fetch when the server reports more, and the page must append
func TestListPaging(t *testing.T) {
	m := NewDemoModel()
	m.demoMode = false
	m.activeTab = TabRevenues
	// Height 18 -> viewport 7, so the prefetch threshold for 20 loaded
	// items sits at cursor 13
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 18})
	m = updated.(Model)

	// 20 loaded items, server says 50 exist
	total := 50
	m.revenues.Items = m.revenues.Items[:1]
	for len(m.revenues.Items) < 20 {
		m.revenues.Items = append(m.revenues.Items, m.revenues.Items[0])
	}
	m.revenues.TotalResults = &total

	// The combined line must show the real total right aligned alongside
	// the search bar
	view := m.View()
	if !strings.Contains(view, "of 50") {
		t.Error("status line does not show the server-reported total")
	}
	for _, line := range strings.Split(view, "\n") {
		plain := stripANSI(line)
		if strings.Contains(plain, "Search:") {
			if !strings.Contains(plain, "of 50") {
				t.Error("search bar and result counter are not on the same line")
			}
			break
		}
	}

	// Scroll down: no fetch while far from the end
	updated, cmd := m.Update(keyMsg("j"))
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("fetch triggered too early")
	}

	// Jump close to the end: within one viewport of item 20
	m.cursor = 12
	updated, cmd = m.Update(keyMsg("j"))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("no next-page fetch near the end of loaded items")
	}
	if !m.fetchingMore {
		t.Fatal("fetchingMore not set")
	}

	// No duplicate fetch while one is in flight
	updated, cmd = m.Update(keyMsg("k"))
	m = updated.(Model)
	updated, cmd = m.Update(keyMsg("j"))
	m = updated.(Model)
	if cmd != nil {
		t.Error("duplicate page fetch while one is in flight")
	}

	// The page appends instead of replacing
	page := &client.RevenueListResponse{
		Items:        []client.Revenue{{SerialCode: "PAGE-2", ClientName: "Second Page"}},
		TotalResults: &total,
	}
	updated, _ = m.Update(revenuesPageMsg(page))
	m = updated.(Model)
	if len(m.revenues.Items) != 21 {
		t.Errorf("items after page = %d, want 21 (appended)", len(m.revenues.Items))
	}
	if m.fetchingMore {
		t.Error("fetchingMore not cleared after the page arrived")
	}
	if m.revenues.Items[20].SerialCode != "PAGE-2" {
		t.Errorf("page items not appended at the end: %+v", m.revenues.Items[20])
	}
}

// The / search captures typing (q must not quit), filters live with a
// debounce, applies on enter and clears on esc
func TestSearchFlow(t *testing.T) {
	m := NewDemoModel()
	m.demoMode = false
	m.activeTab = TabRevenues

	// The search bar is always visible, even when idle
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)
	if view := m.View(); !strings.Contains(view, "Search: ") {
		t.Fatal("search bar not visible when idle")
	}

	updated, _ = m.Update(keyMsg("/"))
	m = updated.(Model)
	if !m.searching {
		t.Fatal("/ must enter search mode")
	}

	// Typing accumulates (hotkey letters included) and schedules a debounce
	for _, ch := range []string{"a", "q", "h"} {
		updated, cmd := m.Update(keyMsg(ch))
		m = updated.(Model)
		if cmd == nil {
			t.Fatalf("typing %q must schedule a debounce tick", ch)
		}
	}
	if m.searchInput != "aqh" {
		t.Fatalf("searchInput = %q, want aqh", m.searchInput)
	}
	if m.activeTab != TabRevenues {
		t.Fatal("typing h in search mode must not switch tabs")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)
	if m.searchInput != "aq" {
		t.Errorf("after backspace: %q, want aq", m.searchInput)
	}
	if view := m.View(); !strings.Contains(view, "aq█") {
		t.Error("search input not rendered")
	}

	// A stale debounce tick (typed again since) must not apply
	updated, cmd := m.Update(searchDebounceMsg{seq: m.searchSeq - 1})
	m = updated.(Model)
	if m.searchQuery != "" || cmd != nil {
		t.Fatalf("stale debounce applied: query %q", m.searchQuery)
	}

	// The latest tick applies the filter live, still in input mode
	updated, cmd = m.Update(searchDebounceMsg{seq: m.searchSeq})
	m = updated.(Model)
	if m.searchQuery != "aq" || !m.searching {
		t.Fatalf("live apply failed: query %q searching %v", m.searchQuery, m.searching)
	}
	if cmd == nil {
		t.Fatal("live apply must refetch the active list")
	}

	// Enter leaves input mode keeping the filter
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.searching || m.searchQuery != "aq" {
		t.Fatalf("after enter: searching=%v query=%q", m.searching, m.searchQuery)
	}
	if view := m.View(); !strings.Contains(view, "esc to clear") {
		t.Error("applied filter not shown in the search bar")
	}

	// Esc clears the applied filter and refetches
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.searchQuery != "" || cmd == nil {
		t.Errorf("esc must clear the filter and refetch (query %q)", m.searchQuery)
	}

	// Switching tabs clears any applied search
	updated, _ = m.Update(keyMsg("/"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("x"))
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.searchQuery != "" {
		t.Errorf("tab switch must clear the search query, got %q", m.searchQuery)
	}
}

// Clicking the search bar focuses it, prefilled with the applied filter
func TestSearchBarClick(t *testing.T) {
	m := NewDemoModel()
	m.demoMode = false
	m.activeTab = TabRevenues
	m.searchQuery = "apple"
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)

	updated, _ = m.Update(tea.MouseMsg{X: 10, Y: searchBarRowY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(Model)
	if !m.searching {
		t.Fatal("clicking the search bar must focus it")
	}
	if m.searchInput != "apple" {
		t.Errorf("searchInput = %q, want prefilled apple", m.searchInput)
	}
}

// The dashboard must surface the company address, CAEN codes and net income
func TestDashboardShowsCompanyDetails(t *testing.T) {
	m := NewDemoModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	m.activeTab = TabDashboard

	view := m.View()
	demoNet := m.summary.TotalRevenues - m.summary.TotalDeductibleExpenses
	for _, want := range []string{
		"Str. Tehnologiei 42",
		"CAEN principal: 6201",
		"CAEN secundare: 6202, 6311",
		"Net Income:",
		fmt.Sprintf("%.2f", demoNet),
	} {
		if !strings.Contains(view, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
}

// The CAEN principal line marquees on the dashboard when it overflows
func TestDashboardCAENMarquees(t *testing.T) {
	m := NewDemoModel()
	// Narrow enough that the demo principal name (70+ chars) overflows
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 40})
	m = updated.(Model)
	m.activeTab = TabDashboard

	before := m.View()
	if !strings.Contains(before, "CAEN principal: 6201") {
		t.Fatal("principal line not at start position before the hold ends")
	}

	m.marqueeOffset = marqueeHoldTicks + 10
	after := m.View()
	if strings.Contains(after, "CAEN principal: 6201") {
		t.Error("principal value did not slide after the marquee hold")
	}
	// The label must stay static while the value scrolls
	if !strings.Contains(after, "CAEN principal: ") {
		t.Error("the CAEN principal label must not scroll away")
	}
}

// Regression: with enough items to completely fill the viewport, the view
// must still fit the terminal exactly. The demo lists are short, so this
// inflates them past any viewport size
func TestFullViewportStaysWithinHeight(t *testing.T) {
	const height = 30

	m := NewDemoModel()
	for len(m.revenues.Items) < 100 {
		m.revenues.Items = append(m.revenues.Items, m.revenues.Items...)
	}
	for len(m.expenses.Items) < 100 {
		m.expenses.Items = append(m.expenses.Items, m.expenses.Items...)
	}
	for len(m.efactura.Items) < 100 {
		m.efactura.Items = append(m.efactura.Items, m.efactura.Items...)
	}
	for len(m.queue.Items) < 100 {
		m.queue.Items = append(m.queue.Items, m.queue.Items...)
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: height})
	m = updated.(Model)

	for _, tab := range []Tab{TabRevenues, TabExpenses, TabEFactura, TabQueue} {
		m.activeTab = tab
		lines := strings.Split(m.View(), "\n")
		if len(lines) != height {
			t.Errorf("tab %s: view has %d lines, want exactly %d", tab, len(lines), height)
		}
		if !strings.Contains(lines[0], "SOLO.ro CLI") {
			t.Errorf("tab %s: title missing from first line: %q", tab, lines[0])
		}
		if !strings.Contains(lines[len(lines)-1], "quit") {
			t.Errorf("tab %s: help bar missing from last line", tab)
		}
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
