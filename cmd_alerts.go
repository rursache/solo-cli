package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"solo-cli/client"
)

func runAlertsCmd(args []string) {
	if len(args) > 0 {
		exitErr(fmt.Errorf("unexpected argument %q\nUsage: solo-cli alerts", args[0]))
	}
	withClient(runAlerts)
}

func runAlerts(c *client.Client) {
	a := c.GetAlerts()
	for _, e := range a.Errors {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", e)
	}
	if len(a.Errors) == 3 {
		exitErr(errors.New("could not load any alert status"))
	}
	writeAlerts(os.Stdout, os.Stderr, a.Active())
}

// writeAlerts prints one tab-separated line per alert: kind, title, message
// and the page to open. "No alerts" is a status, so it goes to errOut
func writeAlerts(out, errOut io.Writer, alerts []client.Alert) {
	if len(alerts) == 0 {
		fmt.Fprintln(errOut, "No alerts")
		return
	}
	for _, a := range alerts {
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", a.Kind, a.Title, a.Message, a.URL)
	}
}

// printSummaryAlerts shows active alerts on stderr after the summary data.
// Failures stay silent, the summary itself already succeeded
func printSummaryAlerts(c *client.Client) {
	writeSummaryAlerts(os.Stderr, c.GetAlerts().Active())
}

func writeSummaryAlerts(w io.Writer, alerts []client.Alert) {
	if len(alerts) == 0 {
		return
	}
	fmt.Fprintln(w)
	for _, a := range alerts {
		fmt.Fprintf(w, "Alert: %s\n  %s\n  %s\n", a.Title, a.Message, a.URL)
	}
}
