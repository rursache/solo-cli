package client

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	cookieFileName = "cookies.json"
	authCookieName = "solo_auth"
)

// SavedCookie represents a cookie that can be serialized. Files written
// before the SSO login only carry name and value, which load as host-only
// falcon cookies
type SavedCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires"`
	HostOnly bool      `json:"host_only,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HttpOnly bool      `json:"http_only,omitempty"`
}

func (sc SavedCookie) expired(now time.Time) bool {
	return !sc.Expires.IsZero() && !sc.Expires.After(now)
}

func (sc SavedCookie) key() string {
	return strings.ToLower(sc.Domain) + "|" + sc.Path + "|" + sc.Name
}

// recordingJar wraps cookiejar.Jar, whose Cookies() only returns name and
// value, and remembers the metadata of every accepted Set-Cookie so the
// session can be persisted faithfully
type recordingJar struct {
	mu   sync.Mutex
	jar  *cookiejar.Jar
	meta map[string]SavedCookie
}

func newRecordingJar() *recordingJar {
	jar, _ := cookiejar.New(nil)
	return &recordingJar{jar: jar, meta: map[string]SavedCookie{}}
}

func (j *recordingJar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.jar.Cookies(u)
}

func (j *recordingJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.jar.SetCookies(u, cookies)
	now := time.Now()
	for _, ck := range cookies {
		sc := SavedCookie{
			Name:     ck.Name,
			Value:    ck.Value,
			Domain:   strings.TrimPrefix(strings.ToLower(ck.Domain), "."),
			Path:     ck.Path,
			Secure:   ck.Secure,
			HttpOnly: ck.HttpOnly,
		}
		if sc.Domain == "" {
			sc.Domain = strings.ToLower(u.Hostname())
			sc.HostOnly = true
		}
		if sc.Path == "" || sc.Path[0] != '/' {
			sc.Path = defaultCookiePath(u.EscapedPath())
		}
		switch {
		case ck.MaxAge < 0:
			sc.Expires = now
		case ck.MaxAge > 0:
			sc.Expires = now.Add(time.Duration(ck.MaxAge) * time.Second)
		case !ck.Expires.IsZero():
			sc.Expires = ck.Expires
		}
		if sc.expired(now) || ck.Value == "" {
			delete(j.meta, sc.key())
			continue
		}
		// Only record what the jar actually accepted
		probe := &url.URL{Scheme: u.Scheme, Host: u.Host, Path: sc.Path}
		for _, got := range j.jar.Cookies(probe) {
			if got.Name == sc.Name && got.Value == sc.Value {
				j.meta[sc.key()] = sc
				break
			}
		}
	}
}

// defaultCookiePath implements RFC 6265 section 5.1.4
func defaultCookiePath(p string) string {
	if p == "" || p[0] != '/' {
		return "/"
	}
	i := strings.LastIndex(p, "/")
	if i == 0 {
		return "/"
	}
	return p[:i]
}

func (j *recordingJar) snapshot() []SavedCookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now()
	out := make([]SavedCookie, 0, len(j.meta))
	for _, sc := range j.meta {
		if !sc.expired(now) {
			out = append(out, sc)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].key() < out[b].key() })
	return out
}

// restore replays saved cookies into the jar with their original scope
func (j *recordingJar) restore(cookies []SavedCookie) {
	falcon, _ := url.Parse(baseURL)
	for _, sc := range cookies {
		if sc.Domain == "" {
			// Legacy entries from cookiejar.Cookies() carry no scope
			sc.Domain = strings.ToLower(falcon.Hostname())
			sc.HostOnly = true
		}
		if sc.Path == "" {
			sc.Path = "/"
		}
		scheme := "http"
		if sc.Secure {
			scheme = "https"
		}
		ck := &http.Cookie{
			Name:     sc.Name,
			Value:    sc.Value,
			Path:     sc.Path,
			Expires:  sc.Expires,
			Secure:   sc.Secure,
			HttpOnly: sc.HttpOnly,
		}
		if !sc.HostOnly {
			ck.Domain = sc.Domain
		}
		j.SetCookies(&url.URL{Scheme: scheme, Host: sc.Domain, Path: sc.Path}, []*http.Cookie{ck})
	}
}

// reset empties the jar, keeping only the cookies keep accepts
func (j *recordingJar) reset(keep func(SavedCookie) bool) {
	var kept []SavedCookie
	for _, sc := range j.snapshot() {
		if keep(sc) {
			kept = append(kept, sc)
		}
	}
	j.mu.Lock()
	j.jar, _ = cookiejar.New(nil)
	j.meta = map[string]SavedCookie{}
	j.mu.Unlock()
	j.restore(kept)
}

func isRememberCookie(sc SavedCookie) bool {
	return strings.Contains(strings.ToLower(sc.Name), "remember")
}

// getCookiePath returns the path to the cookies file
func getCookiePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "solo-cli", cookieFileName), nil
}

// SaveCookies persists the falcon session and the cont.solo.ro cookies,
// including the remember-device cookie that spares the next 2FA code
func (c *Client) SaveCookies() error {
	cookiePath, err := getCookiePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c.jar.snapshot(), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cookiePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(cookiePath, data, 0600)
}

// LoadCookies loads saved cookies from disk and returns true if a usable
// solo_auth session cookie was among them. A remember-device cookie is
// loaded either way so a fresh login can use it
func (c *Client) LoadCookies() (bool, error) {
	cookiePath, err := getCookiePath()
	if err != nil {
		return false, err
	}

	data, err := os.ReadFile(cookiePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	var saved []SavedCookie
	if err := json.Unmarshal(data, &saved); err != nil {
		return false, nil // Invalid JSON, just need to login again
	}

	now := time.Now()
	var valid []SavedCookie
	for _, sc := range saved {
		if !sc.expired(now) && sc.Value != "" {
			valid = append(valid, sc)
		}
	}
	c.jar.restore(valid)
	return c.hasSessionCookie(), nil
}

// HasRememberCookie reports whether a remember-device cookie is loaded, the
// cookie that lets cont.solo.ro skip the 2FA code
func (c *Client) HasRememberCookie() bool {
	for _, sc := range c.jar.snapshot() {
		if isRememberCookie(sc) {
			return true
		}
	}
	return false
}

// ForgetRememberCookie drops the remember-device cookie, used when the
// server asked for a code despite it
func (c *Client) ForgetRememberCookie() {
	c.jar.reset(func(sc SavedCookie) bool { return !isRememberCookie(sc) })
}

// clearForLogin drops the stale session (solo_auth included, otherwise the
// post-bridge check would pass on it) but keeps the remember-device cookie
func (c *Client) clearForLogin() {
	c.jar.reset(isRememberCookie)
}

// ClearCookies removes the saved cookies file
func ClearCookies() error {
	cookiePath, err := getCookiePath()
	if err != nil {
		return err
	}
	return os.Remove(cookiePath)
}
