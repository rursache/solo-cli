# solo-cli

## Project Overview

solo-cli is a Go CLI and terminal UI (TUI) application for [SOLO.ro](https://solo.ro), an online accounting platform used by Romanian freelancers (PFA - Persoana Fizica Autorizata). It provides both a command-line interface for scripting and piping, and an interactive Bubble Tea-based TUI with tabbed navigation.

Key capabilities:
- View dashboard summary (revenues, expenses, taxes per year) and active account alerts
- List and browse revenues/invoices, expenses, queued documents, and e-Factura (national electronic invoicing)
- Invoice details, PDF/XML/original downloads, mark paid or unpaid, delete the newest invoice
- e-Factura status of issued invoices, manual and expedited sending, sending preferences
- Calculate CAS/CASS/income tax breakdown with bracket thresholds and surplus hints, compared with SOLO's own estimate
- Declaratia Unica status and deadline
- Upload expense and revenue documents (PDF, images) to the processing queues, delete queued or rejected ones
- Filed declarations, Documente PFA and cash flow register downloads
- Company VIES and bank accounts, subscription, referral, clients and BNR exchange rates
- SSO login through cont.solo.ro with 2FA and cookie-based session persistence
- Demo mode with mock data for screenshots and testing
- AI skill installation for Claude Code and other agents

Repository: `github.com/rursache/solo-cli`

## Codebase Structure

Package-level map (read the actual files for details, names are self-describing):

- **root** (`main` package): CLI entry point, command routing and help (`main.go`), the original command handlers (`commands.go`: summary, revenues, expenses, queue, efactura list, company, taxes, upload), one `cmd_*.go` per newer area (`cmd_revenues.go` revenue documents, `cmd_invoices.go`, `cmd_efactura.go`, `cmd_taxes.go` SOLO estimate, `cmd_declaration.go`, `cmd_alerts.go`, `cmd_account.go` documents, downloads, report, company extras, subscription, referral, clients, rates), shared flag/confirm/download helpers (`cliutil.go`), auth/client setup with the 2FA prompt (`setup.go`), AI skill installer (`skills.go`), e2e tests (`e2e_test.go` plus `cmd_*_test.go` that reuse its harness)
- **`client/`**: SOLO.ro API client. One file per resource with its types (revenues, revenue_documents, invoices, einvoice, efactura, expenses, company, company_discover, vies, bank_accounts, documents, reports, tax_estimate, tax_declaration, alerts, subscription, referral, customers, exchange_rates, upload), auth in `sso.go` + `htmlform.go` + `cookies.go`, shared HTTP plumbing in `http.go`, `envelope.go` (Ok and Status envelopes), `download.go`, `lenient.go` (number/bool/string decoding that tolerates type drift) and `timezone.go` (embedded Europe/Bucharest), mock data for demo mode in `demo.go`, live/schema tests behind the `live` build tag
- **`config/`**: config.json handling (`config.go`) and taxes.json with `DefaultTaxConfig()` (`taxes.go`)
- **`taxes/`**: pure tax calculation engine, no I/O
- **`tui/`**: Bubble Tea UI. Elm-style split (`model.go`, `update.go`, `view.go`, `fetch.go`, `mouse.go`), one `tab_*.go` per tab, styles in `styles.go`
- **`skill/`**: AI skill manifest (`SKILL.md`) and CLI reference, downloaded by `setup-skills` from the master branch
- **`.github/workflows/`**: `trigger-tap-update.yml` dispatches the Homebrew tap update on release

## Key Concepts

### Authentication and API Client

The application authenticates against the SOLO.ro API at `https://falcon.solo.ro`.

- **Login** (`client/sso.go`, `client/htmlform.go`): signs in on `https://cont.solo.ro` (ASP.NET Razor Pages, `ssoBaseURL`, env override `SOLO_SSO_BASE`): GET `/login?returnTo=%2fdashboard`, post email, post password, post `SkipMfa` on the 2FA recommendation page (never `ActivateMfa`), then GET the falcon bridge from `data-redirect-url` (https `solo.ro` hosts only) which sets `solo_auth`. Forms are parsed with `golang.org/x/net/html`, every hidden input (antiforgery token) is posted back. Max 8 pages. Forms are only ever posted from and to the `ssoBaseURL` origin (https `cont.solo.ro`), anything else is a hard error so credentials never leave it
- **2FA**: `Login(username, password, prompt)` takes a `client.MFAPromptFunc`. `setup.go` passes one only when stdin is a TTY (code prompt on stderr, empty or `r` resends, the remember-device box is always posted checked). The code page is found by `#mfaForm`, `#otpCodeHidden` or `.otp-digit`, else by a form field named like a code or otp. A page recognised as 2FA that cannot be filled in returns `ErrMFAUnsupported`. Without a prompt the code page returns `ErrMFARequired`. Both, and the first prompt call, make `setup.go` write `.mfa-required` in the config dir, after which non-interactive runs refuse to start a login (reaching the code page texts the user) unless a remember cookie is saved
- **Legacy fallback**: POST `/api/security/login` (`UserName`/`Password`, `AuthenticationStatus: "OK"`) is used only on a structural SSO failure (unknown page, missing form, non-2xx, off-domain bridge, no `solo_auth` after the bridge, too many steps), never after rejected credentials, a 2FA page or a network failure (offline, DNS, timeout, like the iOS app). `Client.LoginFallback` says why, cleaned of control characters
- **Cookie persistence**: `client/cookies.go` wraps `cookiejar.Jar` to record Set-Cookie metadata (domain, path, expiry, host-only, secure) and saves falcon and cont.solo.ro cookies to `~/.config/solo-cli/cookies.json` (0600). The `__Host-solo_remember` cookie (14 days) spares the 2FA code. Before a fresh login every cookie except the remember one is dropped. Old name/value-only files load as falcon host-only cookies. On startup saved cookies are validated with `GetSummary`
- **The key auth cookie is named `solo_auth`**, a session cookie with no expiry (zero `Expires` counts as valid).
- **User-Agent**: Configurable via config; defaults to a Chrome user-agent string. All API requests include this header along with `Origin`, `Referer`, and `Accept` headers to mimic browser behavior.

### SOLO.ro API

All API endpoints are under `https://falcon.solo.ro`:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `https://cont.solo.ro/login` | GET/POST forms | SSO authentication (then `/bridge` on falcon) |
| `/api/security/login` | POST | Legacy authentication fallback |
| bridge from `data-redirect-url` | GET | SSO bridge from cont.solo.ro (URL read from the login success page), sets `solo_auth` |
| `/dashboard` | GET (HTML) | Company ID and `CompanyStartYear` discovery |
| `/proxy/accounting/dashboard/summary` | GET | Year summary (revenues, expenses, taxes) |
| `/proxy/accounting/revenues/list` | POST | List revenue invoices |
| `/proxy/accounting/revenues/summary` | GET | Revenue document counts |
| `/proxy/accounting/revenues/queued` | POST | List queued revenue documents |
| `/proxy/accounting/revenues/rejected` | POST | List rejected revenue documents |
| `/proxy/accounting/revenues/{id}` | DELETE | Delete a queued or rejected revenue document |
| `/proxy/accounting/invoices/load/{code}` | GET | Invoice header and line items |
| `/proxy/accounting/invoices/download-options/{code}` | GET | Which invoice files exist (InternalPdf, InternalXml, ExternalFile) |
| `/proxy/accounting/invoices/download/{code}` | GET | Invoice PDF |
| `/proxy/accounting/invoices/mark-paid/{code}` | POST | Mark paid, body is a bare JSON date string at Bucharest midnight |
| `/proxy/accounting/invoices/mark-unpaid/{code}` | POST | Mark unpaid |
| `/proxy/accounting/invoices/pre-cancel/{code}` | GET | Whether an invoice is Deletable (only the newest) or Voidable |
| `/proxy/accounting/invoices/cancel/{code}` | POST | Delete the newest invoice |
| `/proxy/accounting/e-invoice/download/{code}` | GET | e-Factura XML of an issued invoice |
| `/proxy/accounting/e-invoice/list-expenses` | POST | List received e-Factura documents |
| `/proxy/accounting/e-invoice/queue-status/{code}` | GET | Queue state of a queued e-Factura |
| `/proxy/accounting/e-invoice/error-reason/{code}` | GET | Why an e-Factura failed |
| `/proxy/accounting/e-invoice/spv-attributes/{code}` | GET | ANAF upload index of a sent e-Factura |
| `/proxy/accounting/e-invoice/manual-send/{code}` | POST | Send a skipped invoice to ANAF |
| `/proxy/accounting/e-invoice/expedited-send/{code}` | POST | Send a scheduled e-Factura now |
| `/proxy/accounting/e-invoice/settings` | GET | SPV connection and sending preferences |
| `/proxy/accounting/e-invoice/settings/update` | POST | Save SendingDelayDays and SendContactInfo |
| `/proxy/accounting/expenses/list` | POST | List expenses |
| `/proxy/accounting/expenses/summary` | GET | Expense document counts |
| `/proxy/accounting/expenses/queued` | POST | List queued documents |
| `/proxy/accounting/expenses/rejected` | POST | List rejected expenses |
| `/proxy/accounting/expenses/{id}` | DELETE | Delete a queued or rejected expense document |
| `/proxy/accounting/tax-declaration/tax-estimate/{year}` | GET | SOLO's CAS/CASS/income tax estimate |
| `/proxy/accounting/tax-declaration/tax-dashboard/{year}` | GET | Declaratia Unica status and taxes |
| `/proxy/accounting/vat/alert-status` | GET | VAT threshold alert |
| `/proxy/accounting/caen-update/alert-status` | GET | CAEN code update alert |
| `/proxy/accounting/document-workflows/notarial-power-of-attorney/status` | GET | Procura workflow alert |
| `/proxy/accounting/company/basic-profile/company_{id}` | GET | Company profile |
| `/proxy/accounting/company/caen-codes/company_{id}` | GET | CAEN codes |
| `/proxy/accounting/company/vies-profile/company_{id}` | GET | VIES status (Ok envelope) |
| `/proxy/accounting/company/update-settings` | POST | Invoice mentions, body `{"CompanyCode":"company_{id}","InvoiceMentions": text or null}`, Ok envelope, `CompanyNotFound` error code |
| `/proxy/accounting/bank-accounts/company_{id}` | GET | Bank accounts |
| `/proxy/accounting/reports/cashflow-report/company_{id}/{year}` | GET | Cash flow register PDF |
| `/proxy/accounting/subscription/summary` | GET | Subscription, overdue payments, card |
| `/proxy/accounting/subscription/benefits` | GET | Hosted headquarters benefit |
| `/proxy/accounting/customers/list` | POST | Clients directory |
| `/proxy/accounting/utility/exchange-rates` | POST | BNR rates, body is a bare JSON date string |
| `/api/documents/tax-declarations` | GET | Filed tax declarations |
| `/api/documents/registration-documents` | GET | Documente PFA sections |
| `/api/documents/download/{code}` | GET | Declaration or PFA file (unknown codes answer an empty 200) |
| `/api/documents/download-bulk/{documentTypeCode}` | GET | Archive of one Documente PFA section, Referer `/documents/company`, only used for sections with more than one file like the web |
| `/api/financial-documents/download/{documentCode}` | GET | Original of an expense, revenue or external invoice document |
| `/api/referral/campaign` | GET | Referral code and prizes |
| `/api/referral/rewards` | GET | Earned and pending referral rewards |
| `/api/local-storage/upload/{uploadID}` | POST | Multipart file upload |
| `/api/financial-documents/save/{expenses or revenues}/{uploadID}` | POST | Confirm uploaded document |

List endpoints accept JSON bodies with `StartIndex`, `MaxResults`, `SearchText`, `SortBy`, and `SortAsc` fields. `TotalResults` cannot be trusted: revenues, expenses, queued, rejected and customers send `null`, e-Factura sends the length of the returned page. `client.fetchAll` (CLI) and the TUI (`listMore` per tab) keep paging while pages come back full when the total is null or from e-Factura, a non-null total from any other list still ends paging once reached. `revenues/list` search does not match invoice codes, `client.FindRevenueByCode` pages the list instead. Invoice dates carry a `+02:00`/`+03:00` offset and request dates are sent as Bucharest midnight (`client.InvoiceAPIDate`)

### TUI Architecture

The TUI is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm-architecture pattern) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling.

