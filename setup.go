package main

import (
	"errors"
	"fmt"
	"os"

	"solo-cli/client"
	"solo-cli/config"
)

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
		fmt.Fprintln(os.Stderr, "Logging in to SOLO.ro...")
		if err := apiClient.Login(cfg.Username, cfg.Password); err != nil {
			if errors.Is(err, client.ErrAuthenticationFailed) {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				fmt.Fprintln(os.Stderr, "Please check your credentials in the config file.")
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Login error: %v\n", err)
			os.Exit(1)
		}
		if err := apiClient.SaveCookies(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save session: %v\n", err)
		}
	}

	// Auto-discover company ID
	if id, err := apiClient.DiscoverCompanyID(); err == nil {
		apiClient.CompanyID = id
	}

	return apiClient, cfg
}

// withClient handles auth and runs a command with a client
func withClient(fn func(*client.Client)) {
	apiClient, _ := setupClient()
	fn(apiClient)
}

// withClientArgs handles auth and runs a command with client and args
func withClientArgs(fn func(*client.Client, []string), args []string) {
	apiClient, _ := setupClient()
	fn(apiClient, args)
}
