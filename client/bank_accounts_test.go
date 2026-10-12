package client

import (
	"net/http"
	"testing"
)

func TestListBankAccounts(t *testing.T) {
	const id = "0123456789abcdef0123456789abcdef"
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/proxy/accounting/bank-accounts/company_"+id {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`[
			{"Id":1,"Bank":{"Id":12,"Name":"Test Bank","Code":"TB","IBANCode":"TEST"},"BankName":null,"IBAN":"RO00TEST0000000000000001","Currency":{"Id":1,"Code":"RON","Name":"Lei","ShortName":"lei","IsDefault":true}},
			{"Id":2,"Bank":null,"BankName":"Custom Bank","IBAN":"RO00CUST0000000000000002","Currency":{"Id":2,"Code":"EUR","Name":"Euro","ShortName":"eur","IsDefault":false}},
			{"Id":3,"Bank":null,"BankName":" ","IBAN":"RO00XXXX0000000000000003","Currency":null}
		]`))
	}))

	accounts, err := c.ListBankAccounts(id)
	if err != nil || len(accounts) != 3 {
		t.Fatalf("ListBankAccounts = %+v, %v", accounts, err)
	}
	for i, want := range []struct{ name, cur string }{{"Test Bank", "RON"}, {"Custom Bank", "EUR"}, {"Unknown bank", ""}} {
		if accounts[i].DisplayName() != want.name || accounts[i].CurrencyCode() != want.cur {
			t.Errorf("account %d = %q %q, want %q %q", i, accounts[i].DisplayName(), accounts[i].CurrencyCode(), want.name, want.cur)
		}
	}
	if _, err := c.ListBankAccounts(""); err == nil {
		t.Error("missing company ID must fail")
	}
}
