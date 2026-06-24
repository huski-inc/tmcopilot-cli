package tmc

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUninstallPlanRemovesBinariesAndKeepsConfigByDefault(t *testing.T) {
	installDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	tmcPath := filepath.Join(installDir, "tmc")
	aliasPath := filepath.Join(installDir, "tmcopilot")
	if err := os.WriteFile(tmcPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write tmc: %v", err)
	}
	if err := os.WriteFile(aliasPath, []byte("alias"), 0o755); err != nil {
		t.Fatalf("write alias: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	result, err := runUninstallPlan(uninstallPlan{
		BinaryPaths: []string{tmcPath, aliasPath},
	})
	if err != nil {
		t.Fatalf("runUninstallPlan returned error: %v", err)
	}
	if len(result.Removed) != 2 {
		t.Fatalf("removed = %#v, want two paths", result.Removed)
	}
	for _, path := range []string{tmcPath, aliasPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists or stat failed differently: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, "config.json")); err != nil {
		t.Fatalf("config should be kept: %v", err)
	}
	if result.ConfigKept != home {
		t.Fatalf("config_kept = %q, want %q", result.ConfigKept, home)
	}
}

func TestRunUninstallPlanRemovesConfigWhenRequested(t *testing.T) {
	installDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	tmcPath := filepath.Join(installDir, "tmc")
	if err := os.WriteFile(tmcPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write tmc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "credentials.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}

	result, err := runUninstallPlan(uninstallPlan{
		BinaryPaths:  []string{tmcPath},
		ConfigDir:    home,
		RemoveConfig: true,
	})
	if err != nil {
		t.Fatalf("runUninstallPlan returned error: %v", err)
	}
	if result.ConfigRemoved != home {
		t.Fatalf("config_removed = %q, want %q", result.ConfigRemoved, home)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("config dir still exists or stat failed differently: %v", err)
	}
}

func TestUninstallCommandRequiresYes(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"uninstall"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("uninstall without --yes error = nil")
	}
	if !strings.Contains(stderr.String(), "--yes") {
		t.Fatalf("stderr missing --yes guidance: %s", stderr.String())
	}
}

func TestUninstallDryRunWritesPlan(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dry-run", "uninstall", "--remove-config"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("dry-run uninstall failed: %v stderr=%s", err, stderr.String())
	}
	var out struct {
		OK   bool `json:"ok"`
		Data struct {
			DryRun       bool     `json:"dry_run"`
			WouldRemove  []string `json:"would_remove"`
			RemoveConfig bool     `json:"remove_config"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("decode output: %v output=%s", err, stdout.String())
	}
	if !out.Data.DryRun || !out.Data.RemoveConfig || len(out.Data.WouldRemove) < 2 {
		t.Fatalf("dry-run plan mismatch: %#v", out.Data)
	}
}

func TestUninstallSchemaMarksDestructiveAndUsesLocalDryRunExample(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "uninstall"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema uninstall failed: %v stderr=%s", err, stderr.String())
	}
	var out struct {
		Data struct {
			Safety struct {
				Destructive bool `json:"destructive"`
				RequiresYes bool `json:"requires_yes"`
			} `json:"safety"`
			Examples []string `json:"examples"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("decode output: %v output=%s", err, stdout.String())
	}
	if !out.Data.Safety.Destructive || !out.Data.Safety.RequiresYes {
		t.Fatalf("safety mismatch: %#v", out.Data.Safety)
	}
	joined := strings.Join(out.Data.Examples, "\n")
	if strings.Contains(joined, "--request-out") {
		t.Fatalf("local uninstall schema should not suggest --request-out: %s", joined)
	}
	if !strings.Contains(joined, "--dry-run uninstall") {
		t.Fatalf("schema missing dry-run example: %s", joined)
	}
}
