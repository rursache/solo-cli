# solo-cli

## Project Overview

solo-cli is a Go CLI and terminal UI (TUI) application for [SOLO.ro](https://solo.ro), an online accounting platform used by Romanian freelancers (PFA - Persoana Fizica Autorizata). It provides both a command-line interface for scripting and piping, and an interactive Bubble Tea-based TUI with tabbed navigation.

Key capabilities:
- View dashboard summary (revenues, expenses, taxes per year)
- List and browse revenues/invoices, expenses, queued documents, and e-Factura (national electronic invoicing)
- Calculate CAS/CASS/income tax breakdown with bracket thresholds and surplus hints
- Upload expense documents (PDF, images) to the processing queue
- Delete queued expense documents
- Cookie-based session persistence for fast re-authentication
- Demo mode with mock data for screenshots and testing
- AI skill installation for Claude Code and other agents

Repository: `github.com/rursache/solo-cli`

## Codebase Structure

```
solo-cli/
  main.go              - Entry point, CLI command routing, auth flow, TUI launch
  skills.go            - AI skill download/install logic and first-run prompt
  go.mod / go.sum      - Go module definition (module name: solo-cli)
  config/
    config.go          - Config loading/saving, path resolution, validation
    taxes.go           - TaxConfig/TaxThreshold types, DefaultTaxConfig(), LoadTaxes(), EnsureTaxesExists()
  taxes/
    taxes.go           - Tax calculation engine: Calculate(), ThresholdResult, TaxBreakdown, format helpers
  client/
    client.go          - HTTP client with cookie jar, Login(), GetSummary()
    cookies.go         - Cookie persistence (save/load/clear from disk)
    revenues.go        - Revenue types and ListRevenues(), GetRevenueSummary()
    expenses.go        - Expense/Queue/Rejected types, list/delete operations
    efactura.go        - e-Factura types and ListEFactura()
    company.go         - CompanyInfo type and GetCompanyInfo()
    company_discover.go - Auto-discovery of company ID from authenticated HTML
    upload.go          - Two-step document upload (multipart upload + confirm)
    demo.go            - Mock data generators for demo/screenshot mode
  tui/
    app.go             - Bubble Tea Model, Init/Update/View, tab rendering, data fetching
    styles.go          - lipgloss style definitions (colors, tabs, tables, etc.)
  skill/
    SKILL.md           - AI skill manifest for agentic tools
    references/
      help-man-page.md - CLI help reference for AI skills
  docs/
    tui_*.jpg          - TUI screenshots
    PLAN.md, TODO.md, WORK.md - Development notes
  .github/
    workflows/
      trigger-tap-update.yml - GitHub Actions workflow for Homebrew tap updates
```

## Key Concepts

### Authentication and API Client

The application authenticates against the SOLO.ro API at `https://falcon.solo.ro`.

- **Login**: POST to `/api/security/login` with `UserName` and `Password` in JSON body. A successful login returns `AuthenticationStatus: "OK"` and sets session cookies.
- **Cookie persistence**: After login, cookies are serialized to `~/.config/solo-cli/cookies.json` (file permissions 0600). On startup, the app loads saved cookies, validates them with a test API call (`GetSummary`), and only re-authenticates if they are expired or invalid.
- **The key auth cookie is named `solo_auth`**. The cookie loader checks specifically for this cookie when determining if a saved session is valid.
- **User-Agent**: Configurable via config; defaults to a Chrome user-agent string. All API requests include this header along with `Origin`, `Referer`, and `Accept` headers to mimic browser behavior.

### SOLO.ro API

All API endpoints are under `https://falcon.solo.ro`:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/security/login` | POST | Authentication |
| `/proxy/accounting/dashboard/summary` | GET | Year summary (revenues, expenses, taxes) |
| `/proxy/accounting/revenues/list` | POST | List revenue invoices |
| `/proxy/accounting/revenues/summary` | GET | Revenue totals |
| `/proxy/accounting/expenses/list` | POST | List expenses |
| `/proxy/accounting/expenses/summary` | GET | Expense totals |
| `/proxy/accounting/expenses/queued` | POST | List queued documents |
| `/proxy/accounting/expenses/rejected` | POST | List rejected expenses |
| `/proxy/accounting/expenses/{id}` | DELETE | Delete a queued expense |
| `/proxy/accounting/e-invoice/list-expenses` | POST | List e-Factura documents |
| `/proxy/accounting/company/basic-profile/company_{id}` | GET | Company profile |
| `/api/local-storage/upload/{uploadID}` | POST | Multipart file upload |
| `/api/financial-documents/save/expenses/{uploadID}` | POST | Confirm uploaded document |

List endpoints accept JSON bodies with `StartIndex`, `MaxResults`, `SearchText`, `SortBy`, and `SortAsc` fields.

### TUI Architecture

The TUI is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm-architecture pattern) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling.

- **Tabs**: Dashboard, Revenues, Expenses, e-Factura, Queue, Taxes
- **Navigation**: Tab/arrow keys switch tabs, j/k or arrow keys navigate lists, `d` deletes in Queue tab, `r` refreshes all data
- **Data loading**: All data is fetched concurrently on init via `tea.Batch`. Loading is complete when summary, revenues, expenses, rejected, queue, and efactura data are all present.
- **Taxes tab**: Scrollable view of the full tax breakdown (CAS, CASS, income tax, totals, effective rate, threshold buffers, and surplus hints). Scroll with j/k or arrow keys.
- **Viewport scrolling**: Lists use a manual viewport (offset + size) rather than the Bubble Tea viewport component.

### Tax Calculator

The tax calculator (`taxes/taxes.go`) computes CAS, CASS, and income tax for Romanian PFA freelancers.

- **Entry point**: `taxes.Calculate(totalRevenues, totalExpenses float64, cfg *config.TaxConfig) *TaxBreakdown`
- **Net income** = revenues - expenses. CAS and CASS are each computed via bracket logic keyed on multiples of `salariu minim brut` (SMB).
- **SMB January-1 rule**: the Codul Fiscal pegs CAS/CASS plafoane to the SMB in effect on January 1 of the income year and explicitly ignores mid-year raises. For 2026 income the correct value is 4050 RON (the July 2026 raise to 4325 does not apply, it first matters for 2027 income). Never bump `SalariuMinimBrut` in `DefaultTaxConfig()` just because a raise was announced.
- **Bracket logic**: Each `TaxThreshold` defines `MinSalaries`/`MaxSalaries` (bounds in multiples of SMB), `BaseSalaries` (what percentage is applied to — `0` = exempt, `-1` = proportional/actual net income, positive = fixed multiple of SMB), and a `Label`.
- **Threshold buffers**: Each result includes `BufferToNext` (how much more net income before crossing into the next bracket) and `NextLabel`.
- **Surplus hint** (v1.5.1): When net income has already crossed a bracket boundary, `ExpensesToPrev` and `PrevLabel` are populated to show how much in additional deductible expenses would drop the taxpayer back into the cheaper bracket. The hint is only surfaced when the contribution saving exceeds the required expense (fires for CAS; suppressed for CASS when dropping a bracket would be a net loss).
- **Income tax** = `IncomeTaxPercent` % of (net income - CAS - CASS).
- `TaxBreakdown` also exposes `TotalTaxes`, `NetAfterTax`, and `EffectiveRate`.

### Version Injection

The `version` variable in `main.go` defaults to `"dev"` and is overridden at build time via `-ldflags`:
```bash
go build -ldflags "-X main.version=v1.2.3" -o solo-cli .
```

## Build and Run

### Build

```bash
go build -o solo-cli .
```

### Run

```bash
./solo-cli              # Launch interactive TUI
./solo-cli summary      # CLI: current year summary
./solo-cli revenues     # CLI: list revenues
./solo-cli taxes        # CLI: tax breakdown for current year (alias: tax)
./solo-cli taxes 2025   # CLI: tax breakdown for a specific year
./solo-cli demo         # TUI with mock data (no credentials needed)
```

### Test

```bash
go test ./...                     # Offline unit tests (no credentials needed)
go test -tags live ./client -v    # Live integration tests against SOLO.ro
```

- **Offline tests**: `taxes/taxes_test.go` (tax math), `config/config_test.go` (config and taxes.json loading), `client/client_test.go` (all API client paths against an httptest mock server, including login, lists, upload, company discovery, and cookie persistence). The `baseURL` in `client/client.go` is a var so tests can redirect it.
- **Live tests** (`client/live_test.go`, build tag `live`): authenticate with the developer's own `~/.config/solo-cli/config.json` (reusing saved cookies like the CLI does) and exercise read-only endpoints. They never upload or delete anything. Skipped automatically if no valid config exists.
- TUI testing remains manual via `solo-cli demo`.

### Dependencies

- Go 1.25.5+
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - TUI components (spinner)
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/google/uuid` - UUID generation for upload IDs

## Release Process

1. **Update CHANGELOG.md** with the new version section and list of changes.
2. **Commit and push** the changes to the `master` branch.
3. **Create a GitHub release** with a tag matching `vX.Y.Z` (e.g., `v1.3.0`). The release notes should match the CHANGELOG entry for that version.
4. **GitHub Actions triggers automatically**: The `trigger-tap-update.yml` workflow fires on the `release: published` event. It uses `actions/github-script` to dispatch the `update-formula.yml` workflow in `rursache/homebrew-tap`.
5. **Homebrew tap update** (in `rursache/homebrew-tap`): The triggered workflow downloads the source tarball for the new tag, computes its SHA256 hash, updates the Homebrew formula with the new version and hash, commits, and pushes.
6. **Users receive the update** via `brew update && brew upgrade solo-cli`.

The workflow can also be triggered manually via `workflow_dispatch` with a `tag` input.

## GitHub Actions

### trigger-tap-update.yml

Located at `.github/workflows/trigger-tap-update.yml`.

- **Triggers**: On `release: published` events, or manually via `workflow_dispatch` with a `tag` input.
- **What it does**: Determines the tag (from release payload or manual input), then dispatches the `update-formula.yml` workflow in the `rursache/homebrew-tap` repository.
- **Authentication**: Uses the `TAP_GITHUB_TOKEN` secret (a personal access token with permissions to trigger workflows in the homebrew-tap repo).
- **Dispatch payload**: Sends `formula: "solo-cli"`, `tag`, and `repository` (owner/repo of this project) as inputs to the tap workflow.

## AI Skills

The `setup-skills` command and auto-prompt system install AI skill files for Claude Code and other agentic tools.

### How it works

1. **Auto-prompt on first interactive run**: When the user launches the TUI for the first time (no command arguments), `maybePromptSkillInstall()` checks if skills are already installed and if the prompt has been shown before. If not, it asks the user via stdin.
2. **Prompt tracking**: A `.skill-prompted` flag file is created in the config directory (`~/.config/solo-cli/`) to avoid re-prompting.
3. **Download**: Skill files (`SKILL.md` and `references/help-man-page.md`) are downloaded from the `master` branch of the GitHub repository at `raw.githubusercontent.com/rursache/solo-cli/master/skill/`.
4. **Installation targets**: Files are installed to two directories:
   - `~/.agents/skills/solo-cli/` (generic agents)
   - `~/.claude/skills/solo-cli/` (Claude Code specific)
5. **Manual install**: Run `solo-cli setup-skills` to install or update skills at any time.

## Configuration

### Config File

Location: `~/.config/solo-cli/config.json` (created automatically on first run with empty values).

Override with `--config` or `-c` flag.

```json
{
  "username": "your_email@example.com",
  "password": "your_password",
  "page_size": 100,
  "user_agent": "Mozilla/5.0 ..."
}
```

- `username` (required): SOLO.ro login email
- `password` (required): SOLO.ro password
- Company ID is auto-discovered at runtime from the authenticated session (no config needed).
- `page_size` (optional, default 100): Number of items to fetch per API call
- `user_agent` (optional): Custom HTTP User-Agent header

File is created with permissions 0600 (owner read/write only).

### Cookie File

Location: `~/.config/solo-cli/cookies.json`

Stores serialized HTTP cookies (name, value, domain, path, expires) from the SOLO.ro session. Also created with 0600 permissions. Loaded and validated on each run to avoid unnecessary re-authentication.

### Taxes Config File

Location: `~/.config/solo-cli/taxes.json`

Auto-generated with defaults on first use of any taxes feature (CLI or TUI). Safe to edit manually to adjust thresholds, percentages, or the minimum gross salary for a given year.

```json
{
  "year": 2026,
  "salariu_minim_brut": 4325,
  "income_tax_percent": 10,
  "cas_percent": 25,
  "cas_thresholds": [...],
  "cass_percent": 10,
  "cass_thresholds": [...]
}
```

Each threshold entry has `min_salaries`, `max_salaries`, `base_salaries`, and `label`. `base_salaries` values: `0` = exempt, `-1` = proportional (actual net income), positive = fixed multiple of SMB. Created with 0644 permissions.

## Code Conventions

- This is a Go project. Do not apply Swift/iOS patterns.
- Module name is `solo-cli` (not a domain-style module path).
- No test files exist yet. If adding tests, use standard Go testing (`_test.go` files).
- API response types are defined in the same file as the client methods that use them (e.g., `Revenue` type is in `revenues.go`).
- CLI output is tab-separated for piping. Status/progress messages go to stderr; data goes to stdout.
- The TUI uses the Elm architecture pattern (Model, Init, Update, View) via Bubble Tea.
- Error handling follows Go conventions: return errors up the call stack, handle at the top level in `main.go`.
