solo-cli - SOLO.ro accounting platform CLI

Usage:
  solo-cli [options] [command] [args]

Overview:
  summary [year]        Account summary, active alerts follow on stderr
  taxes [year]          Local tax breakdown with thresholds plus SOLO's own estimate (alias: tax)
  declaration [year]    Declaratia Unica status, default last year (alias: du)
  alerts                Active account alerts (VAT threshold, CAEN update, procura) (alias: alert)

Revenues and invoices:
  revenues              List invoices, last column is the invoice code (aliases: revenue, rev)
  revenues queue        List revenue documents still being processed (aliases: queued, q)
  revenues rejected     List rejected revenue documents
  revenues delete <id> [-y]
                        Delete a queued or rejected revenue document (aliases: del, rm)
  invoice <code>        Show an invoice with its lines (alias: inv)
  invoice paid <code> [YYYY-MM-DD] [-y]
                        Mark an invoice paid, default today
  invoice unpaid <code> [-y]
                        Mark an invoice unpaid
  invoice download <code> [--xml | --original] [-o path] [-f]
                        Save the invoice PDF, its e-Factura XML or the uploaded original (alias: dl)
  invoice delete <code> [-y]
                        Delete the last issued invoice (aliases: del, rm)

e-Factura:
  efactura              List received e-Factura documents (aliases: einvoice, ei)
  efactura status <invoice>
                        e-Factura status of an issued invoice (code or serial)
  efactura send <invoice> [-y]
                        Send an invoice left on manual sending to ANAF
  efactura expedite <invoice> [-y]
                        Send a scheduled e-Factura now
  efactura settings     SPV connection and sending preferences
  efactura settings set [--delay 0|2] [--contact-info yes|no] [-y]
                        Change the sending preferences, connecting the SPV happens on the web

Expenses:
  expenses              List expenses, last column is the document code if any (aliases: expense, exp)
  expenses rejected     List rejected expense documents with the ID queue delete takes
  expenses download <code> [-o path] [-f]
                        Save an expense, queued or rejected document (alias: dl)
  queue                 List the expense queue (alias: q)
  queue delete <id> [-y]
                        Delete a queued or rejected expense document (aliases: del, rm)
  upload [-r] <file>    Upload an expense document, -r (--revenue) for a revenue one (alias: up)

Documents:
  documents             List filed tax declarations (alias: docs)
  documents pfa         List the Documente PFA files
  documents download <code> [-o path] [-f]
                        Save a declaration or PFA document (alias: dl)
  report [year] [-o path] [-f]
                        Save the cash flow register PDF (registrul de incasari si plati, alias: reports)

Account:
  company               Company profile, VIES status and bank accounts
  subscription          SOLO.ro subscription, changes stay on the web
  referral              Referral link, prizes and rewards
  clients [text]        List clients, optionally filtered by a search text (alias: customers)
  rates [YYYY-MM-DD]    BNR exchange rates for a day, default today

Other:
  setup-skills          Install AI skills for Claude Code and other agents
  tui                   Start interactive TUI (default when no command)
  demo                  Start TUI with demo data (for screenshots)

Options:
  --config, -c          Path to custom config file
  help, -h              Show this help message
  version, -v           Show version

Command flags:
  --yes, -y             Confirm a destructive or outward-facing action, required without a terminal
  --output, -o          Path or directory for a download, default the current directory
  --force, -f           Overwrite an existing file when downloading

Config:
  Default: ~/.config/solo-cli/config.json

Examples:
  solo-cli                            # Start TUI
  solo-cli summary 2025               # Show 2025 summary
  solo-cli upload invoice.pdf         # Upload expense document
  solo-cli upload -r invoice.pdf      # Upload revenue document
  solo-cli queue delete 123 -y        # Delete a queued expense document
  solo-cli invoice paid <code> -y     # Mark an invoice paid today
  solo-cli invoice download <code>    # Save the invoice PDF
  solo-cli efactura status INV-001    # e-Factura status of an issued invoice
  solo-cli report 2025 -o ~/Desktop   # Save the 2025 cash flow register
  solo-cli -c ~/my-config.json rev    # Use custom config
  solo-cli expenses | grep -i "food"

