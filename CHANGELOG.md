## [Unreleased]

### Added
- **Invoice commands**: `solo-cli invoice <code>` shows an issued invoice with its line items, `invoice download <code>` saves the PDF (`--xml` for the e-Factura XML, `--original` for an uploaded invoice), `invoice paid <code> [YYYY-MM-DD]` and `invoice unpaid <code>` change the payment state and `invoice delete <code>` deletes the newest invoice. The code is the new last column of `solo-cli revenues`
- **Revenue documents**: `revenues queue` and `revenues rejected` list uploaded revenue documents, `revenues delete <id>` removes a queued or rejected one and `upload -r <file>` uploads a revenue document instead of an expense
- **e-Factura for issued invoices**: `efactura status <invoice>` explains where an invoice is in the e-Factura flow (queued with its send date, sent with the ANAF upload index, error with the reason), `efactura send` submits an invoice left on manual sending, `efactura expedite` sends a scheduled one now and `efactura settings` / `efactura settings set --delay 0|2 --contact-info yes|no` show and change the sending preferences. `<invoice>` takes the invoice code or its serial
- **SOLO estimate in taxes**: `solo-cli taxes` now ends with SOLO.ro's own CAS, CASS and income tax estimate for the same year and flags any line that differs from the local calculator by 1 RON or more
- **Declaratia Unica**: `solo-cli declaration [year]` (alias `du`) shows the D212 filing status, the taxes SOLO computed and the 25 May deadline with the days left
- **Alerts**: `solo-cli alerts` lists the active dashboard alerts (VAT threshold, CAEN code update, notarial power of attorney) and `summary` prints them on stderr after its data
- **Documents**: `solo-cli documents` lists filed tax declarations, `documents pfa` the Documente PFA files and `documents download <code>` saves one
- **Downloads of originals**: `expenses download <code>` saves the original of an expense, queued or rejected document
- **Cash flow register**: `solo-cli report [year]` saves the registrul de incasari si plati PDF
- **Account commands**: `solo-cli subscription` (plan, status, price, next billing, card), `solo-cli referral` (link, prizes, rewards), `solo-cli clients [search]` (clients directory) and `solo-cli rates [YYYY-MM-DD]` (BNR exchange rates)
- **Company details**: `solo-cli company` also shows the VIES status and every bank account
- **Shared flags**: commands that change data ask `[y/N]` on a terminal and need `--yes` (`-y`) in scripts. Downloads save to the current directory, `-o` picks a file or directory and an existing file is only overwritten with `--force` (`-f`). The written path is printed on stdout
- **TUI Dashboard notices**: a panel next to the summary box shows the Declaratia Unica status for last year with its 25 May deadline and the active VAT, CAEN and power of attorney alerts. It is hidden when SOLO.ro does not answer
- **TUI SOLO estimate**: the Taxes tab ends with SOLO.ro's own CAS, CASS, income tax and total for the displayed year, refetched with `[` and `]`, and flags lines that differ from the local breakdown
- **TUI rejected revenue documents**: the Revenues tab warns about rejected revenue documents like the Expenses tab does for expenses

### Changed
- **Sign in through cont.solo.ro**: login now uses the new SOLO.ro account pages and bridges the session to the app, falling back to the old login only if those pages change (never when cont.solo.ro is unreachable or shows a 2FA page). Accounts with two-factor authentication get a code prompt in the terminal, the "remember this device" option is always ticked so later runs skip the code for 14 days. Non-interactive runs (scripts, cron) never trigger a code SMS on their own: they stop with a message asking you to run `solo-cli` once in a terminal
- **New last columns**: `revenues` ends with the invoice code, `expenses` and `queue` end with the document code when SOLO.ro has one. Existing columns keep their positions. The rejected expenses warning also shows the document code
- **`queue delete` asks for confirmation**: like the other destructive commands it now needs `--yes` (`-y`) when not run in a terminal, accepts only ids shown by the queue or the rejected expenses, also deletes rejected expense documents and prints `Document deleted`. **Scripts** calling `solo-cli queue delete <id>` must add `-y`
- **Stricter arguments**: unknown subcommands of `efactura`, `queue`, `revenues` and extra arguments to `alerts` now fail with a usage message before signing in instead of being ignored
- **Grouped help**: `solo-cli help` groups the commands by area with one line per subcommand
- **TUI queue delete asks first**: `d` shows `Delete <name>? y/n` on the help row, `y` deletes and any other key cancels. Backspace no longer deletes
- **Rejected warning blocks**: the Revenues and Expenses warnings list at most three documents plus an "and N more" line, so long rejected lists cannot push the table off small terminals
- **Dependencies**: bubbles 1.0.0 and newer charmbracelet/x, x/sys and x/text modules. Building from source now needs Go 1.26 or newer

