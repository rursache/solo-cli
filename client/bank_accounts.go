package client

import (
	"fmt"
	"strings"
)

// Bank is a bank from SOLO's list, IBANCode is the 4 letter code at IBAN
// positions 5 to 8
type Bank struct {
	Id       int    `json:"Id"`
	Name     string `json:"Name"`
	Code     string `json:"Code"`
	IBANCode string `json:"IBANCode"`
}

// BankAccount is an account printed on invoices. Bank is set for a bank from
// SOLO's list, BankName for a custom one
type BankAccount struct {
	Id       int       `json:"Id"`
	Bank     *Bank     `json:"Bank"`
	BankName *string   `json:"BankName"`
	IBAN     string    `json:"IBAN"`
	Currency *Currency `json:"Currency"`
}

// DisplayName is the listed bank name, else the custom one
func (a BankAccount) DisplayName() string {
	if a.Bank != nil && strings.TrimSpace(a.Bank.Name) != "" {
		return a.Bank.Name
	}
	if a.BankName != nil && strings.TrimSpace(*a.BankName) != "" {
		return *a.BankName
	}
	return "Unknown bank"
}

// CurrencyCode is the account currency code (RON, EUR), empty when missing
func (a BankAccount) CurrencyCode() string {
	if a.Currency == nil {
		return ""
	}
	if a.Currency.Code != "" {
		return strings.ToUpper(a.Currency.Code)
	}
	return strings.ToUpper(a.Currency.ShortName)
}

// ListBankAccounts fetches the company bank accounts
func (c *Client) ListBankAccounts(companyID string) ([]BankAccount, error) {
	if companyID == "" {
		return nil, fmt.Errorf("company ID not available")
	}
	var result []BankAccount
	if err := c.doJSON("GET", "/proxy/accounting/bank-accounts/company_"+companyID, "/settings", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to list bank accounts: %w", err)
	}
	return result, nil
}
