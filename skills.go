package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"solo-cli/config"
)

const (
	skillName = "solo-cli"
	skillRepo = "rursache/solo-cli"
)

// skill files to download (relative to skill/ in the repo)
var skillFiles = []string{
	"SKILL.md",
	"references/help-man-page.md",
}

// skillInstallDirs returns the target directories for skill installation
func skillInstallDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".agents", "skills", skillName),
		filepath.Join(home, ".claude", "skills", skillName),
	}
}

// skillsInstalled checks if skills are already present
func skillsInstalled() bool {
	for _, dir := range skillInstallDirs() {
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
			return true
		}
	}
	return false
}

// skillPromptDone checks if we've already asked the user
func skillPromptDone() bool {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return true // fail closed, don't prompt
	}
	_, err = os.Stat(filepath.Join(configDir, ".skill-prompted"))
	return err == nil
}

// markSkillPromptDone creates the flag file so we don't ask again
func markSkillPromptDone() {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return
	}
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, ".skill-prompted"), []byte("1"), 0644)
}

// isTerminal checks if a file descriptor is a terminal
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// maybePromptSkillInstall prompts the user once to install AI skills
func maybePromptSkillInstall() {
	if skillsInstalled() || skillPromptDone() {
		return
	}
	if !isTerminal(os.Stdin) {
		return
	}

	fmt.Fprintf(os.Stderr, "Install AI skills for Claude Code and other agents? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	markSkillPromptDone()

	if answer == "y" || answer == "yes" {
		if err := downloadAndInstallSkills(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to install skills: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "AI skills installed successfully.")
		}
	}
}

// downloadAndInstallSkills fetches skill files from GitHub and installs them
func downloadAndInstallSkills() error {
	dirs := skillInstallDirs()
	if len(dirs) == 0 {
		return fmt.Errorf("could not determine home directory")
	}

	for _, file := range skillFiles {
		url := fmt.Sprintf("https://raw.githubusercontent.com/%s/master/skill/%s", skillRepo, file)

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", file, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("failed to download %s: HTTP %d", file, resp.StatusCode)
		}

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		for _, dir := range dirs {
			dest := filepath.Join(dir, file)
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", dest, err)
			}
			if err := os.WriteFile(dest, content, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", dest, err)
			}
		}
	}

	return nil
}

// runSetupSkills is the CLI command handler for "setup-skills"
func runSetupSkills() {
	if err := downloadAndInstallSkills(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	markSkillPromptDone()
	for _, dir := range skillInstallDirs() {
		fmt.Printf("Installed: %s\n", dir)
	}
}
