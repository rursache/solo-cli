package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"

	"solo-cli/client"
	"solo-cli/config"
)

const mfaFlagFile = ".mfa-required"

// stdinIsTerminal is a var so tests can pretend to be interactive
var stdinIsTerminal = func() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

var stdinReader = bufio.NewReader(os.Stdin)

// readLine reads one trimmed line from stdin
func readLine() (string, error) {
	line, err := stdinReader.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// setupClient creates an authenticated API client and ensures company ID is discovered
func setupClient() (*client.Client, *config.Config) {
	if err := config.EnsureExists(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrCredentialsMissing) {
			configPath, _ := config.GetConfigPath()
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Please edit: %s\n", configPath)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = config.DefaultUserAgent
	}
	apiClient, err := client.New(userAgent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	needsLogin := true
	if loaded, _ := apiClient.LoadCookies(); loaded {
		if _, err := apiClient.GetSummary(); err == nil {
			needsLogin = false
		}
	}

	if needsLogin {
		if err := login(apiClient, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Auto-discover company ID
	if id, err := apiClient.DiscoverCompanyID(); err == nil {
		apiClient.CompanyID = id
	}

	return apiClient, cfg
}

const runInteractivelyHint = "run solo-cli once in a terminal to enter the code, the trusted device cookie then keeps later runs code-free for 14 days"

// login signs in and persists the session. A 2FA code is only asked for on a
// terminal, non-interactive runs never start a login that would text a code
func login(apiClient *client.Client, cfg *config.Config) error {
	interactive := stdinIsTerminal()
	hadRemember := apiClient.HasRememberCookie()
	if !interactive && mfaKnown(cfg.Username) && !hadRemember {
		return fmt.Errorf("the saved session expired and this account uses two-factor authentication, %s", runInteractivelyHint)
	}

	fmt.Fprintln(os.Stderr, "Logging in to SOLO.ro...")
	prompted := false
	var prompt client.MFAPromptFunc
	if interactive {
		prompt = mfaPrompter(cfg.Username, &prompted)
	}

	err := apiClient.Login(cfg.Username, cfg.Password, prompt)
	switch {
	case errors.Is(err, client.ErrMFARequired), errors.Is(err, client.ErrMFAUnsupported):
		markMFA(cfg.Username)
		// The server asked for a code despite the remember cookie, so it is
		// useless and later runs must not try again
		if !prompted {
			apiClient.ForgetRememberCookie()
			apiClient.SaveCookies()
		}
		if errors.Is(err, client.ErrMFAUnsupported) {
			return err
		}
		return fmt.Errorf("this account uses two-factor authentication and a code was just sent, %s", runInteractivelyHint)
	case errors.Is(err, client.ErrAuthenticationFailed):
		return fmt.Errorf("%v\nPlease check your credentials in the config file", err)
	case err != nil:
		return fmt.Errorf("login failed: %w", err)
	}

	if apiClient.LoginFallback != "" {
		fmt.Fprintf(os.Stderr, "Note: cont.solo.ro sign in did not match the known flow (%s), used the legacy login\n", apiClient.LoginFallback)
	}
	if !prompted && !hadRemember && apiClient.LoginFallback == "" {
		// A full SSO login without a code and without a remember cookie
		// proves the account no longer uses 2FA
		clearMFA()
	}
	if err := apiClient.SaveCookies(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save session: %v\n", err)
	}
	return nil
}

// mfaPrompter asks for the code on the terminal and flags the account on
// the first call, the code is already sent even if the login then fails or
// is aborted at the prompt
func mfaPrompter(username string, prompted *bool) client.MFAPromptFunc {
	return func(p client.MFAPrompt) (string, error) {
		if !*prompted {
			markMFA(username)
		}
		*prompted = true
		return promptMFACode(p)
	}
}

func promptMFACode(p client.MFAPrompt) (string, error) {
	if p.Hint != "" {
		fmt.Fprintln(os.Stderr, p.Hint)
	}
	if p.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", p.Error)
	}
	resend := ""
	if p.CanResend {
		resend = ", empty or r to resend"
	}
	fmt.Fprintf(os.Stderr, "Two-factor code (%d digits%s): ", p.CodeLength, resend)
	code, err := readLine()
	if err != nil {
		return "", fmt.Errorf("reading the code: %w", err)
	}
	if p.CanResend && (code == "" || strings.EqualFold(code, "r")) {
		fmt.Fprintln(os.Stderr, "Sending a new code...")
		return "", client.ErrMFAResend
	}
	return strings.ReplaceAll(code, " ", ""), nil
}

func mfaFlagPath() string {
	dir, err := config.GetConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, mfaFlagFile)
}

// mfaKnown reports whether username was seen hitting the 2FA code page
func mfaKnown(username string) bool {
	path := mfaFlagPath()
	if path == "" {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(string(data)), strings.TrimSpace(username))
}

func markMFA(username string) {
	if path := mfaFlagPath(); path != "" {
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(strings.TrimSpace(username)+"\n"), 0600)
	}
}

func clearMFA() {
	if path := mfaFlagPath(); path != "" {
		os.Remove(path)
	}
}

// withClient handles auth and runs a command with a client
func withClient(fn func(*client.Client)) {
	apiClient, _ := setupClient()
	fn(apiClient)
}

// withPagedClient handles auth and runs a list command with the configured
// page_size
func withPagedClient(fn func(*client.Client, int)) {
	apiClient, cfg := setupClient()
	fn(apiClient, cfg.PageSize)
}

// withClientArgs handles auth and runs a command with client and args
func withClientArgs(fn func(*client.Client, []string), args []string) {
	apiClient, _ := setupClient()
	fn(apiClient, args)
}
