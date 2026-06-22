package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultConfigWhenMissing(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.CurrentProfile != DefaultProfile {
		t.Fatalf("CurrentProfile = %q, want %q", cfg.CurrentProfile, DefaultProfile)
	}
	if cfg.Profiles[DefaultProfile].Endpoint != DefaultEndpoint {
		t.Fatalf("endpoint = %q, want %q", cfg.Profiles[DefaultProfile].Endpoint, DefaultEndpoint)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	cfg := DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = Profile{Endpoint: "https://example.test/"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.CurrentProfile != "prod" {
		t.Fatalf("CurrentProfile = %q", loaded.CurrentProfile)
	}
	if got := loaded.Profiles["prod"].Endpoint; got != "https://example.test" {
		t.Fatalf("endpoint = %q", got)
	}
	if _, err := os.Stat(filepath.Join(home, "config.json")); err != nil {
		t.Fatalf("config file missing: %v", err)
	}
}

func TestCredentialsRoundTrip(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	creds := &Credentials{Profiles: map[string]Credential{
		"default": {APIKey: "tmc_test"},
	}}
	if err := SaveCredentials(creds); err != nil {
		t.Fatalf("SaveCredentials() error = %v", err)
	}
	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials() error = %v", err)
	}
	if got := loaded.Profiles["default"].APIKey; got != "tmc_test" {
		t.Fatalf("api key = %q", got)
	}
}

func TestEnvAPIKey(t *testing.T) {
	t.Setenv("TMCOPILOT_API_KEY", "tmc_env")
	got, ok := EnvAPIKey()
	if !ok || got != "tmc_env" {
		t.Fatalf("EnvAPIKey() = %q, %v", got, ok)
	}
}
