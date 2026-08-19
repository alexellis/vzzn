// Package auth piggybacks on opencode's stored toilgate credentials.
//
// toilgate refresh tokens are valid for an undefined period and can be
// revoked or refreshed by a toilgate administrator; they are read from
// opencode's auth store — the well-known ~/.local/share/opencode/auth.json —
// which is treated as a read-only seed. vzzn reads the refresh token from it,
// mints its own short-lived access tokens through the token endpoint, and
// caches those in its own state file under ~/.vzzn/. It never writes to
// opencode's store.
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// seedEntry mirrors the fields of opencode's toilgate auth entry that vzzn
// needs.
type seedEntry struct {
	Refresh string `json:"refresh"`
}

// state is vzzn's own cached access token.
type state struct {
	Access  string `json:"access"`
	Expires int64  `json:"expires"` // unix milliseconds
}

const margin = 60 * time.Second // refresh this long before expiry

// seedPath is the opencode auth store. One location; if it is not there,
// opencode has not been run on this host and the error says so.
func seedPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("no opencode auth store at %s; run opencode once on this host to mint one", p)
	}
	return p, nil
}

func readSeed() (string, error) {
	p, err := seedPath()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	var top struct {
		Toilgate seedEntry `json:"toilgate"`
	}
	if err := json.Unmarshal(raw, &top); err != nil {
		return "", fmt.Errorf("parsing %s: %w", p, err)
	}
	if top.Toilgate.Refresh == "" {
		return "", fmt.Errorf("no toilgate refresh token in %s; run opencode once to mint one", p)
	}
	return top.Toilgate.Refresh, nil
}

// statePath returns vzzn's own state location under ~/.vzzn/.
func statePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".vzzn", "token.json"), nil
}

func loadState() *state {
	p, err := statePath()
	if err != nil {
		return nil
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var s state
	if json.Unmarshal(raw, &s) != nil {
		return nil
	}
	return &s
}

func saveState(s *state) error {
	p, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".vzzn-token.json.tmp.*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, p)
}

// Token returns a valid bearer token for the toilgate gateway.
func Token(tokenURL string) (string, error) {
	if s := loadState(); s != nil && s.Access != "" && time.Now().Add(margin).UnixMilli() < s.Expires {
		return s.Access, nil
	}
	seed, err := readSeed()
	if err != nil {
		return "", err
	}
	access, expires, err := refresh(tokenURL, seed)
	if err != nil {
		return "", err
	}
	if err := saveState(&state{Access: access, Expires: expires}); err != nil {
		return "", fmt.Errorf("renewed token in hand but caching it: %w", err)
	}
	return access, nil
}

func refresh(tokenURL, seed string) (access string, expires int64, err error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {seed},
	}
	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return "", 0, fmt.Errorf("token endpoint %s: %w", tokenURL, err)
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"` // seconds
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", 0, fmt.Errorf("token endpoint %s: %s: %w", tokenURL, resp.Status, err)
	}
	if out.AccessToken == "" {
		return "", 0, fmt.Errorf("token endpoint %s: %s with no access_token", tokenURL, resp.Status)
	}
	if out.ExpiresIn > 0 {
		expires = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second).UnixMilli()
	}
	return out.AccessToken, expires, nil
}
