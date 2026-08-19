// Package config resolves the local opencode configuration.
//
// opencode.json is the source of truth for endpoints on every host:
// the well-known location ~/.config/opencode/opencode.json.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config mirrors the subset of opencode.json that vzzn needs.
type Config struct {
	Model    string              `json:"model"`
	Provider map[string]Provider `json:"provider"`
}

// Provider is one entry under "provider" in opencode.json.
type Provider struct {
	Name    string           `json:"name"`
	Options Options          `json:"options"`
	Models  map[string]Model `json:"models"`
}

// Options carries the connection settings.
type Options struct {
	BaseURL string `json:"baseURL"`
	Timeout int    `json:"timeout"`
}

// Model is one entry under a provider's "models".
type Model struct {
	Name string `json:"name"`
}

// Load reads the opencode config from the well-known location.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".config", "opencode", "opencode.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &c, nil
}

// Local is vzzn's own optional override file at ~/.vzzn/config.json. Every
// field is optional; opencode.json remains the source of truth upstream of
// any override.
type Local struct {
	Model string `json:"model"`
}

// LoadLocal reads ~/.vzzn/config.json, tolerating its absence.
func LoadLocal() (*Local, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(home, ".vzzn", "config.json")
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Local{}, nil
		}
		return nil, err
	}
	var l Local
	if err := json.Unmarshal(raw, &l); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", p, err)
	}
	return &l, nil
}

// Resolve derives the inference base URL and the token endpoint for a
// named provider. The token endpoint is the base URL with the /v1 suffix
// replaced by /token.
func Resolve(c *Config, name string) (baseURL, tokenURL string, err error) {
	p, ok := c.Provider[name]
	if !ok {
		return "", "", fmt.Errorf("provider %q not found in opencode config", name)
	}
	base := strings.TrimSuffix(p.Options.BaseURL, "/")
	if !strings.HasSuffix(base, "/v1") {
		return "", "", fmt.Errorf("provider %q baseURL %q does not end in /v1", name, p.Options.BaseURL)
	}
	token := strings.TrimSuffix(base, "/v1") + "/token"
	return base, token, nil
}
