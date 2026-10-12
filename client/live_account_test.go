//go:build live

package client

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// Read-only live checks for documents, reports and account info. Downloads
// are GETs, nothing here creates, changes or sends anything

func liveCompany(t *testing.T) (*Client, string) {
	t.Helper()
	c := liveClient(t)
	id, err := c.DiscoverCompanyID()
	if err != nil {
		t.Fatalf("DiscoverCompanyID: %v", err)
	}
	return c, id
}

func rawGet(t *testing.T, c *Client, path, referer string) json.RawMessage {
	t.Helper()
	var raw json.RawMessage
	if err := c.doJSON("GET", path, referer, nil, &raw); err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return raw
}

func TestLiveSchemaDocuments(t *testing.T) {
	c := liveClient(t)

	var decl struct{ Documents []json.RawMessage }
	json.Unmarshal(rawGet(t, c, "/api/documents/tax-declarations", "/documents"), &decl)
	if len(decl.Documents) > 0 {
		assertSchema(t, decl.Documents[0], reflect.TypeOf(TaxDeclaration{}), "taxDeclaration")
	} else {
		t.Log("no declarations, item schema not checked")
	}

	var reg struct{ Sections []json.RawMessage }
	json.Unmarshal(rawGet(t, c, "/api/documents/registration-documents", "/documents"), &reg)
	if len(reg.Sections) == 0 {
		t.Fatal("registration-documents returned no sections")
	}
	assertSchema(t, reg.Sections[0], reflect.TypeOf(RegistrationSection{}), "registrationSection")
	for _, raw := range reg.Sections {
		var s struct{ Documents []json.RawMessage }
		json.Unmarshal(raw, &s)
		if len(s.Documents) > 0 {
			assertSchema(t, s.Documents[0], reflect.TypeOf(RegistrationDocument{}), "registrationDocument")
			break
		}
	}
}

func TestLiveDocumentDownloads(t *testing.T) {
	c := liveClient(t)

	if docs, err := c.ListTaxDeclarations(); err != nil {
		t.Fatal(err)
	} else if len(docs) > 0 {
		f, err := c.DownloadDocument(docs[0].UniqueCode)
		if err != nil {
			t.Fatalf("DownloadDocument: %v", err)
		}
		t.Logf("declaration: %s, %d bytes, .%s", f.ContentType, len(f.Data), f.Extension())
	}

	if _, err := c.DownloadDocument("doc_00000000000000000000000000000000"); err == nil {
		t.Error("unknown document code must fail")
	}

	expenses, err := c.ListExpenses(0, 20, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range expenses.Items {
		if e.DocumentCode == nil || *e.DocumentCode == "" {
			continue
		}
		f, err := c.DownloadFinancialDocument(*e.DocumentCode)
		if err != nil {
			t.Fatalf("DownloadFinancialDocument: %v", err)
		}
		t.Logf("expense document: %s, .%s", f.ContentType, f.Extension())
		break
	}
}

func TestLiveCashflowReport(t *testing.T) {
	c, id := liveCompany(t)

	if c.CompanyStartYear == 0 {
		t.Error("CompanyStartYear not found on the dashboard page")
	}
	f, err := c.DownloadCashflowReport(id, time.Now().Year())
	if err != nil {
		t.Fatalf("DownloadCashflowReport: %v", err)
	}
	if f.Extension() != "pdf" {
		t.Errorf("cash flow register is %s (.%s), want a PDF", f.ContentType, f.Extension())
	}
}

func TestLiveSchemaCompanyExtras(t *testing.T) {
	c, id := liveCompany(t)

	var env struct {
		Ok   bool
		Data json.RawMessage
	}
	json.Unmarshal(rawGet(t, c, "/proxy/accounting/company/vies-profile/company_"+id, "/settings"), &env)
	if !env.Ok {
		t.Log("VIES profile unavailable (Ok=false)")
	} else {
		assertSchema(t, env.Data, reflect.TypeOf(ViesProfile{}), "vies")
	}

	var accounts []json.RawMessage
	json.Unmarshal(rawGet(t, c, "/proxy/accounting/bank-accounts/company_"+id, "/settings"), &accounts)
	if len(accounts) > 0 {
		assertSchema(t, accounts[0], reflect.TypeOf(BankAccount{}), "bankAccount")
	} else {
		t.Log("no bank accounts, item schema not checked")
	}
}