- **Tabs**: Dashboard, Revenues, Expenses, e-Factura, Queue, Taxes, Chart
- **Navigation**: Tab/arrow keys switch tabs, j/k or arrow keys navigate lists, `[` and `]` switch year on Dashboard, Taxes and Chart, `r` refreshes all data (a no-op in demo mode)
- **Queue delete**: `d` (or Delete) stores the selected document in `confirmDelete` and the help row asks `Delete <name>? y/n`. The next key decides: `y` deletes, anything else (and any click) cancels. Backspace never deletes. In demo mode the delete removes the item locally instead of calling the API
- **Data loading**: All data is fetched concurrently on init via `tea.Batch`. Loading is complete when summary, revenues, expenses, rejected expenses, rejected revenues, queue, and efactura data are all present. Company, CAEN codes, alerts, the Declaratia Unica status and the SOLO tax estimate are best effort: a failed call hides their block and never sets `m.err`
- **Dashboard notices**: a panel beside the summary box (stacked below it on narrow terminals, only if it still fits the body height) with the Declaratia Unica status of the last fiscal year (label and deadline via `client.DeadlineText`) and `GetAlerts().Active()`
- **Taxes tab**: Scrollable view of the full tax breakdown (CAS, CASS, income tax, totals, effective rate, threshold buffers, and surplus hints) followed by the SOLO estimate box from `GetTaxEstimate`. Scroll with j/k or arrow keys. Recount `taxesLines` (`updateTaxesLines`) whenever the content changes
- **Year scoped data and stale responses**: `[`, `]` and year clicks call `fetchYear`, which refetches the summary and the tax estimate together. `summaryMsg` and `taxEstimateMsg` both drop responses for a year other than `m.year`. The first summary and a refresh (which resets `estimateYear`) request the estimate from the `summaryMsg` handler
- **Rejected warning blocks**: Revenues and Expenses render the same block above the list (`tui/rejected.go`, capped at three documents). `rejectedBlockHeight` is the single source for `tabViewportSize` and `listChromeShift`, keep the rendered block and that height in sync
- **Viewport scrolling**: Lists use a manual viewport (offset + size) rather than the Bubble Tea viewport component.
- **Demo mode**: `NewDemoModel` preloads every piece of data from `client/demo.go`, including alerts, the declaration, the tax estimate and rejected revenues. Any new TUI data needs a demo counterpart, and code paths reachable in demo mode must not touch `m.client` (it is nil)

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
./solo-cli invoice <code>          # CLI: invoice details (code = last column of revenues)
./solo-cli efactura status <code>  # CLI: e-Factura state of an issued invoice
./solo-cli documents    # CLI: filed tax declarations
./solo-cli demo         # TUI with mock data (no credentials needed)
```

### Test

```bash
go test ./...                     # Offline tests: unit, TUI render, e2e CLI (no credentials needed)
go test -tags live ./client -v    # Live integration + schema drift tests against SOLO.ro
```

- **Offline unit tests**: `taxes/taxes_test.go` (tax math), `config/config_test.go` (config and taxes.json loading), `client/*_test.go` (every API client path against an httptest mock server, one test file per resource, plus `sso_test.go` for the login flow and `helpers_test.go` for the shared plumbing), `cliutil_test.go` (flags, confirmation, download writing). The `baseURL` in `client/client.go` is a var so tests can redirect it
- **TUI render tests** (`tui/app_test.go`): render the demo model at fixed terminal sizes and assert width/height fitting, bottom-pinned help bar, full-viewport behavior, marquee output and mouse hit-testing.
- **E2E CLI tests** (`e2e_test.go` and the `cmd_*_test.go` files that reuse its `newMockAPI`/`newEnv` harness): build the real binary and run it against a mock SOLO.ro API (`SOLO_API_BASE` and `SOLO_SSO_BASE` env overrides) with an isolated HOME. They assert exact command output, exit codes, stdout/stderr split, config and cookie file creation with correct permissions, session cookie reuse across runs, that the CLI taxes output matches `taxes.Calculate`, that mutating commands refuse without `--yes` when stdin is not a terminal and that downloads never overwrite without `--force`
- **Live tests** (`client/live_*_test.go`, build tag `live`): authenticate with the developer's own `~/.config/solo-cli/config.json` (reusing saved cookies like the CLI does) and exercise read-only endpoints. The schema tests fetch raw JSON and verify every field our structs map still exists in the live payloads (Go decodes missing fields to silent zero values, so this is the only way to catch API drift), plus cross-endpoint consistency checks. They never upload or delete anything. Skipped automatically if no valid config exists
- TUI visual testing remains manual via `solo-cli demo`.

### Dependencies

- Go 1.26+
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

Stores serialized HTTP cookies (name, value, domain, path, expires, host-only, secure) for falcon.solo.ro and cont.solo.ro, including the 14 day `__Host-solo_remember` cookie that skips the 2FA code. Created with 0600 permissions. Loaded and validated on each run to avoid unnecessary re-authentication. A `.mfa-required` flag file next to it marks accounts that need a 2FA code, so non-interactive runs never start a login that would text one

### Taxes Config File

Location: `~/.config/solo-cli/taxes.json`

Auto-generated with defaults on first use of any taxes feature (CLI or TUI). Safe to edit manually to adjust thresholds, percentages, or the minimum gross salary for a given year.

```json
{
  "year": 2026,
  "salariu_minim_brut": 4050,
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
- Tests use standard Go testing (`_test.go` files) with httptest mocks, never the live API outside the `live` build tag
- API response types are defined in the same file as the client methods that use them (e.g., `Revenue` type is in `revenues.go`).
- CLI output is tab-separated for piping. Status/progress messages go to stderr; data goes to stdout.
- The TUI uses the Elm architecture pattern (Model, Init, Update, View) via Bubble Tea.
- Error handling follows Go conventions: return errors up the call stack, handle at the top level in `main.go`.
- Shared client plumbing: `doJSON` (decode + sanitize), `doRaw` (bare scalar or empty answers), `doOk` with `OkResponse[T]`/`APIError` for the `{Ok, Data, ErrorCode, ErrorMessage}` envelope, `StatusResponse` for `{Status}` answers, `download` + `DownloadedFile` + `FileExtension` for file endpoints (`client/envelope.go`, `client/download.go`)
- CLI output additions go in a new **last** column so scripts keep working. Commands with arguments validate them before signing in (`run<Name>Cmd(args)` wrappers in `main.go`)
- Shared CLI plumbing (`cliutil.go`): `parseCmdFlags` (`--yes/-y`, `--force/-f`, `-o/--output`), `confirm` (TTY `[y/N]` prompt or `--yes`, required for destructive or outward-facing commands), `saveDownload`/`writeDownload` (never overwrite without `--force`, print the written path on stdout), `exitErr`