## Extended Reference

### Output conventions

- Data output goes to **stdout** in tab-separated format suitable for piping to `grep`, `awk`, `cut`, etc
- Status and progress messages (login, uploading, downloading, errors, warnings) go to **stderr** so they do not pollute piped output
- `expenses` and `revenues` print a warning to stderr listing any rejected documents before printing the list to stdout
- Some lists gained a new **last** column (invoice code on `revenues`, document code on `expenses`, `queue` and the revenue document lists). Existing columns keep their positions, so `cut -f1-4` style scripts keep working
- Downloads write a file and print only its path on stdout

### Safety of mutating commands

These commands change data on SOLO.ro or at ANAF: `queue delete`, `revenues delete`, `invoice paid`, `invoice unpaid`, `invoice delete`, `efactura send`, `efactura expedite`, `efactura settings set` and `upload`

- Every one except `upload` confirms first: on a terminal it asks `[y/N]` on stderr, without a terminal it refuses with `... needs confirmation, pass --yes (-y) to run it without a terminal` unless `--yes` / `-y` is given
- Only pass `-y` after the user explicitly asked for that exact action. Prefer showing the item first (`invoice <code>`, `efactura status <invoice>`, `queue`) and confirming with the user
- `efactura send` submits the invoice to ANAF and cannot be undone. `invoice delete` permanently removes the newest invoice
- The delete commands only accept ids that the queue or rejected lists show, so a typo cannot delete a processed document

### Download flags

`invoice download`, `expenses download`, `documents download` and `report` share the same flags:
- `-o` / `--output <path>`: file path, or a directory to save into. Default is a sensible file name in the current directory
- `-f` / `--force`: overwrite an existing file. Without it the command refuses and exits 1

### Authentication flow

1. Loads cookies from `~/.config/solo-cli/cookies.json`
2. Validates cookies with a test API call
3. If valid, uses cached session (no login)
4. If invalid or missing, signs in on cont.solo.ro with the credentials from config and saves the new cookies. Accounts with two-factor authentication get a code prompt on a terminal and the device is remembered for 14 days. Without a terminal the CLI never triggers a code SMS, it stops and asks to run `solo-cli` once in a terminal
5. Company ID is auto-discovered from the authenticated session -- no config field required

### Config file (~/.config/solo-cli/config.json)

Created automatically on first run with empty values. Override location with `--config` / `-c`.

```json
{
  "username": "your_email@example.com",
  "password": "your_password",
  "page_size": 100,
  "user_agent": "Mozilla/5.0 ..."
}
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| username | Yes | -- | SOLO.ro login email |
| password | Yes | -- | SOLO.ro password |
| page_size | No | 100 | Items to fetch per API call |
| user_agent | No | Chrome UA | Custom HTTP User-Agent header |

### Tax config file (~/.config/solo-cli/taxes.json)

Auto-generated on first use of any taxes feature. Safe to edit manually.

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

**Important -- January-1 rule**: `salariu_minim_brut` must be set to the SMB in effect on January 1 of the income year. The Codul Fiscal explicitly pegs CAS/CASS plafoane to that value and ignores mid-year raises. For 2026 income the correct value is **4050 RON** (the July 2026 raise to 4325 does not apply to 2026; it first matters for 2027).

Each threshold entry has `min_salaries`, `max_salaries`, `base_salaries`, and `label`:
- `base_salaries = 0` -- exempt
- `base_salaries = -1` -- proportional (contribution is calculated on actual net income)
- `base_salaries > 0` -- fixed multiple of SMB (capped base)

### taxes command output format

```
Tax Breakdown (2026)
══════════════════════════════════════════
Total Revenues:       XX,XXX.XX RON
Deductible Expenses:  X,XXX.XX RON
Net Income:           XX,XXX.XX RON
  (X.X salarii minime brute)