### Fixed
- **Revenues with foreign currency invoices**: `solo-cli revenues` and the TUI Revenues tab failed to load on accounts with invoices in another currency, because SOLO.ro sends their exchange rate as an object
- **Demo mode crashes**: pressing `r` or deleting a queue document in `solo-cli demo` no longer crashes, refresh does nothing and the delete removes the document from the mock list
- **CLI lists stopped at 100 items**: `revenues`, `expenses`, `queue`, `efactura`, `revenues queue`, `revenues rejected`, `clients` and the rejected documents warnings now fetch every page, `page_size` from the config sets how many items each request asks for. If SOLO.ro never ends a list the command stops after 1000 pages and warns on stderr
- **TUI lists stopped at the first page**: the Revenues, Expenses and e-Factura tabs never loaded more than `page_size` items and the Chart tab only counted the first page of invoices, because SOLO.ro sends no usable total for these lists. They now keep loading while pages come back full and the counter shows `of N+` while more may follow

## [1.7.2]

### Fixed
- **Audit fixes** (multi-agent code audit, 9 confirmed findings): space key no longer inserts double spaces in search, the cursor stays visible after resizes on the Expenses tab, slow page fetches landing after a refresh/search/tab change are dropped instead of corrupting the list, rapid year switching cannot show the wrong year's data, all API calls have a 30 second timeout, server strings are stripped of terminal control characters (escape injection hardening), the `company` command no longer panics on an empty profile response and the skill downloader closes connections promptly

## [1.7.1]

### Added
- **Row detail modal**: press Enter on any list row to open a centered modal with every field untruncated (dates, payment status, local amounts, VAT, statuses, deductibility, queue deadlines). Arrow keys browse items while open, Esc or a click closes it. No nested navigation
- **Chart tab**: monthly revenue bar chart for the selected year, aggregated in RON from the invoice list (foreign currency invoices use their RON local amount). Switch years with `[` and `]`. Loads the complete invoice list automatically when opened

### Changed
- **Quit button moved to the tab bar row**: a `QUIT` button right aligned next to the tabs, in the header as intended. It cannot sit on the very first terminal row because terminals with overlay titlebars swallow clicks there

## [1.7.0]

### Added
- **List paging**: lists are no longer capped at `page_size` items. Scrolling near the end of the loaded items fetches the next page automatically and the status line shows the real server-reported total
- **List search**: an always-visible search bar on the Revenues, Expenses, e-Factura and Queue tabs. Press `/` or click the bar to focus it, typing filters live server-side with a short debounce, Enter keeps the filter, Esc clears it
- **Year switcher**: press `[` and `]` on the Dashboard or Taxes tab to view previous years, like `solo-cli summary 2025` on the CLI. Bounded by the current fiscal year. The Dashboard summary box also shows the last 4 years as clickable options with the selected one highlighted. Years without data show zeros (the API echoes the year back with empty totals)
- **Richer Dashboard**: shows the company address, the CAEN codes (principal with full name and marquee, secundare as a code list) and the net income (revenues - expenses) in the summary box. The `company` CLI command also lists the CAEN codes
- **Test suite**: end-to-end CLI tests against a mock API (exact outputs, config/cookie files, session reuse) and live API schema drift detection that catches SOLO.ro renaming or removing fields our structs map

### Changed
- **Compact header with quit button**: one less blank line between the title and the tabs, the search bar and result counter share one line (counter right aligned) and the help row has a clickable `× quit` button on the right edge (placed on the bottom row because terminals with overlay titlebars swallow clicks on the top row)
- **TUI Queue tab**: removed the `Status` column (queued documents are pending by definition) and the `Days` column is left aligned

### Fixed
- **Revenue/expense summary client types**: the `/revenues/summary` and `/expenses/summary` API endpoints return document counts, not monetary totals. The client mapped a nonexistent `TotalAmount` field that silently decoded to 0 (latent, nothing user-facing consumed it). The types are now `RevenueCounts`/`ExpenseCounts` with the real fields, caught by the new live schema tests

## [1.6.0]

