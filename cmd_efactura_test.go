package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
)

// efacturaMock serves the issued invoice e-Factura endpoints and proxies
// everything else (login, summary) to the shared e2e mock
type efacturaMock struct {
	api          *mockAPI
	server       *httptest.Server
	sendHits     atomic.Int32
	expediteHits atomic.Int32
	updateHits   atomic.Int32
	updateBody   atomic.Value
	sendStatus   atomic.Value
}

const efacturaRevenues = `{"Items":[
	{"UniqueCode":"code-q","SerialCode":"INV-001","ClientName":"ACME Corp","EInvoiceStatus":{"Code":"queued","Name":"În așteptare","IsEligible":true,"EnableCloudFormation":true}},
	{"UniqueCode":"code-s","SerialCode":"INV-002","ClientName":"Globex","EInvoiceStatus":{"Code":"sent","Name":"Trimisă","IsEligible":true,"EnableCloudFormation":true}},
	{"UniqueCode":"code-e","SerialCode":"INV-003","ClientName":"Initech","EInvoiceStatus":{"Code":"error","Name":"Trimitere nereușită","IsEligible":true,"EnableCloudFormation":true}},
	{"UniqueCode":"code-k","SerialCode":"INV-004","ClientName":"Umbrella","EInvoiceStatus":{"Code":"skipped","Name":"Transmitere manuală","IsEligible":true,"EnableCloudFormation":true}},
	{"UniqueCode":"code-n","SerialCode":"INV-005","ClientName":"Hooli","EInvoiceStatus":{"Code":"skipped","Name":"Transmitere manuală","IsEligible":true,"EnableCloudFormation":false}},
	{"UniqueCode":"code-x","SerialCode":"INV-006","ClientName":"Stark","EInvoiceStatus":null}
],"TotalResults":6}`

