package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeHome(t *testing.T, rel, content string) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	abs := filepath.Join(os.Getenv("HOME"), rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return abs
}

func Test_LoadLocalOverride(t *testing.T) {
	writeHome(t, ".vzzn/config.json", `{
  "url": "https://gateway.example.com/v1",
  "model": "my-model",
  "token": "secret"
}`)
	l, err := LoadLocal()
	if err != nil {
		t.Fatal(err)
	}
	if l.URL != "https://gateway.example.com/v1" {
		t.Errorf("URL: %q", l.URL)
	}
	if l.Model != "my-model" {
		t.Errorf("Model: %q", l.Model)
	}
	if l.Token != "secret" {
		t.Errorf("Token: %q", l.Token)
	}
}

func Test_LoadLocalAbsent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	l, err := LoadLocal()
	if err != nil {
		t.Fatal(err)
	}
	if l.URL != "" || l.Model != "" || l.Token != "" {
		t.Errorf("expected empty local config, got %+v", l)
	}
}
