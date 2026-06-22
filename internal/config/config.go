package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultEndpoint = "https://api.tmcopilot.ai"
	DefaultProfile  = "default"
)

type Config struct {
	CurrentProfile string             `json:"current_profile"`
	Profiles       map[string]Profile `json:"profiles"`
	DefaultFormat  string             `json:"default_format,omitempty"`
}

type Profile struct {
	Endpoint    string `json:"endpoint"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Format      string `json:"format,omitempty"`
	DeviceUUID  string `json:"device_uuid,omitempty"`
}

type Credentials struct {
	Profiles map[string]Credential `json:"profiles"`
}

type Credential struct {
	APIKey string `json:"api_key"`
}

func HomeDir() (string, error) {
	if value := strings.TrimSpace(os.Getenv("TMCOPILOT_HOME")); value != "" {
		return value, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tmcopilot"), nil
}

func ConfigPath() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "config.json"), nil
}

func CredentialsPath() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "credentials.json"), nil
}

func DefaultConfig() *Config {
	return &Config{
		CurrentProfile: DefaultProfile,
		DefaultFormat:  "json",
		Profiles: map[string]Profile{
			DefaultProfile: {
				Endpoint: DefaultEndpoint,
			},
		},
	}
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	cfg.normalize()
	return &cfg, nil
}

func Save(cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	cfg.normalize()
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o600)
}

func LoadCredentials() (*Credentials, error) {
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Credentials{Profiles: map[string]Credential{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(raw, &creds); err != nil {
		return nil, fmt.Errorf("parse credentials %s: %w", path, err)
	}
	if creds.Profiles == nil {
		creds.Profiles = map[string]Credential{}
	}
	return &creds, nil
}

func SaveCredentials(creds *Credentials) error {
	if creds == nil {
		creds = &Credentials{Profiles: map[string]Credential{}}
	}
	if creds.Profiles == nil {
		creds.Profiles = map[string]Credential{}
	}
	path, err := CredentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o600)
}

func (c *Config) normalize() {
	if c.CurrentProfile == "" {
		c.CurrentProfile = DefaultProfile
	}
	if c.DefaultFormat == "" {
		c.DefaultFormat = "json"
	}
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	if _, ok := c.Profiles[c.CurrentProfile]; !ok {
		c.Profiles[c.CurrentProfile] = Profile{Endpoint: DefaultEndpoint}
	}
	for name, profile := range c.Profiles {
		profile.Endpoint = NormalizeEndpoint(profile.Endpoint)
		if profile.Endpoint == "" {
			profile.Endpoint = DefaultEndpoint
		}
		c.Profiles[name] = profile
	}
}

func NormalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimRight(endpoint, "/")
	return endpoint
}

func (c *Config) ActiveProfile(name string) (string, Profile) {
	c.normalize()
	if strings.TrimSpace(name) == "" {
		name = c.CurrentProfile
	}
	profile, ok := c.Profiles[name]
	if !ok {
		profile = Profile{Endpoint: DefaultEndpoint}
	}
	profile.Endpoint = NormalizeEndpoint(profile.Endpoint)
	return name, profile
}

func EnvAPIKey() (string, bool) {
	for _, key := range []string{"TMCOPILOT_API_KEY", "TMC_API_KEY"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func EnvEndpoint() string {
	for _, key := range []string{"TMCOPILOT_ENDPOINT", "TMC_ENDPOINT"} {
		value := NormalizeEndpoint(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}
