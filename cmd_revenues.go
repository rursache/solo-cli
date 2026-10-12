package main

import (
	"fmt"
	"os"
	"strconv"

	"solo-cli/client"
)

const revenueDocsPageSize = 100

const revenuesUsage = "Usage: solo-cli revenues [queue | rejected | delete <id> [--yes]]"

// runRevenueDocuments handles the "revenues" subcommands for uploaded
// revenue documents
func runRevenueDocuments(c *client.Client, args []string) {
	switch args[0] {
	case "queue", "queued", "q":
		listQueuedRevenues(c)
	case "rejected":
		listRejectedRevenues(c)
	case "delete", "del", "rm":
		if err := deleteRevenueDocument(c, args[1:]); err != nil {
			exitErr(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown revenues subcommand %q\n%s\n", args[0], revenuesUsage)
		os.Exit(1)
	}
}

// warnRejectedRevenues mirrors the rejected expenses warning of "expenses"
func warnRejectedRevenues(c *client.Client) {
	rejected, err := c.ListRejectedRevenues(0, revenueDocsPageSize)
	if err != nil || len(rejected.Items) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "⚠️  %d rejected revenue document(s):\n", len(rejected.Items))
	for _, r := range rejected.Items {
		fmt.Fprintf(os.Stderr, "   • %s - %s\n", r.DocumentName, r.Reason)
	}
	fmt.Fprintln(os.Stderr)
}

func listQueuedRevenues(c *client.Client) {
	queue, err := c.ListQueuedRevenues(0, revenueDocsPageSize, "")
	if err != nil {
		exitErr(err)
	}
	for _, q := range queue.Items {
		overdue := ""
		if q.IsOverdue {
			overdue = "OVERDUE"
		}
		fmt.Printf("%s\t%d days\t%s\t(ID: %d)\n", q.DocumentName, q.DaysPassed, overdue, q.Id)
	}
}

func listRejectedRevenues(c *client.Client) {
	rejected, err := c.ListRejectedRevenues(0, revenueDocsPageSize)
	if err != nil {
		exitErr(err)
	}
	for _, r := range rejected.Items {
		fmt.Printf("%s\t%s\t(ID: %d)\n", r.DocumentName, r.Reason, r.Id)
	}
}

func deleteRevenueDocument(c *client.Client, args []string) error {
	flags, err := parseCmdFlags(args)
	if err != nil {
		return err
	}
	if len(flags.Args) != 1 {
		return fmt.Errorf("missing ID\nUsage: solo-cli revenues delete <id> [--yes]")
	}
	id, err := strconv.Atoi(flags.Args[0])
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid ID '%s' (must be a number)", flags.Args[0])
	}

	// The DELETE endpoint takes any revenue id, so only ids the queued or
	// rejected lists show are accepted
	name, err := findRevenueDocument(c, id)
	if err != nil {
		return err
	}
	if err := confirm(fmt.Sprintf("Delete revenue document %s (ID %d)?", name, id), flags.Yes); err != nil {
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
	for start := 0; ; start += revenueDocsPageSize {
		page, err := c.ListQueuedRevenues(start, revenueDocsPageSize, "")
		if err != nil {
			return "", err
		}
		for _, q := range page.Items {
			if q.Id == id {
				return q.DocumentName, nil
			}
		}
		if len(page.Items) < revenueDocsPageSize {
			break
		}
	}
	for start := 0; ; start += revenueDocsPageSize {
		page, err := c.ListRejectedRevenues(start, revenueDocsPageSize)
		if err != nil {
			return "", err
		}
		for _, r := range page.Items {
			if r.Id == id {
				return r.DocumentName, nil
			}
		}
		if len(page.Items) < revenueDocsPageSize {
			break
		}
	}
	return "", fmt.Errorf("no queued or rejected revenue document with ID %d, see solo-cli revenues queue and revenues rejected", id)
}
