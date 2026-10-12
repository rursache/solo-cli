package client

import (
	"fmt"
	"net/url"
	"strings"
)

// TaxDeclaration is a filed declaration (D212, D300, D390, D394 and others)
type TaxDeclaration struct {
	TaxDeclarationType       string `json:"TaxDeclarationType"`
	ReportingPeriod          string `json:"ReportingPeriod"`
	FiledOn                  string `json:"FiledOn"`
	UniqueCode               string `json:"UniqueCode"`
	MimeType                 string `json:"MimeType"`
	OriginalFileName         string `json:"OriginalFileName"`
	DocumentTypeCode         string `json:"DocumentTypeCode"`
	IsFiledExternally        bool   `json:"IsFiledExternally"`
	DeclarationFiledManually bool   `json:"DeclarationFiledManually"`
}

// TaxDeclarationsResponse is the tax-declarations payload
type TaxDeclarationsResponse struct {
	Documents []TaxDeclaration `json:"Documents"`
}

// RegistrationDocument is one file of a Documente PFA section
type RegistrationDocument struct {
	UniqueCode       string `json:"UniqueCode"`
	OriginalFileName string `json:"OriginalFileName"`
	DocumentTypeCode string `json:"DocumentTypeCode"`
	MimeType         string `json:"MimeType"`
	ModifiedOn       string `json:"ModifiedOn"`
}

// RegistrationSection groups the PFA registration papers of one type
type RegistrationSection struct {
	DocumentTypeCode     string                 `json:"DocumentTypeCode"`
	Documents            []RegistrationDocument `json:"Documents"`
	AllowMultipleUploads bool                   `json:"AllowMultipleUploads"`
	HasDocuments         bool                   `json:"HasDocuments"`
	ShowHints            bool                   `json:"ShowHints"`
}

// RegistrationDocumentsResponse is the registration-documents payload
type RegistrationDocumentsResponse struct {
	Sections []RegistrationSection `json:"Sections"`
}

var registrationSectionTitles = map[string]string{
	"trade_register":                 "Acte infiintare",
	"trade_certificate_eu":           "Certificat de TVA intracomunitar",
	"trade_certificate_vat":          "Certificat de TVA",
	"trade_register_change":          "Modificari PFA",
	"notarial_power_of_attorney":     "Procura notariala",
	"notarial_power_of_attorney_raw": "Procura notariala",
}

// Title is the section name the web shows, "Alte acte" for unknown codes
func (s RegistrationSection) Title() string {
	if t, ok := registrationSectionTitles[s.DocumentTypeCode]; ok {
		return t
	}
	return "Alte acte"
}

// ListTaxDeclarations fetches the filed tax declarations
func (c *Client) ListTaxDeclarations() ([]TaxDeclaration, error) {
	var result TaxDeclarationsResponse
	if err := c.doJSON("GET", "/api/documents/tax-declarations", "/documents", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to list tax declarations: %w", err)
	}
	return result.Documents, nil
}

// ListRegistrationDocuments fetches the Documente PFA sections, empty ones included
func (c *Client) ListRegistrationDocuments() ([]RegistrationSection, error) {
	var result RegistrationDocumentsResponse
	if err := c.doJSON("GET", "/api/documents/registration-documents", "/documents", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to list registration documents: %w", err)
	}
	return result.Sections, nil
}

// DownloadDocument fetches a tax declaration or PFA document by UniqueCode
func (c *Client) DownloadDocument(uniqueCode string) (*DownloadedFile, error) {
	code, err := pathCode(uniqueCode)
	if err != nil {
		return nil, err
	}
	return c.downloadNonEmpty("/api/documents/download/"+code, "/documents", "document")
}

// DownloadFinancialDocument fetches the original file of an expense, queued
// or rejected document by its DocumentCode
func (c *Client) DownloadFinancialDocument(documentCode string) (*DownloadedFile, error) {
	code, err := pathCode(documentCode)
	if err != nil {
		return nil, err
	}
	return c.downloadNonEmpty("/api/financial-documents/download/"+code, "/expenses", "document")
}

// downloadNonEmpty is download plus a check for the empty 200 body SOLO.ro
// answers with for an unknown document code
func (c *Client) downloadNonEmpty(path, referer, what string) (*DownloadedFile, error) {
	f, err := c.download(path, referer)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", what, err)
	}
	if len(f.Data) == 0 {
		return nil, fmt.Errorf("failed to download %s: SOLO.ro sent an empty file, check the code", what)
	}
	return f, nil
}

// pathCode validates a document code and escapes it for a URL path
func pathCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" || code == "." || code == ".." {
		return "", fmt.Errorf("missing document code")
	}
	return url.PathEscape(code), nil
}
