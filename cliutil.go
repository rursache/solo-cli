package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"solo-cli/client"
)

// errCancelled is returned by confirm when the user answers no
var errCancelled = errors.New("cancelled")

// cmdFlags holds the shared flags of mutating and download commands
type cmdFlags struct {
	Yes    bool     // --yes, -y
	Force  bool     // --force, -f
	Output string   // --output, -o
	Args   []string // positional arguments and flags this parser does not know
}

// parseCmdFlags pulls --yes/-y, --force/-f and -o/--output <path> (or
// --output=path) out of args wherever they appear. Everything after "--" is
// positional
func parseCmdFlags(args []string) (cmdFlags, error) {
	var f cmdFlags
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			f.Args = append(f.Args, args[i+1:]...)
			return f, nil
		case a == "--yes" || a == "-y":
			f.Yes = true
		case a == "--force" || a == "-f":
			f.Force = true
		case a == "--output" || a == "-o":
			if i+1 >= len(args) || args[i+1] == "" {
				return f, fmt.Errorf("%s requires a path", a)
			}
			i++
			f.Output = args[i]
		case strings.HasPrefix(a, "--output="):
			f.Output = strings.TrimPrefix(a, "--output=")
		default:
			f.Args = append(f.Args, a)
		}
	}
	return f, nil
}

// confirm guards destructive or outward-facing actions. yes (from --yes)
// skips the question, on a terminal it asks "[y/N]" on stderr, otherwise it
// fails asking for --yes. A "no" answer returns errCancelled
func confirm(question string, yes bool) error {
	if yes {
		return nil
	}
	if !stdinIsTerminal() {
		return fmt.Errorf("%s needs confirmation, pass --yes (-y) to run it without a terminal", strings.TrimSuffix(question, "?"))
	}
	fmt.Fprintf(os.Stderr, "%s [y/N] ", question)
	answer, err := readLine()
	if err != nil {
		fmt.Fprintln(os.Stderr)
		return errCancelled
	}
	switch strings.ToLower(answer) {
	case "y", "yes", "d", "da":
		return nil
	}
	return errCancelled
}

// exitErr prints err on stderr and exits with status 1
func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

// listed unwraps a ListAll result for a list command, exiting on errors and
// warning on stderr when paging stopped early
func listed[T any](items []T, err error) []T {
	if errors.Is(err, client.ErrListTruncated) {
		fmt.Fprintf(os.Stderr, "Warning: %v, showing the first %d\n", err, len(items))
		return items
	}
	if err != nil {
		exitErr(err)
	}
	return items
}

// lookupErr drops ErrListTruncated so a lookup still searches what was fetched
func lookupErr(err error) error {
	if errors.Is(err, client.ErrListTruncated) {
		return nil
	}
	return err
}

// safeFileName turns a server value (invoice serial, document name) into a
// file name without path separators or control characters
func safeFileName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r == '/' || r == '\\' || r == ':' || r < 0x20 || r == 0x7f:
			return '-'
		}
		return r
	}, strings.TrimSpace(name))
	name = strings.Trim(name, ". ")
	if name == "" {
		return "download"
	}
	return name
}

// downloadName is the default file name for a download: base plus the
// extension guessed from the content type or file signature
func downloadName(base string, f *client.DownloadedFile) string {
	return safeFileName(base) + "." + f.Extension()
}

// writeDownload writes data to output, or to defaultName in the current
// directory when output is empty (inside output when it is a directory). It
// never overwrites an existing file unless force and returns the written path
func writeDownload(data []byte, output, defaultName string, force bool) (string, error) {
	path := output
	if path == "" {
		path = defaultName
	} else if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, defaultName)
	}

	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("%s already exists, pass --force (-f) to overwrite it or -o to pick another path", path)
		}
		return "", err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return path, nil
}

// saveDownload writes a downloaded file per the -o/--force flags and prints
// the written path on stdout
func saveDownload(f *client.DownloadedFile, baseName string, flags cmdFlags) error {
	path, err := writeDownload(f.Data, flags.Output, downloadName(baseName, f), flags.Force)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}
