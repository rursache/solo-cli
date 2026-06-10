solo-cli - SOLO.ro accounting platform CLI

Usage:
  solo-cli [options] [command] [args]

Commands:
  summary [year]  Show account summary (year, revenues, expenses, taxes)
  taxes [year]    Show tax breakdown with thresholds (alias: tax)
  revenues        List revenue invoices (aliases: revenue, rev)
  expenses        List expenses (aliases: expense, exp)
  queue           List expense queue (alias: q). Subcommands: delete <id>
  efactura        List e-Factura documents (aliases: einvoice, ei)
  company         Show company profile
  upload <file>   Upload expense document (alias: up)
  setup-skills    Install AI skills for Claude Code and other agents
  tui             Start interactive TUI (default when no command)
  demo            Start TUI with demo data (for screenshots)

Options:
  --config, -c    Path to custom config file
  help, -h        Show this help message
  version, -v     Show version

Config:
  Default: ~/.config/solo-cli/config.json

Examples:
  solo-cli                          # Start TUI
  solo-cli summary                  # Show current year summary
  solo-cli summary 2025             # Show 2025 summary
  solo-cli upload invoice.pdf       # Upload expense document
  solo-cli queue delete 123         # Delete queued item
  solo-cli -c ~/my-config.json rev  # Use custom config
  solo-cli expenses | grep -i "food"

---

## Extended Reference

### Output conventions

- Data output goes to **stdout** in tab-separated format suitable for piping to `grep`, `awk`, `cut`, etc
- Status and progress messages (login, uploading, errors) go to **stderr** so they do not pollute piped output
- The `expenses` command prints a warning to stderr listing any rejected expenses before printing the expense list to stdout

### Authentication flow

1. Loads cookies from `~/.config/solo-cli/cookies.json`
2. Validates cookies with a test API call
3. If valid, uses cached session (no login prompt)
4. If invalid or missing, logs in with credentials from config and saves new cookies
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

### queue command

```bash
solo-cli queue              # List pending documents
solo-cli q                  # Alias
solo-cli queue delete <id>  # Delete a queued item by numeric ID
solo-cli queue del <id>     # Same (alias)
solo-cli queue rm <id>      # Same (alias)
```

Output columns: document name, days pending, overdue status (OVERDUE or blank), ID

### revenues output columns

Invoice serial code, total amount and currency, paid status (PAID or UNPAID), client name

### expenses output columns

Amount and currency, category, supplier name

### efactura output columns

Serial code, total amount and currency, invoice date, party name

### company output

Labeled fields: Name, CUI, Reg (registration number), Address

### upload command

Accepts PDF files and images. Uploads in two steps (multipart upload then confirmation). On success prints the processed filename and confirms the document was added to the expense queue for processing.

### setup-skills command

Downloads and installs AI skill files to `~/.agents/skills/solo-cli/` and `~/.claude/skills/solo-cli/`. Run this once to enable solo-cli awareness in Claude Code and other agent tools. The TUI also prompts for this automatically on first launch.

### Troubleshooting

- **"credentials missing"**: Edit config.json with your SOLO.ro username and password
- **"authentication failed"**: Check that credentials are correct
- **"invalid JSON in config"**: Fix syntax errors in config.json
- **Company info not showing**: Company ID is auto-discovered; try clearing cookies and logging in again