func newEFacturaMock(t *testing.T) *efacturaMock {
	t.Helper()
	m := &efacturaMock{api: newMockAPI(t)}
	m.sendStatus.Store("Queued")
	m.updateBody.Store("")
	target, _ := url.Parse(m.api.server.URL)
	proxy := httputil.NewSingleHostReverseProxy(target)

	const ei = "/proxy/accounting/e-invoice/"
	mux := http.NewServeMux()
	mux.Handle("/", proxy)
	mux.HandleFunc("/proxy/accounting/revenues/list", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, efacturaRevenues)
	})
	mux.HandleFunc(ei+"queue-status/code-q", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Status":"InvoiceSendingScheduled","SendOn":"2026-10-14","AllowExpeditedSending":true}`)
	})
	mux.HandleFunc(ei+"spv-attributes/code-s", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"IsUploaded":true,"UploadIndex":4000000001,"DownloadIndex":null}`)
	})
	mux.HandleFunc(ei+"error-reason/code-e", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Reason":"UnknownReason"}`)
	})
	mux.HandleFunc(ei+"manual-send/code-k", func(w http.ResponseWriter, r *http.Request) {
		m.sendHits.Add(1)
		fmt.Fprintf(w, `{"Status":%q}`, m.sendStatus.Load())
	})
	mux.HandleFunc(ei+"expedited-send/code-q", func(w http.ResponseWriter, r *http.Request) {
		m.expediteHits.Add(1)
		fmt.Fprint(w, `{"Ok":true}`)
	})
	mux.HandleFunc(ei+"settings", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"IsConnected":false,"HasPersonalAccess":false,"HasDelegatedAccess":true,"DelegatedAccessStatus":"Confirmed","PersonalAccessStatus":"Unavailable","SendingDelayDays":2,"SendContactInfo":false,"IsTransitioning":false}`)
	})
	mux.HandleFunc(ei+"settings/update", func(w http.ResponseWriter, r *http.Request) {
		m.updateHits.Add(1)
		body, _ := io.ReadAll(r.Body)
		m.updateBody.Store(string(body))
		fmt.Fprint(w, `{"Ok":true}`)
	})
	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)
	return m
}

func (m *efacturaMock) run(t *testing.T, e *env, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binPath, append([]string{"--config", e.configPath}, args...)...)
	ssoBase := strings.Replace(m.api.sso.URL, "127.0.0.1", "localhost", 1)
	cmd.Env = append(os.Environ(), "HOME="+e.home, "SOLO_API_BASE="+m.server.URL, "SOLO_SSO_BASE="+ssoBase)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running binary: %v", err)
	}
	return stdout.String(), stderr.String(), code
}

func TestEFacturaStatus(t *testing.T) {
	m := newEFacturaMock(t)
	e := newEnv(t, "good-password")

	cases := map[string][]string{
		"code-q":  {"Invoice: INV-001\n", "Status: În așteptare (queued)\n", "Queue status: InvoiceSendingScheduled\n", "Send on: 2026-10-14\n", "Send now: available, run solo-cli efactura expedite code-q\n"},
		"INV-002": {"Code: code-s\n", "Meaning: Sent to ANAF\n", "Upload index: 4000000001\n"},
		"code-e":  {"Error reason: UnknownReason\n", "Reason meaning: ANAF gave no extra details"},
		"code-k":  {"Managed by SOLO: yes\n", "Send: run solo-cli efactura send code-k\n"},
		"code-n":  {"Status: Transmitere manuală (skipped)\n", "Managed by SOLO: no\n"},
		"INV-006": {"Status: none\n"},
	}
	for code, wants := range cases {
		out, errOut, exit := m.run(t, e, "efactura", "status", code)
		if exit != 0 {
			t.Errorf("status %s exit %d: %s", code, exit, errOut)
		}
		for _, want := range wants {
			if !strings.Contains(out, want) {
				t.Errorf("status %s missing %q in:\n%s", code, want, out)
			}
		}
	}

	_, errOut, exit := m.run(t, e, "efactura", "status", "nope")
	if exit != 1 || !strings.Contains(errOut, `no issued invoice with code or serial "nope"`) {
		t.Errorf("unknown invoice: exit %d, %q", exit, errOut)
	}
}

func TestEFacturaSendNeedsConfirmation(t *testing.T) {
	m := newEFacturaMock(t)
	e := newEnv(t, "good-password")

	_, errOut, exit := m.run(t, e, "efactura", "send", "INV-004")
	if exit != 1 || !strings.Contains(errOut, "--yes") || m.sendHits.Load() != 0 {
		t.Fatalf("send without --yes: exit %d, hits %d, %q", exit, m.sendHits.Load(), errOut)
	}

	out, errOut, exit := m.run(t, e, "efactura", "send", "INV-004", "--yes")
	if exit != 0 || out != "e-Factura for INV-004 queued, it will be sent to ANAF shortly\n" || m.sendHits.Load() != 1 {
		t.Fatalf("send: exit %d, hits %d, out %q, err %q", exit, m.sendHits.Load(), out, errOut)
	}

	m.sendStatus.Store("SpvAccessUnvailable")
	_, errOut, exit = m.run(t, e, "efactura", "send", "-y", "code-k")
	if exit != 1 || !strings.Contains(errOut, "not connected to the ANAF SPV") || !strings.Contains(errOut, "falcon.solo.ro/dashboard") {
		t.Errorf("SPV missing: exit %d, %q", exit, errOut)
	}

	for code, want := range map[string]string{"code-q": "only invoices left for manual sending", "code-n": "does not offer e-Factura sending"} {
		_, errOut, exit = m.run(t, e, "efactura", "send", code, "-y")
		if exit != 1 || !strings.Contains(errOut, want) {
			t.Errorf("send %s: exit %d, %q", code, exit, errOut)
		}
	}
	if m.sendHits.Load() != 2 {
		t.Errorf("refused sends must not reach the API, hits = %d", m.sendHits.Load())
	}
}

func TestEFacturaExpedite(t *testing.T) {
	m := newEFacturaMock(t)
	e := newEnv(t, "good-password")

	_, errOut, exit := m.run(t, e, "efactura", "expedite", "code-q")
	if exit != 1 || !strings.Contains(errOut, "instead of on 2026-10-14 needs confirmation, pass --yes") {
		t.Fatalf("expedite without --yes: exit %d, %q", exit, errOut)
	}
	out, _, exit := m.run(t, e, "efactura", "expedite", "INV-001", "-y")
	if exit != 0 || out != "e-Factura for INV-001 will be sent to ANAF shortly\n" || m.expediteHits.Load() != 1 {
		t.Fatalf("expedite: exit %d, hits %d, %q", exit, m.expediteHits.Load(), out)
	}
	_, errOut, exit = m.run(t, e, "efactura", "expedite", "code-s", "-y")
	if exit != 1 || !strings.Contains(errOut, "only queued e-Facturi") || m.expediteHits.Load() != 1 {
		t.Errorf("expedite sent invoice: exit %d, %q", exit, errOut)
	}
}

func TestEFacturaSettings(t *testing.T) {
	m := newEFacturaMock(t)
	e := newEnv(t, "good-password")

	out, errOut, exit := m.run(t, e, "efactura", "settings")
	want := "SPV connected: no\nDelegated access: yes (Confirmed)\nPersonal access: no (Unavailable)\n" +
		"Sending delay: 2 (within 2 days of issuing)\nContact info: no (not included in the e-Factura)\n"
	if exit != 0 || out != want {
		t.Errorf("settings: exit %d, out:\n%q\nwant:\n%q", exit, out, want)
	}
	if !strings.Contains(errOut, "falcon.solo.ro/dashboard") {
		t.Errorf("not connected must explain the web connection: %q", errOut)
	}

	_, errOut, exit = m.run(t, e, "efactura", "settings", "set", "--delay", "0")
	if exit != 1 || !strings.Contains(errOut, "--yes") || m.updateHits.Load() != 0 {
		t.Fatalf("set without --yes: exit %d, %q", exit, errOut)
	}
	out, errOut, exit = m.run(t, e, "efactura", "settings", "set", "--delay", "0", "-y")
	if exit != 0 || m.updateHits.Load() != 1 || !strings.HasPrefix(out, "e-Factura settings updated\n") {
		t.Fatalf("set delay: exit %d, out %q, err %q", exit, out, errOut)
	}
	if body := m.updateBody.Load(); body != `{"SendingDelayDays":0,"SendContactInfo":false}` {
		t.Errorf("update body = %s", body)
	}
	m.run(t, e, "efactura", "settings", "set", "--contact-info=yes", "--yes")
	if body := m.updateBody.Load(); body != `{"SendingDelayDays":2,"SendContactInfo":true}` {
		t.Errorf("update body = %s", body)
	}

	_, errOut, exit = m.run(t, e, "efactura", "settings", "set", "--delay", "2", "--contact-info", "no", "-y")
	if exit != 0 || !strings.Contains(errOut, "nothing to update") || m.updateHits.Load() != 2 {
		t.Errorf("unchanged settings: exit %d, hits %d, %q", exit, m.updateHits.Load(), errOut)
	}
}

func TestEFacturaArgumentErrors(t *testing.T) {
	m := newEFacturaMock(t)
	e := newEnv(t, "good-password")

	cases := map[string][]string{
		"unknown efactura subcommand":       {"efactura", "bogus"},
		"needs one invoice code":            {"efactura", "status"},
		"settings set needs --delay":        {"efactura", "settings", "set"},
		"--delay must be 0":                 {"efactura", "settings", "set", "--delay", "5"},
		"--contact-info must be yes or no":  {"efactura", "settings", "set", "--contact-info", "maybe"},
		`unknown option "--colour"`:         {"efactura", "settings", "set", "--colour", "red"},
		`unknown settings subcommand "get"`: {"efactura", "settings", "get"},
	}
	for want, args := range cases {
		_, errOut, exit := m.run(t, e, args...)
		if exit != 1 || !strings.Contains(errOut, want) {
			t.Errorf("%v: exit %d, %q", args, exit, errOut)
		}
	}
	if m.api.loginHits.Load()+m.api.ssoHits.Load() != 0 {
		t.Error("argument errors must fail before signing in")
	}

	out, _, exit := m.run(t, e, "efactura", "list")
	if exit != 0 || out != "EF-9\t500.00 RON\t2026-06-01\tTelecom SA\n" {
		t.Errorf("efactura list: exit %d, %q", exit, out)
	}
}
