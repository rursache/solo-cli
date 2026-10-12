package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"solo-cli/client"
	"solo-cli/taxes"
)

var isoDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// dateOnly keeps the YYYY-MM-DD prefix of a server timestamp
func dateOnly(s string) string {
	if isoDateRe.MatchString(s) {
		return s[:10]
	}
	return s
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// lastColumn appends a document code as a new last tab-separated column,
// left out when the server has none so the existing columns stay as they were
func lastColumn(code string) string {
	if code == "" {
		return ""
	}
	return "\t" + code
}

func docCodeSuffix(code string) string {
	if code == "" {
		return ""
	}
	return " (document " + code + ")"
}

const documentsUsage = "Usage: solo-cli documents [pfa | download <code> [-o path] [-f]]"

func runDocuments(args []string) {
	flags, err := parseCmdFlags(args)
	if err != nil {
		exitErr(err)
	}
	sub := ""
	if len(flags.Args) > 0 {
		sub = flags.Args[0]
	}
	switch sub {
	case "", "list", "declarations":
		if len(flags.Args) > 1 {
			exitErr(fmt.Errorf("unexpected argument %q\n%s", flags.Args[1], documentsUsage))
		}
		c, _ := setupClient()
		listTaxDeclarations(c)
	case "pfa":
		c, _ := setupClient()
		listRegistrationDocuments(c)
	case "download", "dl":
		if len(flags.Args) != 2 {
			exitErr(fmt.Errorf("missing document code\n%s", documentsUsage))
		}
		c, _ := setupClient()
		downloadDocument(c, flags.Args[1], flags)
	default:
		exitErr(fmt.Errorf("unknown documents subcommand %q\n%s", sub, documentsUsage))
	}
}

func listTaxDeclarations(c *client.Client) {
	docs, err := c.ListTaxDeclarations()
	if err != nil {
		exitErr(err)
	}
	if len(docs) == 0 {
		fmt.Fprintln(os.Stderr, "No filed declarations")
		return
	}
	for _, d := range docs {
		fmt.Printf("%s\t%s\t%s\t%s\n", d.TaxDeclarationType, d.ReportingPeriod, dateOnly(d.FiledOn), d.UniqueCode)
	}
}

func listRegistrationDocuments(c *client.Client) {
	sections, err := c.ListRegistrationDocuments()
	if err != nil {
		exitErr(err)
	}
	found := false
	for _, s := range sections {
		for _, d := range s.Documents {
			found = true
			fmt.Printf("%s\t%s\t%s\t%s\n", s.Title(), d.UniqueCode, d.OriginalFileName, dateOnly(d.ModifiedOn))
		}
	}
	if !found {
		fmt.Fprintln(os.Stderr, "No PFA documents")
	}
}

// documentBaseName finds a file name for a document code in the declaration
// and PFA lists, falling back to the code itself
func documentBaseName(c *client.Client, code string) string {
	name := ""
	if docs, err := c.ListTaxDeclarations(); err == nil {
		for _, d := range docs {
			if d.UniqueCode == code {
				name = d.OriginalFileName
				if name == "" {
					name = d.TaxDeclarationType + " " + d.ReportingPeriod
				}
				break
			}
		}
	}
	if name == "" {
		if sections, err := c.ListRegistrationDocuments(); err == nil {
		search:
			for _, s := range sections {
				for _, d := range s.Documents {
					if d.UniqueCode == code {
						name = d.OriginalFileName
						break search
					}
				}
			}
		}
	}
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if strings.TrimSpace(name) == "" {
		return code
	}
	return name
}

func downloadDocument(c *client.Client, code string, flags cmdFlags) {
	fmt.Fprintf(os.Stderr, "Downloading document %s...\n", code)
	f, err := c.DownloadDocument(code)
	if err != nil {
		exitErr(err)
	}
	if err := saveDownload(f, documentBaseName(c, code), flags); err != nil {
		exitErr(err)
	}
}

// runExpensesCmd keeps "expenses" as the list and adds "expenses download"
func runExpensesCmd(args []string) {
	if len(args) == 0 || (args[0] != "download" && args[0] != "dl") {
		withClient(runExpenses)
		return
	}
	flags, err := parseCmdFlags(args[1:])
	if err != nil {
		exitErr(err)
	}
	if len(flags.Args) != 1 {
		exitErr(fmt.Errorf("missing document code\nUsage: solo-cli expenses download <DocumentCode> [-o path] [-f]"))
	}
	code := flags.Args[0]
	c, _ := setupClient()
	fmt.Fprintf(os.Stderr, "Downloading expense document %s...\n", code)
	f, err := c.DownloadFinancialDocument(code)
	if err != nil {
		exitErr(err)
	}
	if err := saveDownload(f, code, flags); err != nil {
		exitErr(err)
	}
}

func runReport(args []string) {
	flags, err := parseCmdFlags(args)
	if err != nil {
		exitErr(err)
	}
	year := time.Now().Year()
	switch len(flags.Args) {
	case 0:
	case 1:
		year, err = strconv.Atoi(flags.Args[0])
		if err != nil {
			exitErr(fmt.Errorf("invalid year %q\nUsage: solo-cli report [year] [-o path] [-f]", flags.Args[0]))
		}
	default:
		exitErr(fmt.Errorf("too many arguments\nUsage: solo-cli report [year] [-o path] [-f]"))
	}

	c, _ := setupClient()
	if c.CompanyID == "" {
		exitErr(fmt.Errorf("could not determine company ID"))
	}
	if first, last := c.CashflowYears(); year < first || year > last {
		exitErr(fmt.Errorf("no cash flow register for %d, available years are %d to %d", year, first, last))
	}
	fmt.Fprintf(os.Stderr, "Downloading cash flow register for %d...\n", year)
	f, err := c.DownloadCashflowReport(c.CompanyID, year)
	if err != nil {
		exitErr(err)
	}
	if err := saveDownload(f, fmt.Sprintf("Registru-incasari-plati-%d", year), flags); err != nil {
		exitErr(err)
	}
}

// printCompanyExtras appends the VIES state and the bank accounts to the
// company output. Both are best effort, a failure only warns on stderr
func printCompanyExtras(c *client.Client) {
	if vies, err := c.GetViesProfile(c.CompanyID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	} else {
		fmt.Printf("VIES status: %s\n", vies.ViesProfileStatus)
		if code := strings.TrimSpace(strOrEmpty(vies.Code3)); code != "" {
			fmt.Printf("VIES code: %s\n", code)
		}
		if w := vies.ViesWorkflowStatus; w != "" && !strings.EqualFold(w, "None") {
			fmt.Printf("VIES request: %s\n", w)
		}
	}

	accounts, err := c.ListBankAccounts(c.CompanyID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		return
	}
	for _, a := range accounts {
		line := fmt.Sprintf("Bank: %s - %s", a.DisplayName(), a.IBAN)
		if cur := a.CurrencyCode(); cur != "" {
			line += " (" + cur + ")"
		}
		fmt.Println(line)
	}
}

func runSubscription(args []string) {
	if len(args) > 0 {
		exitErr(fmt.Errorf("unexpected argument %q\nUsage: solo-cli subscription", args[0]))
	}
	c, _ := setupClient()
	resp, err := c.GetSubscription()
	if err != nil {
		exitErr(err)
	}
	summary := resp.SubscriptionSummary
	if summary == nil || summary.Subscription == nil {
		exitErr(fmt.Errorf("no subscription information on this account"))
	}
	sub := summary.Subscription

	period := ""
	if plan := resp.ChangeOptions; plan != nil && plan.CurrentSubscription != nil {
		period = strings.ToLower(plan.CurrentSubscription.BillingPeriod)
	}
	suffix := map[string]string{"yearly": "/year", "monthly": "/month"}[period]

	planLine := sub.Name
	if period != "" {
		planLine += " (" + period + ")"
	}
	fmt.Printf("Plan: %s\n", planLine)
	fmt.Printf("Status: %s\n", sub.DisplayStatus())
	fmt.Printf("Price: %s%s\n", taxes.FormatRON(sub.ProductPrice), suffix)
	if sub.DiscountedAmount > 0 {
		fmt.Printf("Discount: %s\n", taxes.FormatRON(sub.DiscountedAmount))
	}
	if sub.HasUpcomingPriceChange && sub.UpcomingProductPrice > 0 {
		line := taxes.FormatRON(sub.UpcomingProductPrice) + suffix
		if on := dateOnly(strOrEmpty(sub.UpcomingChangeOn)); on != "" {
			line += " from " + on
		}
		fmt.Printf("Upcoming price: %s\n", line)
	}
	if sub.NextBillingDate != "" {
		fmt.Printf("Next billing: %s\n", dateOnly(sub.NextBillingDate))
	}
	if cancelled := strOrEmpty(sub.CancelledOn); cancelled != "" {
		fmt.Printf("Cancelled on: %s\n", dateOnly(cancelled))
	}
	if ch := resp.ScheduledChange; ch != nil {
		target := strings.ToLower(ch.TargetBillingPeriod)
		if target == "" {
			target = "a new plan"
		} else {
			target += " billing"
		}
		fmt.Printf("Scheduled change: switch to %s\n", target)
	}
	if card := summary.DefaultCard; card != nil {
		line := card.BrandLabel()
		if card.LastDigits != "" {
			line += " ending in " + card.LastDigits
		}
		if exp := card.Expiry(); exp != "" {
			line += ", expires " + exp
		}
		if card.Expired(time.Now()) {
			line += " (EXPIRED)"
		}
		fmt.Printf("Card: %s\n", line)
	}
	if debt := summary.OverduePaymentSummary; debt != nil && debt.Total > 0 {
		line := taxes.FormatRON(debt.Total)
		if debt.PaymentCount > 1 {
			line += fmt.Sprintf(" (%d payments)", debt.PaymentCount)
		}
		fmt.Printf("Overdue: %s\n", line)
	}
	if benefits, err := c.GetSubscriptionBenefits(); err == nil {
		hosted := "no"
		if benefits.HasProviderHostedHeadquarter {
			hosted = "yes"
		}
		fmt.Printf("Hosted headquarters: %s\n", hosted)
	}
	fmt.Printf("Manage: %s\n", client.SubscriptionURL())
}

func runReferral(args []string) {
	if len(args) > 0 {
		exitErr(fmt.Errorf("unexpected argument %q\nUsage: solo-cli referral", args[0]))
	}
	c, _ := setupClient()
	campaign, err := c.GetReferralCampaign()
	if err != nil {
		exitErr(err)
	}
	if link := campaign.Link(); link != "" {
		fmt.Printf("Link: %s\n", link)
	} else {
		fmt.Fprintln(os.Stderr, "Warning: this account has no referral code")
	}
	if cc := campaign.CurrentCampaign; cc != nil {
		prize := taxes.FormatRON(cc.ReferrerPrizeValue) + " for you"
		if cc.GuestPrizeValue > 0 {
			prize += ", " + taxes.FormatRON(cc.GuestPrizeValue) + " for the invited friend"
		}
		fmt.Printf("Prize: %s\n", prize)
		start, end := dateOnly(strOrEmpty(cc.StartDate)), dateOnly(strOrEmpty(cc.EndDate))
		if cc.IsTimeLimited && start != "" && end != "" {
			fmt.Printf("Period: %s - %s\n", start, end)
		}
	}

	rewards, err := c.GetReferralRewards()
	if err != nil {
		exitErr(err)
	}
	fmt.Printf("Earned: %s\n", taxes.FormatRON(rewards.EarnedWinnings))
	fmt.Printf("Pending: %s\n", taxes.FormatRON(rewards.PendingWinnings))
	if rewards.EarnedWinnings+rewards.PendingWinnings > 0 && c.CompanyID != "" {
		if accounts, err := c.ListBankAccounts(c.CompanyID); err == nil && len(accounts) == 0 {
			fmt.Fprintln(os.Stderr, "Note: payouts need a bank account (IBAN) in the SOLO.ro settings")
		}
	}
}

func runClients(args []string) {
	flags, err := parseCmdFlags(args)
	if err != nil {
		exitErr(err)
	}
	search := strings.Join(flags.Args, " ")
	c, cfg := setupClient()
	customers, err := c.ListAllCustomers(cfg.PageSize, search)
	if err != nil {
		exitErr(err)
	}
	if len(customers) == 0 {
		fmt.Fprintln(os.Stderr, "No clients found")
		return
	}
	for _, cu := range customers {
		fmt.Printf("%d\t%s\t%s\t%s\t%s\t%s\n", cu.CustomerId, cu.Name, cu.Code1, cu.Code2, cu.Email, cu.Phone)
	}
}

func runRates(args []string) {
	day := time.Now()
	switch len(args) {
	case 0:
	case 1:
		d, err := time.ParseInLocation("2006-01-02", args[0], time.Local)
		if err != nil {
			exitErr(fmt.Errorf("invalid date %q, use YYYY-MM-DD", args[0]))
		}
		day = d
	default:
		exitErr(fmt.Errorf("too many arguments\nUsage: solo-cli rates [YYYY-MM-DD]"))
	}

	c, _ := setupClient()
	rates, err := c.GetExchangeRates(day)
	if err != nil {
		exitErr(err)
	}
	if len(rates) == 0 {
		fmt.Fprintf(os.Stderr, "No exchange rates for %s\n", day.Format("2006-01-02"))
		return
	}
	for _, r := range rates {
		fmt.Printf("%s\t%.4f\t%s\n", r.CurrencyCode(), r.RateValue, dateOnly(r.RateDate))
	}
}
