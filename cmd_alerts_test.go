package main

import (
	"strings"
	"testing"

	"solo-cli/client"
)

var testAlerts = (&client.Alerts{
	VAT:  &client.VATAlertStatus{Status: "DangerPending", CriticalThresholdValue: 300000},
	CAEN: &client.CAENAlertStatus{Status: "Requested"},
}).Active()

func TestWriteAlerts(t *testing.T) {
	if len(testAlerts) != 2 {
		t.Fatalf("fixture alerts = %+v", testAlerts)
	}
	var out, errOut strings.Builder
	writeAlerts(&out, &errOut, testAlerts)
	want := "vat\tClose to the VAT threshold\tRevenues are getting close to the VAT exemption threshold of 300000 RON\thttps://falcon.solo.ro/vat\n" +
		"caen\tCAEN code update\tThe CAEN codes of your activity need updating, continue on the web\thttps://falcon.solo.ro/caen\n"
	if out.String() != want || errOut.Len() != 0 {
		t.Errorf("stdout:\n%q\nwant:\n%q\nstderr %q", out.String(), want, errOut.String())
	}

	out.Reset()
	writeAlerts(&out, &errOut, nil)
	if out.Len() != 0 || errOut.String() != "No alerts\n" {
		t.Errorf("no alerts: stdout %q, stderr %q", out.String(), errOut.String())
	}
}

func TestWriteSummaryAlerts(t *testing.T) {
	var w strings.Builder
	writeSummaryAlerts(&w, nil)
	if w.Len() != 0 {
		t.Errorf("no alerts must print nothing: %q", w.String())
	}
	writeSummaryAlerts(&w, testAlerts[1:])
	want := "\nAlert: CAEN code update\n  The CAEN codes of your activity need updating, continue on the web\n  https://falcon.solo.ro/caen\n"
	if w.String() != want {
		t.Errorf("got %q, want %q", w.String(), want)
	}
}