### Added
- **TUI mouse support**: click a tab to switch to it, click a table row to select it and use the mouse wheel to scroll lists and the Taxes view. Note: with mouse capture on, text selection in the terminal needs the usual modifier key (Shift or Option/Alt depending on the terminal)
- **TUI marquee for long text**: when the focused row has text too long for its column (category, supplier, client, document, party), it scrolls in place like a label marquee. Unfocused rows keep the static `...` truncation

### Changed
- **TUI help bar is pinned to the bottom**: the keyboard controls now always render on the last terminal row on every tab instead of floating directly under the content
- **TUI lists use the full terminal height**: the number of visible rows now adapts to the terminal size (and live resizes) instead of being fixed at 10. The Expenses tab accounts for the rejected documents warning block
- **TUI tables use the full terminal width**: the last column of every tab (Client, Supplier, Party, Document) now stretches to fill the available space instead of truncating at a fixed width, and the selection highlight spans the full row. Queue tab columns reordered to `Days, Status, Document` so the filename is the one that flexes. Column widths tightened (Curr 5→4, queue Days 8→5) and Category widened (25→30)
- **TUI Expenses tab**: removed the `Ded` column (every expense that reaches this list is deductible), currency codes are now uppercase to match the other tabs and the amount column is left aligned
- **TUI Revenues tab**: removed the `Paid` column and currency codes are now uppercase for consistency

### Fixed
- **TUI text truncation is now UTF-8 safe**: long names with Romanian diacritics could previously be cut mid-character, producing garbled output
- **Correct salariu minim brut for 2026 plafoane**: default `salariu_minim_brut` changed from 4325 to 4050 RON. The Codul Fiscal pegs CAS/CASS thresholds to the SMB in effect on January 1 of the income year and explicitly ignores mid-year raises, so the July 2026 raise to 4325 RON does not apply to 2026 income. With the old value, bracket detection was optimistic: the 12 salarii CAS threshold appeared at 51900 RON instead of the correct 48600 RON, under-warning users close to owing CAS. **Existing users**: update `salariu_minim_brut` to `4050` in `~/.config/solo-cli/taxes.json` (or delete the file to regenerate it), since it is not migrated automatically

## [1.5.1]

### Added
- **Tax Surplus Hint**: When net income has crossed a CAS/CASS threshold, the same row that previously showed `Buffer:` now shows `Surplus: X RON → prev_bracket` — the deductible expenses needed to drop back under the <plafon>. Only displayed when the contribution saving actually exceeds the required expense, so it stays useful (fires for CAS, suppressed for CASS where dropping a bracket is a net loss)

## [1.5.0]

### Added
- **Auto-discover Company ID**: Company ID is now automatically obtained from the authenticated session — no more manual setup via browser DevTools
- Removed `company_id` from config file (existing field is silently ignored for upgrading users)

## [1.4.0]

### Added
- **Tax Calculator**: New `solo-cli taxes` command with full CAS, CASS, and income tax breakdown
- **TUI Taxes Tab**: New scrollable Taxes tab showing tax breakdown, threshold buffers, and effective rate
- **Configurable Tax Rules**: Auto-generated `~/.config/solo-cli/taxes.json` with editable thresholds, percentages, and minimum gross salary (salariu minim brut)
- **Threshold Buffers**: Shows how much income remains before reaching the next CAS/CASS bracket, with color-coded warnings (green/amber/red)

## [1.3.0]

### Added
- **Setup Skills**: New `solo-cli setup-skills` command to install AI skills for Claude Code and other agents
- **Auto-prompt**: On first interactive run, prompts to install AI skills (downloads latest from GitHub)
- **Homebrew**: Simplified formula, skill installation handled by the binary itself

## [1.2.0]

### Added
- **Rejected Expenses**: New support for rejected expenses from the SOLO.ro API
- **TUI**: Expenses tab now shows rejected expenses with warning banner and rejection reason
- **CLI**: `solo-cli expenses` command now displays rejected expenses with reason before listing normal expenses

## [1.1.1]

### Fixed
- **CLI Version**: The app follows the version tag from its release when using the `solo-cli -v` command

## [1.1.0]

### Added
- **Upload**: New `solo-cli upload <file>` command for documents.
- **Delete**: New `solo-cli queue delete <id>` command.
- **TUI**: Delete queue items with `d` key.
- **Demo**: New `solo-cli demo` mode with mock data.

### Improved
- **Docs**: Comprehensive updates to README and skill/ documentation.
- **Install**: Added Windows/Linux installation instructions.

## [1.0.0]

### Added
- Initial release.
- CLI and TUI for SOLO.ro accounting platform.
- View dashboard, revenues, expenses, e-Factura, and queue.
