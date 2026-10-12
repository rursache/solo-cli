package main

import (
	"strings"
	"testing"

	"solo-cli/client"
)

func TestMFAPrompterFlagsAccountBeforeAnswer(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	fakeStdin(t, true, "")

	prompted := false
	_, err := mfaPrompter("user@example.com", &prompted)(client.MFAPrompt{CodeLength: 6})
	if err == nil || !strings.Contains(err.Error(), "reading the code") {
		t.Fatalf("prompt at EOF = %v", err)
	}
	if !prompted || !mfaKnown("user@example.com") {
		t.Errorf("aborted prompt must still flag the account: prompted %v, known %v", prompted, mfaKnown("user@example.com"))
	}
}