CAS (25%): <bracket label>
  Base: XX,XXX.XX RON → Amount: X,XXX.XX RON
  Buffer: X,XXX.XX RON rămas până la următorul plafon (<next label>)

CASS (10%): <bracket label>
  Base: XX,XXX.XX RON → Amount: X,XXX.XX RON
  Buffer: X,XXX.XX RON rămas până la următorul plafon (<next label>)

Income Tax (10%): X,XXX.XX RON
  Base: Net Income - CAS - CASS = XX,XXX.XX RON

══════════════════════════════════════════
Total Taxes:          XX,XXX.XX RON
Net After Tax:        XX,XXX.XX RON
Effective Tax Rate:   XX.X%
```

**Threshold buffer**: shows how much additional net income remains before crossing into the next (more expensive) bracket

**Surplus hint**: when net income has already crossed a bracket boundary, the buffer line is replaced by a surplus hint showing how much in additional deductible expenses would drop the taxpayer back into the cheaper bracket -- for example:
```
  Surplus: X,XXX.XX RON (adaugă cheltuieli pentru a coborî sub plafon) (→ <prev label>)
```
The hint is only shown when the contribution saving from dropping a bracket exceeds the required expense amount (it fires for CAS; it is suppressed for CASS when dropping a bracket would be a net loss)

### SOLO estimate section of taxes

After the local breakdown, `taxes` asks SOLO.ro for its own estimate of the same year and prints it with the same alignment:

```
SOLO Estimate (2026)
══════════════════════════════════════════
Taxable Income:       XX,XXX.XX RON
CAS:                  X,XXX.XX RON
CASS:                 X,XXX.XX RON
Income Tax:           X,XXX.XX RON
Total Taxes:          X,XXX.XX RON
Matches the local breakdown
```

A line that differs from the local calculator by 1 RON or more gets `(local X RON, diff +N.NN)` appended and the last line becomes `Differs from the local breakdown, check taxes.json and the revenues and expenses SOLO counts`. `SOLO has no estimate for this year` is printed when SOLO has no figures. If the estimate call fails a warning goes to stderr and the exit code stays 0

### declaration command

```bash
solo-cli declaration        # Declaratia Unica for last year (filed this year)
solo-cli declaration 2025   # A specific fiscal year
solo-cli du                 # Alias
```

```
Declaratia Unica 2025 (filed in 2026)
══════════════════════════════════════════
Status:               Filed at ANAF by SOLO (DeclarationAutoFiledOk)
Last Completed:       YYYY-MM-DD

CAS:                  X,XXX.XX RON
CASS:                 X,XXX.XX RON
  of which other income: X.XX RON
Income Tax:           X,XXX.XX RON
Income Tax Discount:  -X.XX RON
Total Taxes:          X,XXX.XX RON
Total After Discount: X,XXX.XX RON

