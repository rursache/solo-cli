---
name: solo-cli
description: Monitor and interact with SOLO.ro accounting platform via CLI or TUI (summary, taxes, Declaratia Unica, alerts, revenues and invoices, e-Factura, expenses, queue, documents, cash flow register, company, subscription, clients, exchange rates). Use when a user asks to check their accounting data, view or download invoices and documents, mark invoices paid, follow e-Factura, check taxes or translate a task into safe solo-cli commands
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
- Accounts with two-factor authentication need one interactive run in a terminal to enter the code, the device is then remembered for 14 days. Without a terminal the CLI never triggers a code SMS, it stops with a message instead

## Mutating commands (ask the user first)
These change data on SOLO.ro or at ANAF: `queue delete`, `revenues delete`, `invoice paid`, `invoice unpaid`, `invoice delete`, `efactura send`, `efactura expedite`, `efactura settings set` and `upload`
- All of them except `upload` ask `[y/N]` on a terminal and, without a terminal (agents, scripts), refuse unless `--yes` / `-y` is passed
- Only add `-y` after the user explicitly confirmed that exact action on that exact item. Show the item first (`solo-cli invoice <code>`, `solo-cli efactura status <invoice>`, `solo-cli queue`)
- `efactura send` submits the invoice to ANAF and cannot be undone. `invoice delete` permanently deletes the newest invoice, older ones are refused
- Connecting the ANAF SPV, filing the Declaratia Unica and subscription changes only happen on the web, the CLI points to the right page

## Output conventions
- Data is written to **stdout** in tab-separated format -- safe to pipe to `grep`, `awk`, `cut`, etc
- Status, progress, and error messages are written to **stderr**
- The `expenses` and `revenues` commands print a stderr warning listing any rejected documents before the main data
- New columns are always appended last (invoice code on `revenues`, document code on `expenses` and the queue lists), so existing column positions do not change
- Downloads (`invoice download`, `expenses download`, `documents download`, `report`) write a file, print its path on stdout, take `-o <path or dir>` and refuse to overwrite without `-f`

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
- Declaratia Unica: `solo-cli declaration`
- Alerts: `solo-cli alerts`
- Invoice details: `solo-cli invoice <code>` (code = last column of `revenues`)
- Invoice PDF: `solo-cli invoice download <code>`
- e-Factura status: `solo-cli efactura status <code or serial>`
- Documents: `solo-cli documents`, `solo-cli documents pfa`
- Cash flow register: `solo-cli report 2025`
- Clients: `solo-cli clients`
- Exchange rates: `solo-cli rates`
- Upload: `solo-cli upload file.pdf` (revenue document: `solo-cli upload -r file.pdf`)
- Delete: `solo-cli queue delete <ID> -y`
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
Output: Year, Revenues, Expenses, Taxes (Taxes line only shown when available). Active account alerts follow on stderr as `Alert: <title>` blocks, stdout is unchanged

### declaration [year]
Declaratia Unica (D212) status for a fiscal year, last year by default
```bash
solo-cli declaration        # Last year, filed this year
solo-cli declaration 2025
solo-cli du                 # Alias
```
Output: status label and raw value, last completed date, CAS, CASS, income tax, total (or `Taxes: not computed yet`), the 25 May deadline with days left and the web link to file or review it

### alerts
Active dashboard alerts (VAT threshold, CAEN code update, notarial power of attorney)
```bash
solo-cli alerts
solo-cli alert            # Alias
```
Output (tab-separated): kind (vat, caen, procura), title, message, web URL. Empty stdout and `No alerts` on stderr when nothing is active

### revenues
List revenue invoices.
```bash
solo-cli revenues
solo-cli revenue          # Alias
solo-cli rev              # Alias
```
Output (tab-separated): invoice serial code, amount and currency, paid status (PAID/UNPAID), client name, invoice code

Uploaded revenue documents:
```bash
solo-cli revenues queue            # Still processing: name, days, OVERDUE, (ID: n), document code
solo-cli revenues rejected         # Rejected: name, reason, (ID: n), document code
solo-cli revenues delete <id> -y   # Delete a queued or rejected revenue document (mutating)
```

### invoice <code>
Issued invoice details and actions. `<code>` is the last column of `solo-cli revenues`
```bash
solo-cli invoice <code>                        # Header and line items (alias: inv)
solo-cli invoice download <code>               # PDF, Factura-<serial>.pdf
solo-cli invoice download <code> --xml         # e-Factura XML
solo-cli invoice download <code> --original    # Uploaded original of an external invoice
solo-cli invoice paid <code> [YYYY-MM-DD] -y   # Mark paid, default today (mutating)
solo-cli invoice unpaid <code> -y              # Mark unpaid (mutating)
solo-cli invoice delete <code> -y              # Delete the newest invoice (mutating, permanent)
```
Output of `invoice <code>`: `Key: value` header lines (Invoice, Code, Client, Issued, Due, Status, Payment, e-Factura, Amount, VAT, Total, Local total), a blank line, then tab-separated items: name, quantity and unit, unit price, VAT %, total

### expenses
List expenses. Prints a stderr warning if there are any rejected expenses.
```bash
solo-cli expenses
solo-cli expense          # Alias
solo-cli exp              # Alias
```
Output (tab-separated): amount and currency, category, supplier name, document code (only when SOLO.ro has one)

Rejected expense documents with the id `queue delete` takes:
```bash
solo-cli expenses rejected   # name, reason, (ID: n), document code
```

