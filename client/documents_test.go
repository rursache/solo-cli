package client

import (
	"net/http"
	"strings"
	"testing"
)

func TestListTaxDeclarations(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/documents/tax-declarations" || !strings.HasSuffix(r.Header.Get("Referer"), "/documents") {
			t.Errorf("unexpected request: %s %s referer %s", r.Method, r.URL.Path, r.Header.Get("Referer"))
		}
		w.Write([]byte(`{"Documents":[{"TaxDeclarationType":"D212","ReportingPeriod":"2025","FiledOn":"2026-05-20T10:00:00+03:00","UniqueCode":"doc_1","MimeType":"application/pdf","OriginalFileName":"D212_test.pdf","DocumentTypeCode":"declaration_212","IsFiledExternally":false,"DeclarationFiledManually":false}]}`))
	}))

	docs, err := c.ListTaxDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].TaxDeclarationType != "D212" || docs[0].UniqueCode != "doc_1" || docs[0].OriginalFileName != "D212_test.pdf" {
		t.Errorf("declarations = %+v", docs)
	}
}

func TestListRegistrationDocuments(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/documents/registration-documents" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(`{"Sections":[
			{"DocumentTypeCode":"trade_register","Documents":[{"UniqueCode":"doc_a","OriginalFileName":"cert.pdf","DocumentTypeCode":"trade_register","MimeType":"application/pdf","ModifiedOn":"2024-01-10T09:00:00+02:00"}],"AllowMultipleUploads":true,"HasDocuments":true,"ShowHints":false},
			{"DocumentTypeCode":"mystery","Documents":[],"AllowMultipleUploads":false,"HasDocuments":false,"ShowHints":true}
		]}`))
	}))

	sections, err := c.ListRegistrationDocuments()
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) != 2 || len(sections[0].Documents) != 1 || sections[0].Documents[0].UniqueCode != "doc_a" {
		t.Fatalf("sections = %+v", sections)
	}
	if sections[0].Title() != "Acte infiintare" || sections[1].Title() != "Alte acte" {
		t.Errorf("titles = %q, %q", sections[0].Title(), sections[1].Title())
	}
}

func TestDocumentDownloads(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		referer := r.Header.Get("Referer")
		switch r.URL.EscapedPath() {
		case "/api/documents/download/doc_1":
			if !strings.HasSuffix(referer, "/documents") {
				t.Errorf("documents referer = %s", referer)
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Write([]byte("%PDF-1.4 declaration"))
		case "/api/financial-documents/download/doc%2F2":
			if !strings.HasSuffix(referer, "/expenses") {
				t.Errorf("financial referer = %s", referer)
			}
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
		case "/api/documents/download/missing":
			// SOLO.ro answers an unknown code with an empty 200
		default:
			t.Errorf("unexpected path %s", r.URL.EscapedPath())
			http.NotFound(w, r)
		}
	}))

	f, err := c.DownloadDocument("doc_1")
	if err != nil || f.Extension() != "pdf" {
		t.Errorf("DownloadDocument: %v, %+v", err, f)
	}
	f, err = c.DownloadFinancialDocument("doc/2")
	if err != nil || f.Extension() != "jpg" {
		t.Errorf("DownloadFinancialDocument must escape the code: %v, %+v", err, f)
	}
	if _, err := c.DownloadDocument("missing"); err == nil || !strings.Contains(err.Error(), "empty file") {
		t.Errorf("empty body must fail, got %v", err)
	}
	for _, code := range []string{"", "  ", ".."} {
		if _, err := c.DownloadDocument(code); err == nil {
			t.Errorf("code %q must be rejected", code)
		}
	}
}
