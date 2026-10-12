//go:build live

package client

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Read-only live checks for documents, reports and account info. Downloads
// are GETs, nothing here creates, changes or sends anything

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