Download the original document by its code (also works for queue and rejected document codes):
```bash
solo-cli expenses download <code> [-o path] [-f]
```

### queue
List pending documents in expense queue or delete them.
```bash
solo-cli queue            # List queue
solo-cli q                # Alias
solo-cli queue delete <id> -y  # Delete a queued or rejected expense document (mutating)
solo-cli queue del <id> -y     # Alias
solo-cli queue rm <id> -y      # Alias
```
Output (tab-separated): document name, days pending, overdue status (OVERDUE or blank), ID, document code when set

### efactura
List received e-Factura documents and follow the e-Factura of issued invoices
```bash
solo-cli efactura                            # Received documents (aliases: einvoice, ei)
solo-cli efactura status <invoice>           # e-Factura state of an issued invoice (code or serial)
solo-cli efactura settings                   # SPV connection and sending preferences
solo-cli efactura send <invoice> -y          # Send a manual-sending invoice to ANAF (mutating, irreversible)
solo-cli efactura expedite <invoice> -y      # Send a scheduled e-Factura now (mutating)
solo-cli efactura settings set --delay 0 -y  # --delay 0|2, --contact-info yes|no (mutating)
```
Output of the list (tab-separated): serial code, total amount and currency, invoice date, party name. `status` and `settings` print `Key: value` lines. Connecting the SPV happens on the web

### company
Show company profile.
```bash
solo-cli company
```
Output: labeled fields -- Name, CUI, Reg (registration number), Address, CAEN codes, VIES status, one `Bank:` line per bank account

### upload <file>
Upload an expense document (PDF or image).
```bash
solo-cli upload invoice.pdf
solo-cli up invoice.pdf   # Alias
```
Output: uploaded filename and confirmation that the document was added to the expense queue. `-r` / `--revenue` uploads a revenue document instead

### documents
Filed tax declarations and Documente PFA
```bash
solo-cli documents                       # Declarations: type, period, filed date, code (alias: docs)
solo-cli documents pfa                   # PFA files: section, code, file name, modified date
solo-cli documents download <code>       # Save one (alias: dl), -o path, -f to overwrite
```

### report [year]
Save the cash flow register (registrul de incasari si plati) PDF
```bash
solo-cli report                 # Current year, Registru-incasari-plati-YYYY.pdf
solo-cli report 2025 -o ~/Desktop
```

### subscription, referral
```bash
solo-cli subscription     # Plan, status, price, next billing, card, manage link
solo-cli referral         # Referral link, prizes, earned and pending rewards
```

### clients [search]
```bash
solo-cli clients          # alias: customers
solo-cli clients acme     # Server-side search
```
Output (tab-separated): client id, name, CUI/CNP, registration number, email, phone

### rates [YYYY-MM-DD]
```bash
solo-cli rates
solo-cli rates 2026-01-05
```
Output (tab-separated): currency, BNR rate (4 decimals), rate date

### taxes [year]
Show tax breakdown with CAS, CASS, and income tax calculations, bracket labels, buffer to next threshold, and surplus hints.
```bash
solo-cli taxes            # Current year
solo-cli taxes 2025       # Specific year
solo-cli tax              # Alias
```
Output includes: net income (in RON and as multiples of salariu minim brut), CAS amount and bracket, CASS amount and bracket, income tax, total taxes, net after tax, effective rate. A `SOLO Estimate` section follows with SOLO.ro's own figures for the same year and marks any line that differs from the local calculator

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

## Command flags

These go after the subcommand, for example `solo-cli queue delete 42 -y`. Before the command (`solo-cli -y queue delete 42`) they fail with "Unknown command"

| Flag | Short | Description |
|------|-------|-------------|
| --yes | -y | Confirm a mutating command without a terminal |
| --output | -o | Download path or directory |
| --force | -f | Overwrite an existing download |

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

# Delete a queued item (after the user confirmed it)
solo-cli queue delete 123456 -y

# Invoice PDF and e-Factura state of the newest invoice
code=$(solo-cli revenues | head -1 | cut -f5)
solo-cli invoice download "$code" -o ~/Downloads
solo-cli efactura status "$code"

# Unpaid invoices
solo-cli revenues | awk -F'\t' '$3 == "UNPAID"'

# Tax breakdown for a past year
solo-cli taxes 2025
```

## Authentication flow
1. On startup, loads cookies from `~/.config/solo-cli/cookies.json`
2. Validates cookies with a test API call
3. If valid, uses cached session
4. If invalid or missing, signs in on cont.solo.ro with credentials from config (2FA code prompt on a terminal only)
5. Saves new cookies for next session
6. Company ID is auto-discovered after authentication

## Troubleshooting
- **"credentials missing"**: Edit config.json with your SOLO.ro username and password
- **"authentication failed"**: Check credentials are correct
- **"two-factor authentication ... run solo-cli once in a terminal"**: ask the user to run any command once in a terminal and enter the SMS code
- **"needs confirmation, pass --yes (-y)"**: a mutating command ran without a terminal, confirm with the user before adding `-y`
- **"already exists, pass --force (-f)"**: pick another `-o` path or add `-f` if overwriting is fine
- **"invalid JSON in config"**: Fix syntax errors in config.json
- **Company info not showing**: Company ID is auto-discovered; clear cookies and log in again

## Reference
See `references/help-man-page.md` for the full help text, detailed output formats, taxes.json schema, threshold bracket documentation, and surplus hint logic.
