package main

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"solo-cli/client"
)

func fakeStdin(t *testing.T, terminal bool, input string) {
	t.Helper()
	origTTY, origReader := stdinIsTerminal, stdinReader
	stdinIsTerminal = func() bool { return terminal }
	stdinReader = bufio.NewReader(strings.NewReader(input))
	t.Cleanup(func() { stdinIsTerminal, stdinReader = origTTY, origReader })
}

func TestParseCmdFlags(t *testing.T) {
	f, err := parseCmdFlags([]string{"ABC-1", "-o", "out.pdf", "--yes", "--extra", "-f", "--", "-literal"})
	if err != nil {
		t.Fatal(err)
	}
	if !f.Yes || !f.Force || f.Output != "out.pdf" {
		t.Errorf("flags: %+v", f)
	}
	if !reflect.DeepEqual(f.Args, []string{"ABC-1", "--extra", "-literal"}) {
		t.Errorf("args = %q", f.Args)
	}
	if f, _ := parseCmdFlags([]string{"--output=a/b.xml", "-y"}); f.Output != "a/b.xml" || !f.Yes {
		t.Errorf("--output=: %+v", f)
	}
	if _, err := parseCmdFlags([]string{"x", "-o"}); err == nil {
		t.Error("-o without a path must fail")
	}
}

func TestConfirm(t *testing.T) {
	fakeStdin(t, false, "")
	if err := confirm("Delete invoice ABC?", true); err != nil {
		t.Errorf("--yes must skip the prompt: %v", err)
	}
	err := confirm("Delete invoice ABC?", false)
	if err == nil || !strings.Contains(err.Error(), "--yes") || !strings.Contains(err.Error(), "Delete invoice ABC") {
		t.Errorf("non-interactive without --yes = %v", err)
	}

	for input, want := range map[string]error{"y\n": nil, "YES\n": nil, "n\n": errCancelled, "\n": errCancelled, "": errCancelled} {
		fakeStdin(t, true, input)
		if err := confirm("Mark paid?", false); !errors.Is(err, want) {
			t.Errorf("answer %q = %v, want %v", input, err, want)
		}
	}
}

func TestSafeFileName(t *testing.T) {
	for in, want := range map[string]string{
		"INV/2026:001":   "INV-2026-001",
		"../etc/passwd":  "-etc-passwd",
		"  bon\x1b[2J  ": "bon-[2J",
		"":               "download",
		"..":             "download",
	} {
		if got := safeFileName(in); got != want {
			t.Errorf("safeFileName(%q) = %q, want %q", in, got, want)
		}
	}
	f := &client.DownloadedFile{Data: []byte("%PDF"), ContentType: "application/pdf"}
	if got := downloadName("SOLO 12", f); got != "SOLO 12.pdf" {
		t.Errorf("downloadName = %q", got)
	}
}

func TestWriteDownload(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	path, err := writeDownload([]byte("one"), "", "inv.pdf", false)
	if err != nil || path != "inv.pdf" {
		t.Fatalf("default name: %q, %v", path, err)
	}
	if _, err := writeDownload([]byte("two"), "", "inv.pdf", false); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Errorf("overwrite without force = %v", err)
	}
	if data, _ := os.ReadFile("inv.pdf"); string(data) != "one" {
		t.Errorf("existing file changed: %q", data)
	}
	if _, err := writeDownload([]byte("three"), "inv.pdf", "ignored.pdf", true); err != nil {
		t.Errorf("force overwrite: %v", err)
	}
	if data, _ := os.ReadFile("inv.pdf"); string(data) != "three" {
		t.Errorf("forced file = %q", data)
	}

	sub := filepath.Join(dir, "out")
	os.Mkdir(sub, 0755)
	if path, err := writeDownload([]byte("x"), sub, "doc.xml", false); err != nil || path != filepath.Join(sub, "doc.xml") {
		t.Errorf("directory output: %q, %v", path, err)
	}
}
