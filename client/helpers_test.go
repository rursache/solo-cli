package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

type okData struct {
	Name string `json:"Name"`
}

func TestDoOkEnvelope(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Write([]byte(`{"Ok":true,"Data":{"Name":"ACME\u001b[2J"},"ErrorCode":null,"ErrorMessage":null}`))
		case "/ok-null":
			w.Write([]byte(`{"Ok":true,"Data":null}`))
		case "/fail":
			w.Write([]byte(`{"Ok":false,"Data":null,"ErrorCode":"InvoiceAlreadyPaid","ErrorMessage":"Factura este deja încasată"}`))
		case "/fail-bare":
			w.Write([]byte(`{"Ok":false}`))
		}
	}))

	var out okData
	if err := c.doOk("GET", "/ok", "/", nil, &out); err != nil || out.Name != "ACME[2J" {
		t.Errorf("doOk ok: %v, %+v (Data must be sanitized)", err, out)
	}

	var ptr *okData
	if err := c.doOk("GET", "/ok-null", "/", nil, &ptr); err != nil || ptr != nil {
		t.Errorf("doOk null Data: %v, %+v", err, ptr)
	}
	if err := c.doOk("POST", "/ok", "/", struct{}{}, nil); err != nil {
		t.Errorf("doOk without out: %v", err)
	}

	err := c.doOk("POST", "/fail", "/", struct{}{}, &out)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "InvoiceAlreadyPaid" || apiErr.Message != "Factura este deja încasată" {
		t.Fatalf("doOk fail = %v", err)
	}
	if err.Error() != "Factura este deja încasată (InvoiceAlreadyPaid)" {
		t.Errorf("error text = %q", err.Error())
	}
	if err := c.doOk("POST", "/fail-bare", "/", nil, nil); err == nil || err.Error() != "SOLO.ro rejected the request" {
		t.Errorf("bare Ok=false = %v", err)
	}
}

func TestOkResponseGenericDecode(t *testing.T) {
	var r OkResponse[*okData]
	if err := json.Unmarshal([]byte(`{"Ok":true,"Data":{"Name":"x"}}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.Err() != nil || r.Data == nil || r.Data.Name != "x" {
		t.Errorf("OkResponse decode: %+v", r)
	}
	code := "Busy"
	r = OkResponse[*okData]{ErrorCode: &code}
	if err := r.Err(); err == nil || !strings.Contains(err.Error(), "Busy") {
		t.Errorf("Err() = %v", err)
	}
}

func TestStatusResponse(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Status":"Queued"}`))
	}))
	var s StatusResponse
	if err := c.doJSON("GET", "/status", "/revenues", nil, &s); err != nil || s.Status != "Queued" {
		t.Errorf("StatusResponse: %v, %+v", err, s)
	}
}

func TestDownload(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "*/*" || r.Header.Get("User-Agent") != "test-agent" || !strings.HasSuffix(r.Header.Get("Referer"), "/expenses") {
			t.Errorf("download headers: %v", r.Header)
		}
		switch r.URL.Path {
		case "/pdf":
			w.Header().Set("Content-Type", "application/pdf")
			w.Write([]byte("%PDF-1.7 fake"))
		case "/octet":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("<?xml version=\"1.0\"?><Invoice/>"))
		case "/login-page":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte("<html>login</html>"))
		case "/sniffed-html":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("\n  <!DOCTYPE html><html></html>"))
		default:
			http.NotFound(w, r)
		}
	}))

	f, err := c.download("/pdf", "/expenses")
	if err != nil || string(f.Data) != "%PDF-1.7 fake" || f.ContentType != "application/pdf" || f.Extension() != "pdf" {
		t.Errorf("pdf download: %v, %+v", err, f)
	}
	if f, err := c.download("/octet", "/expenses"); err != nil || f.Extension() != "xml" {
		t.Errorf("octet download: %v, %+v", err, f)
	}
	for _, path := range []string{"/login-page", "/sniffed-html", "/missing"} {
		if _, err := c.download(path, "/expenses"); err == nil {
			t.Errorf("download %s must fail", path)
		}
	}
}

func TestFileExtension(t *testing.T) {
	for _, tc := range []struct {
		ct   string
		data string
		want string
	}{
		{"application/pdf", "", "pdf"},
		{"application/pdf; charset=binary", "", "pdf"},
		{"image/png", "", "png"},
		{"image/jpeg", "", "jpg"},
		{"image/jpg", "", "jpg"},
		{"image/heic", "", "heic"},
		{"text/xml; charset=utf-8", "", "xml"},
		{"application/zip", "", "zip"},
		{"application/x-zip-compressed", "", "zip"},
		{"", "%PDF-1.4", "pdf"},
		{"application/octet-stream", "PK\x03\x04", "zip"},
		{"application/octet-stream", "\x89PNG\r\n", "png"},
		{"", "\xFF\xD8\xFF\xE0", "jpg"},
		{"", "\xEF\xBB\xBF<?xml version=\"1.0\"?>", "xml"},
		{"", "  <Invoice xmlns=\"x\">", "xml"},
		{"", "<CreditNote>", "xml"},
		{"application/octet-stream", "random", "dat"},
		{"", "", "dat"},
	} {
		if got := FileExtension(tc.ct, []byte(tc.data)); got != tc.want {
			t.Errorf("FileExtension(%q, %q) = %q, want %q", tc.ct, tc.data, got, tc.want)
		}
	}
}
