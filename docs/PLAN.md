# Solo CLI Implementation Plan

Complete implementation roadmap for the SOLO.ro TUI CLI tool.

## Overview

A terminal-based interface for SOLO.ro accounting platform supporting:
- Dashboard summary view
- Revenue/expense management  
- Document upload workflow
- Expense queue processing

---

## Phase 1: Core Client Enhancements ✅ (DONE)

### Already Implemented
- [x] `config/config.go` - Config file at `~/.config/solo-cli/config.json`
- [x] `client/client.go` - Login + GetSummary API
- [x] `main.go` - Basic authentication flow

---

## Phase 2: API Client Expansion

### Revenue APIs

#### List Revenues
- **Endpoint**: `POST /proxy/accounting/revenues/list`
- **Request Body**:
```json
{
  "SearchText": "",
  "StartIndex": 0,
  "MaxResults": 30,
  "SortBy": "",
  "SortAsc": true,
  "InvoiceStatus": 1,
  "ElectronicInvoiceStatus": 0
}
```
- **Response Fields**: `UniqueCode`, `SerialCode`, `ClientName`, `IssueDate`, `PaymentDate`, `IsPaid`, `Total`, `Currency`, `Status`, `EInvoiceStatus`

#### Revenue Summary
- **Endpoint**: `GET /proxy/accounting/revenues/summary`
- **Query Parameters**: `?year=2025` (optional, defaults to current year if not specified)
- **Response**: `{"TotalAmount": 8921.34}`
- **Note**: Supports historical year queries for YoY comparison

---

### Expense APIs

#### List Expenses
- **Endpoint**: `POST /proxy/accounting/expenses/list`
- **Request Body**:
```json
{
  "SearchText": "",
  "StartIndex": 0,
  "MaxResults": 30,
  "SortBy": "",
  "SortAsc": true
}
```
- **Response Fields**: `UniqueCode`, `SupplierName`, `PurchaseDate`, `Category`, `Total`, `Deductibility`, `Currency`, `DocumentCode`, `DocumentMimeType`

#### Queued Expenses (Pending Processing)
- **Endpoint**: `POST /proxy/accounting/expenses/queued`
- **Same request body as expense list**
- **Response Fields**: `Id`, `DocumentCode`, `DocumentName`, `DocumentMimeType`, `CreatedOn`, `DaysPassed`, `ProcessingDeadline`, `IsOverdue`

#### Delete Expense
- **Endpoint**: `DELETE /proxy/accounting/expenses/{id}`
- **Response**: `null`

#### Expense Summary
- **Endpoint**: `GET /proxy/accounting/expenses/summary`
- **Response**: `{"TotalAmount": 0.0}`

---

### Document Upload (2-Step Process)

#### Step 1: Upload File
- **Endpoint**: `POST /api/local-storage/upload/{uuid}`
- **Content-Type**: `multipart/form-data`
- **UUID**: Generate random UUID for each upload
- **Response**: Filename string

#### Step 2: Confirm Upload
- **Endpoint**: `POST /api/financial-documents/save/expenses/{uuid}`
- **Request Body**: `{}`
- **Response**: `0` (success)

---

### Supporting APIs

#### Currencies
- **Endpoint**: `GET /proxy/accounting/currencies`
- **Response**: DefaultCurrency + AvailableCurrencies array (RON, EUR, USD, GBP)

#### E-Factura Settings
- **Endpoint**: `GET /proxy/accounting/e-factura/settings`
- **Response**: E-invoice configuration

#### Notarial Status
- **Endpoint**: `GET /proxy/accounting/notarial`
- **Response**: Notarial certification status

---

## Phase 3: TUI Implementation

### Recommended Libraries
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Tables**: [Bubble Table](https://github.com/charmbracelet/bubbles/tree/master/table)

### Main Views

1. **Dashboard** (default)
   - Year summary, revenues, expenses, taxes
   - Items awaiting review count

2. **Revenues List**
   - Table: Serial, Client, Date, Amount, Status
   - Pagination support

3. **Expenses List**
   - Table: Supplier, Date, Category, Amount, Deductibility
   - Pagination support

4. **Expense Queue**
   - Documents pending accountant review
   - Days passed, deadline, overdue indicator

5. **Upload View**
   - File picker for PDF/image selection
   - Progress indicator

### Navigation
- `Tab` / `Shift+Tab` - Switch views
- `j/k` or `↑/↓` - Navigate list
- `Enter` - Select/action
- `u` - Upload document
- `d` - Delete expense (with confirmation)
- `q` - Quit

---

## Phase 4: Implementation Files

### New Files to Create

```
solo-cli/
├── client/
│   ├── client.go        # (exists) add remaining API methods
│   ├── revenues.go      # Revenue-specific types and methods
│   ├── expenses.go      # Expense-specific types and methods
│   └── upload.go        # Document upload logic
├── tui/
│   ├── app.go           # Main Bubble Tea app
│   ├── dashboard.go     # Dashboard view
│   ├── revenues.go      # Revenues list view
│   ├── expenses.go      # Expenses list view
│   ├── queue.go         # Expense queue view
│   ├── upload.go        # Upload view
│   └── styles.go        # Lip Gloss styles
├── config/
│   └── config.go        # (exists)
└── main.go              # (exists) integrate TUI
```

---

## Implementation Order

1. **Client APIs** (extend `client/`)
   - Add revenue list/summary methods
   - Add expense list/summary/delete methods
   - Add expense queue method
   - Add document upload (multipart)

2. **TUI Foundation** (`tui/`)
   - Set up Bubble Tea app structure
   - Dashboard view with summary
   - Navigation between views

3. **Revenue/Expense Views**
   - Paginated tables
   - Refresh support

4. **Queue & Upload**
   - Queue list with overdue highlighting
   - File upload with picker

5. **Polish**
   - Error handling UI
   - Loading states
   - Keyboard shortcuts help
