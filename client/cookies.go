package client

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const cookieFileName = "cookies.json"

// SavedCookie represents a cookie that can be serialized
type SavedCookie struct {
	Name    string    `json:"name"`
	Value   string    `json:"value"`
	Domain  string    `json:"domain"`
	Path    string    `json:"path"`
	Expires time.Time `json:"expires"`
}

// getCookiePath returns the path to the cookies file
func getCookiePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "solo-cli", cookieFileName), nil
}

// SaveCookies saves the current session cookies to disk
func (c *Client) SaveCookies() error {
	cookiePath, err := getCookiePath()
	if err != nil {
		return err
	}

	// Get cookies for the SOLO.ro domain
	u, _ := url.Parse(baseURL)
	cookies := c.httpClient.Jar.Cookies(u)

	var savedCookies []SavedCookie
	for _, cookie := range cookies {
		savedCookies = append(savedCookies, SavedCookie{
			Name:    cookie.Name,
			Value:   cookie.Value,
			Domain:  cookie.Domain,
			Path:    cookie.Path,
			Expires: cookie.Expires,
		})
	}

	data, err := json.MarshalIndent(savedCookies, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cookiePath, data, 0600)
}

// LoadCookies loads saved cookies from disk and returns true if valid cookies were loaded
func (c *Client) LoadCookies() (bool, error) {
	cookiePath, err := getCookiePath()
	if err != nil {
		return false, err
	}

	data, err := os.ReadFile(cookiePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // No saved cookies, not an error
		}
		return false, err
	}

	var savedCookies []SavedCookie
	if err := json.Unmarshal(data, &savedCookies); err != nil {
		return false, nil // Invalid JSON, just need to login again
	}

	if len(savedCookies) == 0 {
		return false, nil
	}

	// Check if any auth cookie is still valid
	now := time.Now()
	var validCookies []*http.Cookie
	hasValidAuthCookie := false

	for _, sc := range savedCookies {
		// Skip expired cookies
		if !sc.Expires.IsZero() && sc.Expires.Before(now) {
			continue
		}

		cookie := &http.Cookie{
			Name:    sc.Name,
			Value:   sc.Value,
			Domain:  sc.Domain,
			Path:    sc.Path,
			Expires: sc.Expires,
		}
		validCookies = append(validCookies, cookie)

		// Check for the auth cookie specifically
		if sc.Name == "solo_auth" {
			hasValidAuthCookie = true
		}
	}

	if !hasValidAuthCookie {
		return false, nil
	}

	// Load valid cookies into the jar
	u, _ := url.Parse(baseURL)
	c.httpClient.Jar.SetCookies(u, validCookies)

	return true, nil
}

// ClearCookies removes the saved cookies file
func ClearCookies() error {
	cookiePath, err := getCookiePath()
	if err != nil {
		return err
	}
	return os.Remove(cookiePath)
}
