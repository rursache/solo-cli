## [Unreleased]

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
