package main

import (
	"fmt"
	"os"

	"solo-cli/client"
)

// docsPageSize pages the queued and rejected document lists
const docsPageSize = 100

const revenuesUsage = "Usage: solo-cli revenues [queue | rejected | delete <id> [-y]]"

// runRevenuesCmd validates the revenues subcommands for uploaded revenue
// documents before signing in, then runs them
func runRevenuesCmd(args []string) {
	if len(args) == 0 {
		withClient(runRevenues)
		return
	}
	switch args[0] {
	case "queue", "queued", "q", "rejected":
		if len(args) > 1 {
			exitErr(fmt.Errorf("unexpected argument %q\n%s", args[1], revenuesUsage))
		}
		c, _ := setupClient()
		if args[0] == "rejected" {
			listRejectedRevenues(c)
		} else {
			listQueuedRevenues(c)
		}
	case "delete", "del", "rm":
		flags, err := parseCmdFlags(args[1:])
		if err != nil {
			exitErr(err)
		}
		id, err := parseDocumentID(flags.Args, "solo-cli revenues delete <id> [-y]")
		if err != nil {
			exitErr(err)
		}
		c, _ := setupClient()
		if err := deleteRevenueDocument(c, id, flags.Yes); err != nil {
			exitErr(err)
		}
	default:
		exitErr(fmt.Errorf("unknown revenues subcommand %q\n%s", args[0], revenuesUsage))
	}
}

// warnRejectedRevenues mirrors the rejected expenses warning of "expenses"
func warnRejectedRevenues(c *client.Client) {
	rejected, err := c.ListRejectedRevenues(0, docsPageSize)
	if err != nil || len(rejected.Items) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "⚠️  %d rejected revenue document(s):\n", len(rejected.Items))
	for _, r := range rejected.Items {
		fmt.Fprintf(os.Stderr, "   • %s - %s%s\n", r.DocumentName, r.Reason, docCodeSuffix(r.DocumentCode))
	}
	fmt.Fprintln(os.Stderr)
}

func listQueuedRevenues(c *client.Client) {
	queue, err := c.ListQueuedRevenues(0, docsPageSize, "")
	if err != nil {
		exitErr(err)
	}
	for _, q := range queue.Items {
		overdue := ""
		if q.IsOverdue {
			overdue = "OVERDUE"
		}
		fmt.Printf("%s\t%d days\t%s\t(ID: %d)%s\n", q.DocumentName, q.DaysPassed, overdue, q.Id, lastColumn(q.DocumentCode))
	}
}

func listRejectedRevenues(c *client.Client) {
	rejected, err := c.ListRejectedRevenues(0, docsPageSize)
	if err != nil {
		exitErr(err)
	}
	for _, r := range rejected.Items {
		fmt.Printf("%s\t%s\t(ID: %d)%s\n", r.DocumentName, r.Reason, r.Id, lastColumn(r.DocumentCode))
	}
}

func deleteRevenueDocument(c *client.Client, id int, yes bool) error {
	// The DELETE endpoint takes any revenue id, so only ids the queued or
	// rejected lists show are accepted
	name, err := findRevenueDocument(c, id)
	if err != nil {
		return err
	}
	if err := confirm(fmt.Sprintf("Delete revenue document %s (ID %d)?", name, id), yes); err != nil {
		return err
	}
	if err := c.DeleteRevenueDocument(id); err != nil {
		return err
	}
	fmt.Println("Document deleted")
	return nil
}

// findRevenueDocument returns the name of the queued or rejected revenue
// document with the given id
func findRevenueDocument(c *client.Client, id int) (string, error) {
	for start := 0; ; start += docsPageSize {
		page, err := c.ListQueuedRevenues(start, docsPageSize, "")
		if err != nil {
			return "", err
		}
		for _, q := range page.Items {
			if q.Id == id {
				return q.DocumentName, nil
			}
		}
		if len(page.Items) < docsPageSize {
			break
		}
	}
	for start := 0; ; start += docsPageSize {
		page, err := c.ListRejectedRevenues(start, docsPageSize)
		if err != nil {
			return "", err
		}
		for _, r := range page.Items {
			if r.Id == id {
				return r.DocumentName, nil
			}
		}
		if len(page.Items) < docsPageSize {
			break
		}
	}
	return "", fmt.Errorf("no queued or rejected revenue document with ID %d, see solo-cli revenues queue and revenues rejected", id)
}
