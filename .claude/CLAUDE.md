# solo-cli

## Project Overview

solo-cli is a Go CLI and terminal UI (TUI) application for [SOLO.ro](https://solo.ro), an online accounting platform used by Romanian freelancers (PFA - Persoana Fizica Autorizata). It provides both a command-line interface for scripting and piping, and an interactive Bubble Tea-based TUI with tabbed navigation.

Key capabilities:
- View dashboard summary (revenues, expenses, taxes per year)
- List and browse revenues/invoices, expenses, queued documents, and e-Factura (national electronic invoicing)
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
  client/
    client.go          - HTTP client with cookie jar, Login(), GetSummary()
    cookies.go         - Cookie persistence (save/load/clear from disk)
    revenues.go        - Revenue types and ListRevenues(), GetRevenueSummary()
    expenses.go        - Expense/Queue/Rejected types, list/delete operations
    efactura.go        - e-Factura types and ListEFactura()
    company.go         - CompanyInfo type and GetCompanyInfo()
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

- **Tabs**: Dashboard, Revenues, Expenses, e-Factura, Queue
- **Navigation**: Tab/arrow keys switch tabs, j/k or arrow keys navigate lists, `d` deletes in Queue tab, `r` refreshes all data
- **Data loading**: All data is fetched concurrently on init via `tea.Batch`. Loading is complete when summary, revenues, expenses, rejected, queue, and efactura data are all present.
- **Viewport scrolling**: Lists use a manual viewport (offset + size) rather than the Bubble Tea viewport component.

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
./solo-cli demo         # TUI with mock data (no credentials needed)
```

### Test

There are currently no automated tests in this project. Manual testing is done by building and running the CLI commands or TUI against the live SOLO.ro API, or using `solo-cli demo` for TUI testing without credentials.

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
  "company_id": "32-character-hex-id",
  "page_size": 100,
  "user_agent": "Mozilla/5.0 ..."
}
```

- `username` (required): SOLO.ro login email
- `password` (required): SOLO.ro password
- `company_id` (optional): 32-character hex ID for company profile display. Found in the SOLO.ro Network tab at `/settings#!/company` -- look for requests to `company_XXXXXXXX`.
- `page_size` (optional, default 100): Number of items to fetch per API call
- `user_agent` (optional): Custom HTTP User-Agent header

File is created with permissions 0600 (owner read/write only).

### Cookie File

Location: `~/.config/solo-cli/cookies.json`

Stores serialized HTTP cookies (name, value, domain, path, expires) from the SOLO.ro session. Also created with 0600 permissions. Loaded and validated on each run to avoid unnecessary re-authentication.

## Code Conventions

- This is a Go project. Do not apply Swift/iOS patterns.
- Module name is `solo-cli` (not a domain-style module path).
- No test files exist yet. If adding tests, use standard Go testing (`_test.go` files).
- API response types are defined in the same file as the client methods that use them (e.g., `Revenue` type is in `revenues.go`).
- CLI output is tab-separated for piping. Status/progress messages go to stderr; data goes to stdout.
- The TUI uses the Elm architecture pattern (Model, Init, Update, View) via Bubble Tea.
- Error handling follows Go conventions: return errors up the call stack, handle at the top level in `main.go`.
