package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"solo-cli/client"
)

const invoiceUsage = `Usage:
  solo-cli invoice <code>                           Show an invoice with its lines
  solo-cli invoice paid <code> [YYYY-MM-DD] [-y]    Mark paid (default today)
  solo-cli invoice unpaid <code> [-y]               Mark unpaid
  solo-cli invoice download <code> [--xml | --original] [-o path] [-f]
  solo-cli invoice delete <code> [-y]               Delete the last issued invoice
<code> is the invoice code, the last column of solo-cli revenues`

// runInvoice handles "invoice <code>" and its action subcommands. Arguments
// are checked before signing in
func runInvoice(args []string) {
	flags, err := parseCmdFlags(args)
	if err != nil {
		exitErr(err)
	}
	rest := flags.Args
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, invoiceUsage)
		os.Exit(1)
	}

	action := "show"
	switch rest[0] {
	case "show", "view":
		rest = rest[1:]
	case "paid", "unpaid", "download", "delete":
		action, rest = rest[0], rest[1:]
	case "dl":
		action, rest = "download", rest[1:]
	case "del", "rm":
		action, rest = "delete", rest[1:]
	}

	format := client.InvoiceFormatPDF
	if action == "download" {
		if format, rest, err = downloadFormat(rest); err != nil {
			invoiceUsageErr(err.Error())
		}
	}
	for _, a := range rest {
		if strings.HasPrefix(a, "-") {
			invoiceUsageErr("unknown flag " + a)
		}
	}
	maxArgs := 1
	if action == "paid" {
		maxArgs = 2
	}
	if len(rest) < 1 || len(rest) > maxArgs {
		invoiceUsageErr("expected one invoice code")
	}
	code := rest[0]
	if !client.ValidCode(code) {
		exitErr(fmt.Errorf("invalid invoice code %q, use the last column of solo-cli revenues", code))
	}

	var run func(*client.Client) error
	switch action {
	case "show":
		run = func(c *client.Client) error { return showInvoice(c, code) }
	case "paid":
		day, err := parsePaymentDay(rest[1:], time.Now())
		if err != nil {
			exitErr(err)
		}
		run = func(c *client.Client) error { return markInvoicePaid(c, code, day, flags.Yes) }
	case "unpaid":
		run = func(c *client.Client) error { return markInvoiceUnpaid(c, code, flags.Yes) }
	case "download":
		run = func(c *client.Client) error { return downloadInvoice(c, code, format, flags) }
	case "delete":
		run = func(c *client.Client) error { return deleteInvoice(c, code, flags.Yes) }
	}

	c, _ := setupClient()
	if err := run(c); err != nil {
		exitErr(err)
	}
}

func invoiceUsageErr(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n%s\n", msg, invoiceUsage)
	os.Exit(1)
}

// parsePaymentDay reads the optional YYYY-MM-DD argument, default today.
// Future dates are refused like the app's date picker does
func parsePaymentDay(args []string, now time.Time) (time.Time, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if len(args) == 0 {
		return today, nil
	}
	day, err := time.ParseInLocation("2006-01-02", args[0], time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q, use YYYY-MM-DD", args[0])
	}
	if day.After(today) {
		return time.Time{}, fmt.Errorf("the payment date %s is in the future", args[0])
	}
	return day, nil
}

// downloadFormat picks the format from --xml / --original and returns the
// remaining arguments
func downloadFormat(args []string) (string, []string, error) {
	format := client.InvoiceFormatPDF
	var rest []string
	for _, a := range args {
		switch a {
		case "--xml", "--original":
			if format != client.InvoiceFormatPDF {
				return "", nil, errors.New("pass only one of --xml and --original")
			}
			format = client.InvoiceFormatXML
			if a == "--original" {
				format = client.InvoiceFormatOriginal
			}
		default:
			if strings.HasPrefix(a, "-") {
				return "", nil, fmt.Errorf("unknown flag %s", a)
			}
			rest = append(rest, a)
		}
	}
	return format, rest, nil
}

func invoiceNumber(q float64) string {
	return strconv.FormatFloat(q, 'f', -1, 64)
}

func showInvoice(c *client.Client, code string) error {
	inv, err := c.GetInvoice(code)
	if err != nil {
		return err
	}
	cur := inv.Currency.ShortName
	paid := "UNPAID"
	if inv.IsPaid {
		paid = "PAID " + dateOnly(inv.PaymentDate)
	}

	fmt.Printf("Invoice: %s\n", inv.SerialCode)
	fmt.Printf("Code: %s\n", inv.UniqueCode)
	fmt.Printf("Client: %s\n", inv.ClientName)
	fmt.Printf("Issued: %s\n", dateOnly(inv.IssueDate))
	if inv.DueDate != "" {
		fmt.Printf("Due: %s\n", dateOnly(inv.DueDate))
	}
	if inv.Status != nil {
		status := inv.Status.Name
		if inv.Status.IsCancelled {
			status += " (cancelled)"
		}
		fmt.Printf("Status: %s\n", status)
	}
	fmt.Printf("Payment: %s\n", strings.TrimSpace(paid))
	if inv.EInvoiceStatus != nil && inv.EInvoiceStatus.Name != "" {
		fmt.Printf("e-Factura: %s\n", inv.EInvoiceStatus.Name)
	}
	fmt.Printf("Amount: %.2f %s\n", inv.Amount, cur)
	fmt.Printf("VAT: %.2f %s\n", inv.VAT, cur)
	fmt.Printf("Total: %.2f %s\n", inv.Total, cur)
	if la := inv.InvoiceLocalAmount; la != nil && la.Currency.ShortName != "" && la.Currency.ShortName != cur {
		fmt.Printf("Local total: %.2f %s", la.Total, la.Currency.ShortName)
		if la.ExchangeRate != nil {
			fmt.Printf(" (rate %.4f)", la.ExchangeRate.RateValue)
		}
		fmt.Println()
	}

	fmt.Println()
	for _, l := range inv.LineItems {
		unit := ""
		if l.MeasureUnit != nil {
			unit = " " + l.MeasureUnit.LabelRo
		}
		fmt.Printf("%s\t%s%s\t%.2f\t%s%%\t%.2f %s\n", l.ItemName, invoiceNumber(l.Quantity), unit, l.UnitPrice, invoiceNumber(l.VatPercentage), l.Total, cur)
	}
	return nil
}