Deadline:             25 May 2026 (in N days)
File or review at https://falcon.solo.ro/tax-declaration/dashboard
```

The `of which other income`, discount and `Last Completed` lines only appear when they apply. An unfiled year prints `Taxes:                not computed yet`. The deadline suffix is `in N days`, `tomorrow`, `today` or `passed`, counted in Bucharest time. Filing itself happens on the web

### alerts command

Tab-separated, one line per active alert: kind (`vat`, `caen` or `procura`), title, message, web page to open. With no active alert stdout stays empty and `No alerts` goes to stderr. Each of the three status calls is best effort, a failing one prints `Warning: ...` on stderr and only all three failing exits 1

`summary` loads the same alerts after its data and prints each active one on stderr as `Alert: <title>` followed by indented message and URL lines, so its stdout is unchanged

### revenues command

```bash
solo-cli revenues                 # Invoices
solo-cli revenues queue           # Uploaded revenue documents still processing (aliases: queued, q)
solo-cli revenues rejected        # Rejected revenue documents
solo-cli revenues delete <id> -y  # Delete a queued or rejected revenue document (aliases: del, rm)
```

- `revenues` columns: serial code, total and currency, PAID or UNPAID, client name, invoice code. The invoice code is what `invoice` and `efactura` subcommands take. Rejected revenue documents are listed as a warning on stderr first
- `revenues queue` columns: document name, days pending, OVERDUE or blank, `(ID: n)`, document code when set
- `revenues rejected` columns: document name, reason, `(ID: n)`, document code when set
- `revenues delete` only accepts ids shown by the queue or rejected lists and prints `Document deleted`

### invoice command

```bash
solo-cli invoice <code>                         # Header and line items (also: invoice show|view <code>, alias: inv)
solo-cli invoice paid <code> [YYYY-MM-DD] -y    # Mark paid, default today
solo-cli invoice unpaid <code> -y               # Mark unpaid
solo-cli invoice download <code>                # Invoice PDF (alias: dl)
solo-cli invoice download <code> --xml          # e-Factura XML
solo-cli invoice download <code> --original     # Uploaded original of an external invoice
solo-cli invoice delete <code> -y               # Delete the last issued invoice (aliases: del, rm)
```

`<code>` is the invoice code from the last column of `revenues`. `invoice <code>` prints `Key: value` lines (Invoice, Code, Client, Issued, Due, Status, Payment, e-Factura, Amount, VAT, Total and Local total with the exchange rate for foreign currency invoices), a blank line, then one tab-separated line per item: name, quantity and unit, unit price, VAT percent, line total and currency

- `paid` refuses future dates, dates before the issue date and cancelled invoices, prints `Invoice <serial> marked paid on <date>`
- `unpaid` refuses invoices that are not paid, prints `Invoice <serial> marked unpaid`
- `delete` asks SOLO.ro first and only continues when the invoice is deletable (only the newest invoice is, older ones can only be voided on the web), prints `Invoice <serial> deleted`
- `download` checks which files exist first. Default names are `Factura-<serial>.pdf` and `e-Factura-<serial>.xml`. An uploaded invoice without a SOLO PDF falls back to its original file

### efactura command

```bash
solo-cli efactura                           # Received e-Factura documents (aliases: einvoice, ei, also efactura list|ls)
solo-cli efactura status <invoice>          # e-Factura state of an issued invoice
solo-cli efactura send <invoice> -y         # Send an invoice left on manual sending to ANAF
solo-cli efactura expedite <invoice> -y     # Send a scheduled e-Factura now
solo-cli efactura settings                  # SPV connection and sending preferences
solo-cli efactura settings set --delay 0 -y # Change preferences (--delay 0|2, --contact-info yes|no)
```

`<invoice>` is the invoice code or its serial. The list columns are serial code, total and currency, invoice date, party name

- `status` prints `Key: value` lines: Invoice, Code, Client, `Status: <name> (<code>)`, Meaning, `Managed by SOLO: yes|no`, then depending on the state the queue status, meaning, send date and a send now hint (queued), the ANAF upload index (sent), the error reason (error) or a hint to run `send` (skipped)
- `send` only works for invoices left on manual sending that SOLO manages. Submitting to ANAF cannot be undone
- `expedite` only works for a queued e-Factura that SOLO allows to send early
- `settings` prints SPV connected, Delegated access, Personal access, Sending delay and Contact info. Connecting the SPV is an ANAF sign in that only works in a browser, at https://falcon.solo.ro/dashboard
- `settings set` changes only the values given, skips the update when nothing changes and applies to every future invoice

### expenses command

```bash
solo-cli expenses                       # List expenses
solo-cli expenses rejected              # List rejected expense documents
solo-cli expenses download <code>       # Save the original document (alias: dl)
```

Columns: amount and currency, category, supplier name and, when SOLO.ro has one, the document code as a fourth column. Rejected expenses are listed as a stderr warning first, each with `(ID: n, document CODE)`. `expenses rejected` columns: document name, reason, `(ID: n)`, document code when set. `expenses download` takes a document code from `expenses`, `queue` or the rejected warning and saves `<code>.<ext>` by default

### queue command

```bash
solo-cli queue                  # List pending expense documents (alias: q)
solo-cli queue delete <id> -y   # Delete a queued or rejected expense document (aliases: del, rm)
```

Columns: document name, days pending, OVERDUE or blank, `(ID: n)` and the document code when set. `queue delete` only accepts ids shown by `queue` or `expenses rejected` (also in the rejected warning of `expenses`) and prints `Document deleted`

### upload command

```bash
solo-cli upload receipt.pdf       # Expense document (alias: up)
solo-cli upload -r invoice.pdf    # Revenue document (--revenue)
```

Accepts PDF files and images. Uploads in two steps (multipart upload then confirmation). Prints `Uploaded: <name>` and `Document added to <expense|revenue> queue for processing.`

### documents command

```bash
solo-cli documents                  # Filed tax declarations (alias: docs, also documents list)
solo-cli documents pfa              # Documente PFA files
solo-cli documents download <code>  # Save a declaration or PFA document (alias: dl)
```

- `documents` columns: declaration type, reporting period, filed date (YYYY-MM-DD), document code
- `documents pfa` columns: section title, document code, original file name, modified date
- `documents download` names the file after the original file name from those lists, falling back to the type and period or the code

### report command

```bash
solo-cli report                 # Cash flow register for the current year (alias: reports)
solo-cli report 2025 -o ~/Desktop
```

Saves the registrul de incasari si plati PDF as `Registru-incasari-plati-YYYY.pdf`. Years before the company start year or after the current one are refused before any download

### company command

Labeled fields: Name, CUI, Reg (registration number), Address, one `CAEN:` line per code (the principal one marked), `VIES status:`, `VIES code:` and `VIES request:` when set, then one `Bank: <bank> - <IBAN> (<currency>)` line per bank account. VIES and bank account failures only warn on stderr

### subscription command

Labeled fields: Plan (with billing period), Status (Active, Paused, Overdue, Cancelled), Price, Discount and Upcoming price when set, Next billing, Cancelled on, Scheduled change, Card (brand, last digits, expiry, EXPIRED flag), Overdue (only with an amount due), Hosted headquarters and the Manage link. Changes happen on the web

### referral command

Labeled fields: Link, Prize, Period (time limited campaigns), Earned and Pending. A note on stderr explains when payouts need a bank account

### clients command

```bash
solo-cli clients            # Every client (alias: customers)
solo-cli clients acme       # Server-side search
```

Columns: client id, name, CUI or CNP, registration number, email, phone. `No clients found` goes to stderr when the list is empty

### rates command

```bash
solo-cli rates              # BNR rates for today
solo-cli rates 2026-01-05   # For a specific day
```

Columns: currency code, rate with 4 decimals, rate date (YYYY-MM-DD)

### setup-skills command

Downloads and installs AI skill files to `~/.agents/skills/solo-cli/` and `~/.claude/skills/solo-cli/`. Run this once to enable solo-cli awareness in Claude Code and other agent tools. The TUI also prompts for this automatically on first launch.

### Troubleshooting

- **"credentials missing"**: Edit config.json with your SOLO.ro username and password
- **"authentication failed"**: Check that credentials are correct
- **"two-factor authentication ... run solo-cli once in a terminal"**: the saved session expired on an account with 2FA. Run any command once in a terminal and enter the code, later runs reuse the remembered device for 14 days
- **"needs confirmation, pass --yes (-y)"**: a destructive command ran without a terminal, pass `-y` once you are sure
- **"already exists, pass --force (-f)"**: a download would overwrite a file, pass `-f` or pick another path with `-o`
- **"invalid JSON in config"**: Fix syntax errors in config.json
- **Company info not showing**: Company ID is auto-discovered; try clearing cookies and logging in again
