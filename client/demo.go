package client

import "time"

// GetDemoSummary returns mock summary data for demo mode
func GetDemoSummary() *Summary {
	return &Summary{
		Year:                    time.Now().Year(),
		DisplayCurrency:         "RON",
		TotalRevenues:           125840.50,
		TotalDeductibleExpenses: 42350.75,
		HasTaxes:                true,
		Taxes:                   8349.88,
		RevenuesAwaitingReview:  0,
		ExpensesAwaitingReview:  3,
	}
}

// GetDemoCompany returns mock company data for demo mode
func GetDemoCompany() *CompanyInfo {
	return &CompanyInfo{
		Name:            "Demo PFA",
		Code1:           "RO12345678",
		Code2:           "J40/1234/2020",
		Address:         "Str. Tehnologiei 42, București, Sector 1",
		InvoiceMentions: "Operator de date cu caracter personal",
	}
}

// GetDemoRevenues returns mock revenue data for demo mode
func GetDemoRevenues() *RevenueListResponse {
	ronCurrency := Currency{Id: 1, Code: "RON", Name: "Romanian Leu", ShortName: "RON", IsDefault: true}
	eurCurrency := Currency{Id: 2, Code: "EUR", Name: "Euro", ShortName: "EUR", IsDefault: false}
	usdCurrency := Currency{Id: 3, Code: "USD", Name: "US Dollar", ShortName: "USD", IsDefault: false}

	items := []Revenue{
		{UniqueCode: "inv-001", SerialCode: "ACME-2025-001", ClientName: "Cloud Services Inc", IssueDate: "2025-01-05", PaymentDate: "2025-01-10", IsPaid: true, Total: 15000.00, Currency: eurCurrency},
		{UniqueCode: "inv-002", SerialCode: "ACME-2025-002", ClientName: "DevTools Pro SRL", IssueDate: "2025-01-08", PaymentDate: "", IsPaid: false, Total: 8500.00, Currency: ronCurrency},
		{UniqueCode: "inv-003", SerialCode: "ACME-2025-003", ClientName: "TechStart Solutions", IssueDate: "2025-01-12", PaymentDate: "2025-01-15", IsPaid: true, Total: 22400.00, Currency: ronCurrency},
		{UniqueCode: "inv-004", SerialCode: "ACME-2025-004", ClientName: "Nordic Systems AB", IssueDate: "2025-01-15", PaymentDate: "2025-01-20", IsPaid: true, Total: 5200.00, Currency: eurCurrency},
		{UniqueCode: "inv-005", SerialCode: "ACME-2025-005", ClientName: "DataFlow Analytics", IssueDate: "2025-01-18", PaymentDate: "", IsPaid: false, Total: 12750.00, Currency: usdCurrency},
		{UniqueCode: "inv-006", SerialCode: "ACME-2025-006", ClientName: "InnovateTech GmbH", IssueDate: "2025-01-22", PaymentDate: "2025-01-25", IsPaid: true, Total: 18900.00, Currency: eurCurrency},
		{UniqueCode: "inv-007", SerialCode: "ACME-2025-007", ClientName: "Quantum Labs SRL", IssueDate: "2025-01-25", PaymentDate: "", IsPaid: false, Total: 6300.00, Currency: ronCurrency},
		{UniqueCode: "inv-008", SerialCode: "ACME-2025-008", ClientName: "ByteForge Studios", IssueDate: "2025-01-28", PaymentDate: "2025-02-01", IsPaid: true, Total: 31500.00, Currency: ronCurrency},
	}

	total := len(items)
	return &RevenueListResponse{
		Items:        items,
		TotalResults: &total,
	}
}

