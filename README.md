# Solo CLI

A terminal-based user interface for [SOLO.ro](https://solo.ro), an online accounting platform for Romanian freelancers (PFA).

![Dashboard](docs/tui_1.jpg)

## Features

- 🔐 Secure authentication with SOLO.ro
- 📊 Dashboard with company info and yearly summary
- 💰 View revenues and expenses
- 📄 View e-Factura (national electronic invoicing system)
- 📤 Upload expense documents (PDF, Images)
- 🗑️ Delete expenses/queued documents
- 🍪 Cookie persistence for faster logins
- 🧮 Tax calculator with CAS/CASS/income tax breakdown and threshold buffers

## Installation

### macOS (Homebrew)

```bash
brew install rursache/tap/solo-cli
```

### Windows / Linux / macOS (Go)

```bash
go install github.com/rursache/solo-cli@latest
```

### Build from Source

```bash
git clone https://github.com/rursache/solo-cli.git
cd solo-cli
go build -o solo-cli .
```

## Screenshots

![Revenues](docs/tui_2.jpg)
![Expenses](docs/tui_3.jpg)
![e-Factura](docs/tui_4.jpg)
![Queue](docs/tui_5.jpg)
![Taxes](docs/tui_6.jpg)

## Configuration

On first run, the CLI creates a config at `~/.config/solo-cli/config.json`:

```json
{
  "username": "your_email@example.com",
  "password": "your_password",
  "company_id": "your_company_id",
  "page_size": 100,
  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36..."
}
```

| Field | Required | Description |
|-------|----------|-------------|
| username | Yes | SOLO.ro login email |
| password | Yes | SOLO.ro password |
| company_id | No | Company ID for profile display |
| page_size | No | Number of items to fetch (default: 100) |
| user_agent | No | Custom HTTP user agent string |

### Tax Configuration

On first run of the `taxes` command (or TUI Taxes tab), the CLI creates `~/.config/solo-cli/taxes.json` with default 2026 values:

```json
{
  "year": 2026,
  "salariu_minim_brut": 4325,
  "income_tax_percent": 10,
  "cas_percent": 25,
  "cas_thresholds": [
    { "min_salaries": 0,  "max_salaries": 12, "base_salaries": 0,  "label": "Fără CAS (sub 12 salarii)" },
    { "min_salaries": 12, "max_salaries": 24, "base_salaries": 12, "label": "CAS pe 12 salarii" },
    { "min_salaries": 24, "max_salaries": 0,  "base_salaries": 24, "label": "CAS pe 24 salarii" }
  ],
  "cass_percent": 10,
  "cass_thresholds": [
    { "min_salaries": 0,  "max_salaries": 6,  "base_salaries": 6,  "label": "CASS minim (6 salarii)" },
    { "min_salaries": 6,  "max_salaries": 72, "base_salaries": -1, "label": "CASS proporțional" },
    { "min_salaries": 72, "max_salaries": 0,  "base_salaries": 72, "label": "CASS plafonat (72 salarii)" }
  ]
}
```

**Threshold fields:**
- `min_salaries` / `max_salaries`: income bracket bounds in multiples of SMB (`0` = unlimited)
- `base_salaries`: what to multiply by the percentage — positive = fixed multiple of SMB, `0` = exempt, `-1` = proportional (use actual net income)

Update `salariu_minim_brut` when it changes, and adjust thresholds as tax law evolves.

### Finding Your Company ID

1. Log in to [SOLO.ro](https://falcon.solo.ro)
2. Go to **Settings → Company**: https://falcon.solo.ro/settings#!/company
3. Open browser DevTools (F12) → **Network** tab
4. Type `company_` in the filter box
5. Look for a request like `company_0e5f5310aec44ea7ba27025d2fd7551c`
6. Copy the ID part (the 32 characters after `company_`)

## Usage

### Interactive TUI Mode

```bash
solo-cli
```

Navigate with keyboard:
- `Tab` / `←` `→` - Switch between tabs
- `↑` `↓` / `j` `k` - Navigate lists
- `d` - Delete item (Queue tab only)
- `r` - Refresh data
- `q` - Quit

**Tabs:** Dashboard → Revenues → Expenses → e-Factura → Queue → Taxes

### CLI Commands

```bash
solo-cli summary          # Account summary (current year)
solo-cli summary 2025     # Summary for specific year
solo-cli taxes            # Tax breakdown (alias: tax)
solo-cli taxes 2025       # Tax breakdown for specific year
solo-cli revenues         # List revenues (alias: rev)
solo-cli expenses         # List expenses (alias: exp)
solo-cli efactura         # e-Factura documents (alias: ei)
solo-cli queue            # Expense queue (alias: q)
solo-cli company          # Company profile
solo-cli upload file.pdf  # Upload expense document (alias: up)
solo-cli queue delete 123 # Delete queued item by ID
```

### Global Options

```bash
solo-cli --help           # Show help
solo-cli --version        # Show version
solo-cli -c /path/to/config.json summary  # Use custom config
```

### Examples

```bash
# Pipe to grep
solo-cli expenses | grep -i "food"

# Use custom config
solo-cli -c ~/work.json revenues

# View past year
solo-cli summary 2024
```

Output is tab-separated for piping to other tools.

## AI Skills

This project also provides a "skill" for agentic AI tools, allowing AI assistants to interact with SOLO.ro on your behalf:

- **GitHub**: [skill folder](https://github.com/rursache/solo-cli/tree/master/skill)
- **ClawdHub**: [rursache/solo-cli](https://clawdhub.com/rursache/solo-cli)

## Acknowledgments

This entire codebase was created using [Claude Opus 4.5](https://www.anthropic.com/claude) and [Claude Opus 4.6](https://www.anthropic.com/claude). Issues and PRs are welcome.

## License

MIT License - see [LICENSE](LICENSE) for details