// invoiceLabel is the serial of an invoice for messages, the code when the
// invoice could not be loaded
func invoiceLabel(inv *client.InvoiceDetail, code string) string {
	if inv != nil && inv.SerialCode != "" {
		return inv.SerialCode
	}
	return code
}

func markInvoicePaid(c *client.Client, code string, day time.Time, yes bool) error {
	inv, err := c.GetInvoice(code)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load the invoice, the issue date is not checked (%v)\n", err)
		inv = nil
	}
	if inv != nil {
		if inv.Status != nil && inv.Status.IsCancelled {
			return fmt.Errorf("invoice %s is cancelled", inv.SerialCode)
		}
		if issued, err := time.ParseInLocation("2006-01-02", dateOnly(inv.IssueDate), time.Local); err == nil && day.Before(issued) {
			return fmt.Errorf("%w, it was issued on %s", client.ErrInvalidPaymentDate, dateOnly(inv.IssueDate))
		}
	}

	label := invoiceLabel(inv, code)
	date := day.Format("2006-01-02")
	question := fmt.Sprintf("Mark invoice %s paid on %s?", label, date)
	if inv != nil && inv.IsPaid {
		question = fmt.Sprintf("Invoice %s is already paid on %s, change the payment date to %s?", label, dateOnly(inv.PaymentDate), date)
	}
	if err := confirm(question, yes); err != nil {
		return err
	}
	if _, err := c.MarkInvoicePaid(code, day); err != nil {
		return err
	}
	fmt.Printf("Invoice %s marked paid on %s\n", label, date)
	return nil
}

func markInvoiceUnpaid(c *client.Client, code string, yes bool) error {
	inv, err := c.GetInvoice(code)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load the invoice (%v)\n", err)
		inv = nil
	}
	if inv != nil && !inv.IsPaid {
		return fmt.Errorf("invoice %s is not marked paid", inv.SerialCode)
	}
	label := invoiceLabel(inv, code)
	if err := confirm(fmt.Sprintf("Mark invoice %s unpaid?", label), yes); err != nil {
		return err
	}
	if err := c.MarkInvoiceUnpaid(code); err != nil {
		return err
	}
	fmt.Printf("Invoice %s marked unpaid\n", label)
	return nil
}

func downloadInvoice(c *client.Client, code, format string, flags cmdFlags) error {
	opts, optsErr := c.GetInvoiceDownloadOptions(code)
	if optsErr == nil {
		// External invoices have no SOLO PDF, their file is the original
		if format == client.InvoiceFormatPDF && !opts.Has(client.InvoiceFormatPDF) && opts.Has(client.InvoiceFormatOriginal) {
			fmt.Fprintln(os.Stderr, "This invoice was uploaded, downloading the original file")
			format = client.InvoiceFormatOriginal
		}
		if !opts.Has(format) {
			switch format {
			case client.InvoiceFormatXML:
				return errors.New("this invoice has no e-Factura XML")
			case client.InvoiceFormatOriginal:
				return errors.New("this invoice has no uploaded original, it was issued in SOLO")
			}
			return errors.New("this invoice has no PDF")
		}
	}

	serial := code
	if inv, err := c.GetInvoice(code); err == nil && inv.SerialCode != "" {
		serial = inv.SerialCode
	}

	var f *client.DownloadedFile
	var err error
	base := "Factura-" + serial
	switch format {
	case client.InvoiceFormatXML:
		base = "e-Factura-" + serial
		f, err = c.DownloadInvoiceXML(code)
	case client.InvoiceFormatOriginal:
		if optsErr != nil {
			return optsErr
		}
		if opts.DocumentCode == nil || *opts.DocumentCode == "" {
			return errors.New("SOLO.ro did not return the document code of the original file")
		}
		f, err = c.DownloadInvoiceOriginal(*opts.DocumentCode)
	default:
		f, err = c.DownloadInvoicePDF(code)
	}
	if err != nil {
		return err
	}
	return saveDownload(f, base, flags)
}

func deleteInvoice(c *client.Client, code string, yes bool) error {
	pre, err := c.PreCancelInvoice(code)
	if err != nil {
		return err
	}
	if !pre.Deletable() {
		return fmt.Errorf("%w, SOLO.ro answered %s", client.ErrInvoiceNotDeletable, pre.Status)
	}
	inv, _ := c.GetInvoice(code)
	label := invoiceLabel(inv, code)
	if err := confirm(fmt.Sprintf("Permanently delete invoice %s?", label), yes); err != nil {
		return err
	}
	if err := c.CancelInvoice(code); err != nil {
		return err
	}
	fmt.Printf("Invoice %s deleted\n", label)
	return nil
}