// GetDemoExpenses returns mock expense data for demo mode
func GetDemoExpenses() *ExpenseListResponse {
	ronCurrency := Currency{Id: 1, Code: "RON", Name: "Romanian Leu", ShortName: "RON", IsDefault: true}
	eurCurrency := Currency{Id: 2, Code: "EUR", Name: "Euro", ShortName: "EUR", IsDefault: false}

	items := []Expense{
		{UniqueCode: "exp-001", SupplierName: "Adobe Systems", PurchaseDate: "2025-01-03", Category: "Software & Subscriptions", PrimaryCategory: "Operating", Total: 450.00, Currency: eurCurrency, Deductibility: "100%"},
		{UniqueCode: "exp-002", SupplierName: "DigitalOcean", PurchaseDate: "2025-01-05", Category: "Cloud Hosting", PrimaryCategory: "Operating", Total: 1250.00, Currency: ronCurrency, Deductibility: "100%"},
		{UniqueCode: "exp-003", SupplierName: "Petrom", PurchaseDate: "2025-01-08", Category: "Cheltuieli auto - Nedeductibilă", PrimaryCategory: "Transport", Total: 380.50, Currency: ronCurrency, Deductibility: "0%"},
		{UniqueCode: "exp-004", SupplierName: "eMAG", PurchaseDate: "2025-01-10", Category: "Office Equipment", PrimaryCategory: "Equipment", Total: 2890.00, Currency: ronCurrency, Deductibility: "100%"},
		{UniqueCode: "exp-005", SupplierName: "GitHub Enterprise", PurchaseDate: "2025-01-12", Category: "Software & Subscriptions", PrimaryCategory: "Operating", Total: 210.00, Currency: usdCurrency()},
		{UniqueCode: "exp-006", SupplierName: "Restaurant La Mama", PurchaseDate: "2025-01-15", Category: "Cheltuieli protocol - Nedeductibilă", PrimaryCategory: "Entertainment", Total: 520.00, Currency: ronCurrency, Deductibility: "0%"},
		{UniqueCode: "exp-007", SupplierName: "Telekom Romania", PurchaseDate: "2025-01-18", Category: "Telecommunications", PrimaryCategory: "Operating", Total: 189.00, Currency: ronCurrency, Deductibility: "100%"},
		{UniqueCode: "exp-008", SupplierName: "JetBrains", PurchaseDate: "2025-01-20", Category: "Software & Subscriptions", PrimaryCategory: "Operating", Total: 649.00, Currency: eurCurrency, Deductibility: "100%"},
	}

	total := len(items)
	return &ExpenseListResponse{
		Items:        items,
		TotalResults: &total,
	}
}

// helper for USD currency
func usdCurrency() Currency {
	return Currency{Id: 3, Code: "USD", Name: "US Dollar", ShortName: "USD", IsDefault: false}
}

// GetDemoQueue returns mock queue data for demo mode
func GetDemoQueue() *QueuedExpenseResponse {
	items := []QueuedExpense{
		{Id: 1, DocumentCode: "doc-001", DocumentName: "factura_orange_ian2025.pdf", DocumentMimeType: "application/pdf", CreatedOn: "2025-01-20", DaysPassed: 5, ProcessingDeadline: "2025-02-04", IsOverdue: false},
		{Id: 2, DocumentCode: "doc-002", DocumentName: "bon_fiscal_carburant.jpg", DocumentMimeType: "image/jpeg", CreatedOn: "2025-01-18", DaysPassed: 7, ProcessingDeadline: "2025-02-02", IsOverdue: false},
		{Id: 3, DocumentCode: "doc-003", DocumentName: "chitanta_curier_dec2024.pdf", DocumentMimeType: "application/pdf", CreatedOn: "2024-12-28", DaysPassed: 28, ProcessingDeadline: "2025-01-12", IsOverdue: true},
	}

	total := len(items)
	return &QueuedExpenseResponse{
		Items:        items,
		TotalResults: &total,
	}
}

// GetDemoRejectedExpenses returns mock rejected expense data for demo mode
func GetDemoRejectedExpenses() *RejectedExpenseResponse {
	items := []RejectedExpense{
		{Id: 101, DocumentCode: "doc_abc123", DocumentName: "factura_duplicat.pdf", DocumentMimeType: "application/pdf", Reason: "Vom procesa în curând echivalentul din e-Factura", AllowResubmit: false, CreatedOn: "2025-01-15T10:30:00+02:00", RejectedOn: "2025-01-17T14:00:00+02:00", DaysPassed: 2},
	}

	total := len(items)
	return &RejectedExpenseResponse{
		Items:        items,
		TotalResults: &total,
	}
}

// GetDemoEFactura returns mock e-factura data for demo mode
func GetDemoEFactura() *EFacturaListResponse {
	items := []EFactura{
		{SerialCode: "EF-2025-00142", TotalAmount: 1890.50, CurrencyCode: "RON", InvoiceDate: "2025-01-22", PartyCode1: "RO9876543", PartyName: "Supplier Alpha SRL"},
		{SerialCode: "EF-2025-00138", TotalAmount: 4250.00, CurrencyCode: "RON", InvoiceDate: "2025-01-20", PartyCode1: "RO1122334", PartyName: "Tech Imports SA"},
		{SerialCode: "EF-2025-00125", TotalAmount: 780.00, CurrencyCode: "EUR", InvoiceDate: "2025-01-18", PartyCode1: "RO5544332", PartyName: "Office Supplies Pro"},
		{SerialCode: "EF-2025-00119", TotalAmount: 2340.75, CurrencyCode: "RON", InvoiceDate: "2025-01-15", PartyCode1: "RO6677889", PartyName: "Logistics Express SRL"},
		{SerialCode: "EF-2025-00108", TotalAmount: 560.00, CurrencyCode: "RON", InvoiceDate: "2025-01-12", PartyCode1: "RO3344556", PartyName: "Cleaning Services Pro"},
	}

	total := len(items)
	return &EFacturaListResponse{
		Items:        items,
		TotalResults: &total,
	}
}
