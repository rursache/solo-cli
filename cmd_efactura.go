package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"solo-cli/client"
)

const spvWebHint = "connecting the ANAF SPV is an ANAF sign in that only works in a browser, open https://falcon.solo.ro/dashboard to do it"

const efacturaUsage = `Usage:
  solo-cli efactura                         List received e-Factura documents
  solo-cli efactura status <invoice>        e-Factura status of an issued invoice
  solo-cli efactura send <invoice> [-y]     Send an invoice left on manual sending to ANAF
  solo-cli efactura expedite <invoice> [-y] Send a scheduled e-Factura now
  solo-cli efactura settings                SPV connection and sending preferences
  solo-cli efactura settings set [--delay 0|2] [--contact-info yes|no] [-y]
<invoice> is the invoice unique code or its serial (e.g. INV-001)`

// settingsChange holds the values passed to "efactura settings set", nil when not given
type settingsChange struct {
	delay       *int
	contactInfo *bool
}

// runEFacturaCmd validates the subcommand before signing in, then runs it
func runEFacturaCmd(args []string) {
	flags, err := parseCmdFlags(args)
	if err != nil {
		exitErr(err)
	}
	sub := ""
	if len(flags.Args) > 0 {
		sub = flags.Args[0]
	}

	switch sub {
	case "", "list", "ls":
		if len(flags.Args) > 1 {
			exitErr(fmt.Errorf("unexpected argument %q\n%s", flags.Args[1], efacturaUsage))
		}
		withPagedClient(runEFactura)
	case "status", "send", "expedite":
		if len(flags.Args) != 2 {
			exitErr(fmt.Errorf("efactura %s needs one invoice code\n%s", sub, efacturaUsage))
		}
		code := flags.Args[1]
		c, _ := setupClient()
		switch sub {
		case "status":
			err = efacturaStatus(c, code)
		case "send":
			err = efacturaSend(c, code, flags.Yes)
		case "expedite":
			err = efacturaExpedite(c, code, flags.Yes)
		}
	case "settings":
		rest := flags.Args[1:]
		if len(rest) == 0 {
			c, _ := setupClient()
			err = efacturaSettings(c)
			break
		}
		if rest[0] != "set" {
			exitErr(fmt.Errorf("unknown settings subcommand %q\n%s", rest[0], efacturaUsage))
		}
		change, perr := parseSettingsChange(rest[1:])
		if perr != nil {
			exitErr(fmt.Errorf("%v\n%s", perr, efacturaUsage))
		}
		c, _ := setupClient()
		err = efacturaSettingsSet(c, change, flags.Yes)
	case "help", "-h", "--help":
		fmt.Println(efacturaUsage)
	default:
		exitErr(fmt.Errorf("unknown efactura subcommand %q\n%s", sub, efacturaUsage))
	}
	if err != nil {
		exitErr(err)
	}
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func efacturaStatus(c *client.Client, code string) error {
	r, err := c.FindRevenueByCode(code)
	if err != nil {
		return err
	}
	fmt.Printf("Invoice: %s\n", r.SerialCode)
	fmt.Printf("Code: %s\n", r.UniqueCode)
	fmt.Printf("Client: %s\n", r.ClientName)
	if r.EInvoiceStatus == nil {
		fmt.Println("Status: none")
		fmt.Println("Managed by SOLO: no")
		return nil
	}
	kind := r.EInvoiceStatus.Kind()
	fmt.Printf("Status: %s (%s)\n", r.EInvoiceStatus.Name, r.EInvoiceStatus.Code)
	fmt.Printf("Meaning: %s\n", client.EInvoiceStatusMeaning(kind))
	tracked := r.TracksEInvoice()
	fmt.Printf("Managed by SOLO: %s\n", yesNo(tracked))
	if !tracked {
		return nil
	}

	switch kind {
	case client.EInvoiceQueued:
		q, err := c.GetEInvoiceQueueStatus(r.UniqueCode)
		if err != nil {
			return err
		}
		fmt.Printf("Queue status: %s\n", q.Status)
		fmt.Printf("Queue meaning: %s\n", q.Message())
		if q.SendOn != "" {
			fmt.Printf("Send on: %s\n", q.SendOn)
		}
		if q.CanExpedite() {
			fmt.Printf("Send now: available, run solo-cli efactura expedite %s\n", r.UniqueCode)
		}
		if q.Status == client.NoSpvAccess {
			fmt.Fprintf(os.Stderr, "Note: %s\n", spvWebHint)
		}
	case client.EInvoiceSent:
		s, err := c.GetEInvoiceSpvAttributes(r.UniqueCode)
		if err != nil {
			return err
		}
		if idx := s.DisplayIndex(); idx != "" {
			fmt.Printf("Upload index: %s\n", idx)
		} else {
			fmt.Println("Upload index: not confirmed by ANAF yet")
		}
	case client.EInvoiceError:
		e, err := c.GetEInvoiceErrorReason(r.UniqueCode)
		if err != nil {
			return err
		}
		fmt.Printf("Error reason: %s\n", e.Reason)
		fmt.Printf("Reason meaning: %s\n", e.Message())
	case client.EInvoiceSkipped:
		fmt.Printf("Send: run solo-cli efactura send %s\n", r.UniqueCode)
	}
	return nil
}

func efacturaSend(c *client.Client, code string, yes bool) error {
	r, err := c.FindRevenueByCode(code)
	if err != nil {
		return err
	}
	if kind := r.EInvoiceStatus.Kind(); kind != client.EInvoiceSkipped {
		return fmt.Errorf("invoice %s has e-Factura status %s, only invoices left for manual sending (skipped) can be sent", r.SerialCode, kind)
	}
	if !r.TracksEInvoice() {
		return fmt.Errorf("SOLO.ro does not offer e-Factura sending for invoice %s", r.SerialCode)
	}
	if !yes {
		fmt.Fprintln(os.Stderr, "Submitting an e-Factura to ANAF cannot be undone")
	}
	if err := confirm(fmt.Sprintf("Send invoice %s to ANAF through e-Factura?", r.SerialCode), yes); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Sending invoice %s to e-Factura...\n", r.SerialCode)
	err = c.SendEInvoice(r.UniqueCode)
	var sendErr *client.EInvoiceSendError
	if errors.As(err, &sendErr) && (sendErr.Status == "SpvAccessUnvailable" || sendErr.Status == "SpvAccessUnavailable") {
		return fmt.Errorf("%v, %s", err, spvWebHint)
	}
	if err != nil {
		return err
	}
	fmt.Printf("e-Factura for %s queued, it will be sent to ANAF shortly\n", r.SerialCode)
	return nil
}

func efacturaExpedite(c *client.Client, code string, yes bool) error {
	r, err := c.FindRevenueByCode(code)
	if err != nil {
		return err
	}
	if kind := r.EInvoiceStatus.Kind(); kind != client.EInvoiceQueued || !r.TracksEInvoice() {
		return fmt.Errorf("invoice %s has e-Factura status %s, only queued e-Facturi can be sent now", r.SerialCode, kind)
	}
	q, err := c.GetEInvoiceQueueStatus(r.UniqueCode)
	if err != nil {
		return err
	}
	if !q.CanExpedite() {
		return fmt.Errorf("the e-Factura of invoice %s cannot be sent now: %s (%s)", r.SerialCode, q.Message(), q.Status)
	}
	question := fmt.Sprintf("Send the e-Factura of invoice %s to ANAF now?", r.SerialCode)
	if q.SendOn != "" {
		question = fmt.Sprintf("Send the e-Factura of invoice %s to ANAF now instead of on %s?", r.SerialCode, q.SendOn)
	}
	if err := confirm(question, yes); err != nil {
		return err
	}
	if err := c.ExpediteEInvoice(r.UniqueCode); err != nil {
		return err
	}
	fmt.Printf("e-Factura for %s will be sent to ANAF shortly\n", r.SerialCode)
	return nil
}

func accessLine(has bool, status string) string {
	if status == "" {
		return yesNo(has)
	}
	return fmt.Sprintf("%s (%s)", yesNo(has), status)
}

func efacturaSettings(c *client.Client) error {
	s, err := c.GetEInvoiceSettings()
	if err != nil {
		return err
	}
	fmt.Printf("SPV connected: %s\n", yesNo(s.IsConnected))
	fmt.Printf("Delegated access: %s\n", accessLine(s.HasDelegatedAccess, s.DelegatedAccessStatus))
	fmt.Printf("Personal access: %s\n", accessLine(s.HasPersonalAccess, s.PersonalAccessStatus))
	if s.SendingDelayDays != nil {
		fmt.Printf("Sending delay: %d (%s)\n", *s.SendingDelayDays, client.SendingDelayLabel(*s.SendingDelayDays))
	} else {
		fmt.Println("Sending delay: unknown")
	}
	if s.SendContactInfo != nil {
		fmt.Printf("Contact info: %s (%s)\n", yesNo(*s.SendContactInfo), client.ContactInfoLabel(*s.SendContactInfo))
	} else {
		fmt.Println("Contact info: unknown")
	}
	if s.IsTransitioning {
		fmt.Println("Transitioning: yes")
	}
	if !s.IsConnected {
		fmt.Fprintf(os.Stderr, "Note: the SPV is not connected, %s\n", spvWebHint)
	}
	return nil
}

// parseSettingsChange reads --delay and --contact-info (also as --flag=value)
func parseSettingsChange(args []string) (settingsChange, error) {
	var ch settingsChange
	for i := 0; i < len(args); i++ {
		name, value, hasValue := strings.Cut(args[i], "=")
		if name != "--delay" && name != "--contact-info" {
			return ch, fmt.Errorf("unknown option %q", args[i])
		}
		if !hasValue {
			if i+1 >= len(args) {
				return ch, fmt.Errorf("%s needs a value", name)
			}
			i++
			value = args[i]
		}
		value = strings.ToLower(strings.TrimSpace(value))
		switch name {
		case "--delay":
			days, err := strconv.Atoi(value)
			if err != nil || (days != 0 && days != 2) {
				return ch, fmt.Errorf("--delay must be 0 (as soon as possible) or 2 (within 2 days), got %q", value)
			}
			ch.delay = &days
		case "--contact-info":
			var send bool
			switch value {
			case "yes", "y", "true", "da", "on":
				send = true
			case "no", "n", "false", "nu", "off":
			default:
				return ch, fmt.Errorf("--contact-info must be yes or no, got %q", value)
			}
			ch.contactInfo = &send
		}
	}
	if ch.delay == nil && ch.contactInfo == nil {
		return ch, errors.New("settings set needs --delay or --contact-info")
	}
	return ch, nil
}

func efacturaSettingsSet(c *client.Client, ch settingsChange, yes bool) error {
	current, err := c.GetEInvoiceSettings()
	if err != nil {
		return err
	}
	delay, contact := ch.delay, ch.contactInfo
	if delay == nil {
		if delay = current.SendingDelayDays; delay == nil {
			return errors.New("SOLO.ro did not return the current sending delay, pass --delay too")
		}
	}
	if contact == nil {
		if contact = current.SendContactInfo; contact == nil {
			return errors.New("SOLO.ro did not return the current contact info preference, pass --contact-info too")
		}
	}
	if current.SendingDelayDays != nil && *current.SendingDelayDays == *delay &&
		current.SendContactInfo != nil && *current.SendContactInfo == *contact {
		fmt.Fprintln(os.Stderr, "e-Factura settings already match, nothing to update")
		return nil
	}

	question := fmt.Sprintf("Update e-Factura settings for every future invoice (sending delay %d, contact info %s)?", *delay, yesNo(*contact))
	if err := confirm(question, yes); err != nil {
		return err
	}
	if err := c.UpdateEInvoiceSettings(*delay, *contact); err != nil {
		return err
	}
	fmt.Println("e-Factura settings updated")
	fmt.Printf("Sending delay: %d (%s)\n", *delay, client.SendingDelayLabel(*delay))
	fmt.Printf("Contact info: %s (%s)\n", yesNo(*contact), client.ContactInfoLabel(*contact))
	return nil
}
