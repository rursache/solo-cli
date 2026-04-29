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
