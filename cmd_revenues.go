package main

import (
	"fmt"
	"os"

	"solo-cli/client"
)

const revenuesUsage = "Usage: solo-cli revenues [queue | rejected | delete <id> [-y]]"

// runRevenuesCmd validates the revenues subcommands for uploaded revenue
// documents before signing in, then runs them
func runRevenuesCmd(args []string) {
	if len(args) == 0 {
		withPagedClient(runRevenues)
		return
	}
	switch args[0] {
	case "queue", "queued", "q", "rejected":
		if len(args) > 1 {
			exitErr(fmt.Errorf("unexpected argument %q\n%s", args[1], revenuesUsage))
		}
		c, cfg := setupClient()
		if args[0] == "rejected" {
			listRejectedRevenues(c, cfg.PageSize)
		} else {
			listQueuedRevenues(c, cfg.PageSize)
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
		c, cfg := setupClient()
		if err := deleteRevenueDocument(c, cfg.PageSize, id, flags.Yes); err != nil {
			exitErr(err)
		}
	default:
		exitErr(fmt.Errorf("unknown revenues subcommand %q\n%s", args[0], revenuesUsage))
	}
}

// warnRejectedRevenues mirrors the rejected expenses warning of "expenses"
func warnRejectedRevenues(c *client.Client, pageSize int) {
	rejected, err := c.ListAllRejectedRevenues(pageSize)
	if lookupErr(err) != nil || len(rejected) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "⚠️  %d rejected revenue document(s):\n", len(rejected))
	for _, r := range rejected {
		fmt.Fprintf(os.Stderr, "   • %s - %s%s\n", r.DocumentName, r.Reason, rejectedRef(r.Id, r.DocumentCode))
	}
	fmt.Fprintln(os.Stderr)
}

func listQueuedRevenues(c *client.Client, pageSize int) {
	for _, q := range listed(c.ListAllQueuedRevenues(pageSize, "")) {
		overdue := ""
		if q.IsOverdue {
			overdue = "OVERDUE"
		}
		fmt.Printf("%s\t%d days\t%s\t(ID: %d)%s\n", q.DocumentName, q.DaysPassed, overdue, q.Id, lastColumn(q.DocumentCode))
	}
}

func listRejectedRevenues(c *client.Client, pageSize int) {
	for _, r := range listed(c.ListAllRejectedRevenues(pageSize)) {
		fmt.Printf("%s\t%s\t(ID: %d)%s\n", r.DocumentName, r.Reason, r.Id, lastColumn(r.DocumentCode))
	}
}

func deleteRevenueDocument(c *client.Client, pageSize, id int, yes bool) error {
	// The DELETE endpoint takes any revenue id, so only ids the queued or
	// rejected lists show are accepted
	name, err := findRevenueDocument(c, pageSize, id)
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
func findRevenueDocument(c *client.Client, pageSize, id int) (string, error) {
	queue, err := c.ListAllQueuedRevenues(pageSize, "")
	if err := lookupErr(err); err != nil {
		return "", err
	}
	for _, q := range queue {
		if q.Id == id {
			return q.DocumentName, nil
		}
	}
	rejected, err := c.ListAllRejectedRevenues(pageSize)
	if err := lookupErr(err); err != nil {
		return "", err
	}
	for _, r := range rejected {
		if r.Id == id {
			return r.DocumentName, nil
		}
	}
	return "", fmt.Errorf("no queued or rejected revenue document with ID %d, see solo-cli revenues queue and revenues rejected", id)
}
