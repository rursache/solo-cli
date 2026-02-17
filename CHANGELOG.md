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

