---
name: solo-cli
description: Monitor and interact with SOLO.ro accounting platform via CLI or TUI (summary, revenues, expenses, queue, e-factura, company, taxes). Use when a user asks to check their accounting data, view invoices, expenses, e-factura documents, tax breakdown, or translate a task into safe solo-cli commands.
---

# SOLO CLI

## Overview
Use solo-cli to access SOLO.ro accounting platform data via command-line interface or interactive TUI.

## Installation
If the `solo-cli` command is not available, install via Homebrew:
```bash
brew install rursache/tap/solo-cli
```

## Defaults and safety
- Config file location: `~/.config/solo-cli/config.json` (created on first run)
- Tax config location: `~/.config/solo-cli/taxes.json` (created on first use with 2026 defaults)
- Use `--config` or `-c` to specify a custom config path
- Credentials are stored locally; never passed as command arguments
- Session cookies are cached to `~/.config/solo-cli/cookies.json` for faster subsequent logins
- Company ID is auto-discovered from the authenticated session -- no config field required

## Output conventions
- Data is written to **stdout** in tab-separated format -- safe to pipe to `grep`, `awk`, `cut`, etc
- Status, progress, and error messages are written to **stderr**
- The `expenses` command prints a stderr warning listing any rejected expenses before the main data

## Quick start
- Configure: Edit `~/.config/solo-cli/config.json` with username/password
- Summary: `solo-cli summary`
- Summary for year: `solo-cli summary 2025`
- Revenues: `solo-cli revenues`
- Expenses: `solo-cli expenses`
- Queue: `solo-cli queue`
- E-Factura: `solo-cli efactura`
- Company: `solo-cli company`
- Taxes: `solo-cli taxes`
- Taxes for year: `solo-cli taxes 2025`
- Upload: `solo-cli upload file.pdf`
- Delete: `solo-cli queue delete <ID>`
- TUI: `solo-cli` (no command)
- Demo: `solo-cli demo`
- Install AI skills: `solo-cli setup-skills`
- Version: `solo-cli version`

## Configuration
Config file structure:
```json
{
  "username": "your_email@solo.ro",
  "password": "your_password",
  "page_size": 100,
  "user_agent": "Mozilla/5.0 ..."
}
```

| Field | Required | Description |
|-------|----------|-------------|
| username | Yes | SOLO.ro login email |
| password | Yes | SOLO.ro password |
| page_size | No | Number of items to fetch (default: 100) |
| user_agent | No | Custom HTTP user agent string |

## Commands

### summary [year]
Show account summary for a year.
```bash
solo-cli summary          # Current year
solo-cli summary 2025     # Specific year
```
Output: Year, Revenues, Expenses, Taxes (Taxes line only shown when available)

### revenues
List revenue invoices.
```bash
solo-cli revenues
solo-cli revenue          # Alias
solo-cli rev              # Alias
```
Output (tab-separated): invoice serial code, amount and currency, paid status (PAID/UNPAID), client name

### expenses
List expenses. Prints a stderr warning if there are any rejected expenses.
```bash
solo-cli expenses
solo-cli expense          # Alias
solo-cli exp              # Alias
```
Output (tab-separated): amount and currency, category, supplier name

### queue
List pending documents in expense queue or delete them.
```bash
solo-cli queue            # List queue
solo-cli q                # Alias
solo-cli queue delete <id>  # Delete item by numeric ID
solo-cli queue del <id>     # Alias
solo-cli queue rm <id>      # Alias
```
Output (tab-separated): document name, days pending, overdue status (OVERDUE or blank), ID

### efactura
List e-Factura documents.
```bash
solo-cli efactura
solo-cli einvoice         # Alias
solo-cli ei               # Alias
```
Output (tab-separated): serial code, total amount and currency, invoice date, party name

### company
Show company profile.
```bash
solo-cli company
```
Output: labeled fields -- Name, CUI, Reg (registration number), Address

### upload <file>
Upload an expense document (PDF or image).
```bash
solo-cli upload invoice.pdf
solo-cli up invoice.pdf   # Alias
```
Output: uploaded filename and confirmation that the document was added to the expense queue

### taxes [year]
Show tax breakdown with CAS, CASS, and income tax calculations, bracket labels, buffer to next threshold, and surplus hints.
```bash
solo-cli taxes            # Current year
solo-cli taxes 2025       # Specific year
solo-cli tax              # Alias
```
Output includes: net income (in RON and as multiples of salariu minim brut), CAS amount and bracket, CASS amount and bracket, income tax, total taxes, net after tax, effective rate.

Each CAS/CASS line shows either:
- **Buffer**: how much additional net income before crossing into the next (more expensive) bracket
- **Surplus hint**: when already in a higher bracket, how much in additional deductible expenses would drop back into the cheaper bracket (only shown when the contribution saving exceeds the required expense)

Uses configurable thresholds from `~/.config/solo-cli/taxes.json`. See the reference for the full output format.

**January-1 rule for salariu_minim_brut**: taxes.json defaults to **4050 RON** for 2026 income. This is the SMB in effect on January 1, 2026. The Codul Fiscal pegs CAS/CASS plafoane to that value and ignores mid-year raises -- the July 2026 raise to 4325 does not apply to 2026 income; it first matters for 2027.

### setup-skills
Install AI skill files for Claude Code and other agents.
```bash
solo-cli setup-skills
```
Installs to `~/.agents/skills/solo-cli/` and `~/.claude/skills/solo-cli/`. The TUI also auto-prompts for this on first launch.

### demo
Start TUI with mock data for screenshots or testing (no API calls).
```bash
solo-cli demo
```

### tui
Start interactive TUI mode (default when no command given).
```bash
solo-cli tui
solo-cli                  # Same as above
```

### version
Print the current version string.
```bash
solo-cli version
solo-cli --version
solo-cli -v
```

## Global options

| Option | Short | Description |
|--------|-------|-------------|
| --config | -c | Path to custom config file |
| help | -h | Show help message |
| version | -v | Show version |

## Examples
```bash
# Basic usage
solo-cli summary
solo-cli revenues

# Custom config
solo-cli -c ~/work-config.json summary

# Pipe to grep
solo-cli expenses | grep -i "food"

# View specific year
solo-cli summary 2024

# Upload a document
solo-cli upload invoice.pdf

# Delete a queued item
solo-cli queue delete 123456

# Tax breakdown for a past year
solo-cli taxes 2025
```

## Authentication flow
1. On startup, loads cookies from `~/.config/solo-cli/cookies.json`
2. Validates cookies with a test API call
3. If valid, uses cached session
4. If invalid or missing, logs in with credentials from config
5. Saves new cookies for next session
6. Company ID is auto-discovered after authentication

## Troubleshooting
- **"credentials missing"**: Edit config.json with your SOLO.ro username and password
- **"authentication failed"**: Check credentials are correct
- **"invalid JSON in config"**: Fix syntax errors in config.json
- **Company info not showing**: Company ID is auto-discovered; clear cookies and log in again

## Reference
See `references/help-man-page.md` for the full help text, detailed output formats, taxes.json schema, threshold bracket documentation, and surplus hint logic.
