package tmc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/huski-inc/tmcopilot-cli/internal/version"
)

func TestCompareCLIVersion(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want int
	}{
		{a: "v0.1.0", b: "0.1.0-experimental.9", want: 1},
		{a: "0.1.0-experimental.10", b: "0.1.0-experimental.6", want: 1},
		{a: "0.1.1", b: "0.1.0", want: 1},
		{a: "0.1.0-experimental.6", b: "0.1.0-experimental.6", want: 0},
		{a: "0.1.0-experimental.6", b: "0.1.0-experimental.10", want: -1},
	}
	for _, tt := range tests {
		a, ok := parseCLIVersion(tt.a)
		if !ok {
			t.Fatalf("parse %q failed", tt.a)
		}
		b, ok := parseCLIVersion(tt.b)
		if !ok {
			t.Fatalf("parse %q failed", tt.b)
		}
		got := compareCLIVersion(a, b)
		switch {
		case tt.want > 0 && got <= 0:
			t.Fatalf("compare(%q,%q) = %d, want > 0", tt.a, tt.b, got)
		case tt.want < 0 && got >= 0:
			t.Fatalf("compare(%q,%q) = %d, want < 0", tt.a, tt.b, got)
		case tt.want == 0 && got != 0:
			t.Fatalf("compare(%q,%q) = %d, want 0", tt.a, tt.b, got)
		}
	}
}

func TestCheckForCLIUpdateUsesExperimentalChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dist-tags": {"latest": "0.1.0", "experimental": "0.1.0-experimental.6"},
			"versions": {
				"0.1.0": {},
				"0.1.0-experimental.5": {},
				"0.1.0-experimental.6": {}
			}
		}`))
	}))
	defer server.Close()

	result, err := checkForCLIUpdate(context.Background(), updateCheckOptions{
		CurrentVersion: "v0.1.0-experimental.5",
		RegistryURL:    server.URL,
		Timeout:        time.Second,
		Now:            time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("check update: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatalf("update_available = false, result=%#v", result)
	}
	if result.Channel != "experimental" {
		t.Fatalf("channel = %q", result.Channel)
	}
	if result.LatestVersion != "v0.1.0-experimental.6" {
		t.Fatalf("latest_version = %q", result.LatestVersion)
	}
	if result.InstallCommand != "npx --yes @tmcopilot/cli@experimental update" {
		t.Fatalf("install_command = %q", result.InstallCommand)
	}
}

func TestCheckForCLIUpdateStableIgnoresPrereleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dist-tags": {"latest": "0.2.0-experimental.1", "experimental": "0.2.0-experimental.1"},
			"versions": {
				"0.1.0": {},
				"0.1.1": {},
				"0.2.0-experimental.1": {}
			}
		}`))
	}))
	defer server.Close()

	result, err := checkForCLIUpdate(context.Background(), updateCheckOptions{
		CurrentVersion: "0.1.0",
		RegistryURL:    server.URL,
		Timeout:        time.Second,
		Now:            time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("check update: %v", err)
	}
	if result.Channel != "latest" {
		t.Fatalf("channel = %q", result.Channel)
	}
	if result.LatestVersion != "0.1.1" {
		t.Fatalf("latest_version = %q", result.LatestVersion)
	}
	if result.InstallCommand != "npx --yes @tmcopilot/cli@latest update" {
		t.Fatalf("install_command = %q", result.InstallCommand)
	}
}

func TestUpdateCheckCommandWritesResultAndCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dist-tags": {"experimental": "0.1.0-experimental.6"},
			"versions": {
				"0.1.0-experimental.5": {},
				"0.1.0-experimental.6": {}
			}
		}`))
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_UPDATE_REGISTRY_URL", server.URL)
	oldVersion := version.Version
	version.Version = "v0.1.0-experimental.5"
	t.Cleanup(func() { version.Version = oldVersion })

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"update", "check"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("update check failed: %v stderr=%s", err, stderr.String())
	}
	var out struct {
		OK   bool `json:"ok"`
		Data struct {
			CurrentVersion  string `json:"current_version"`
			LatestVersion   string `json:"latest_version"`
			UpdateAvailable bool   `json:"update_available"`
			InstallCommand  string `json:"install_command"`
			CachePath       string `json:"cache_path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("decode output: %v output=%s", err, stdout.String())
	}
	if !out.Data.UpdateAvailable || out.Data.LatestVersion != "v0.1.0-experimental.6" {
		t.Fatalf("update result mismatch: %#v", out.Data)
	}
	if out.Data.InstallCommand != "npx --yes @tmcopilot/cli@experimental update" {
		t.Fatalf("install command = %q", out.Data.InstallCommand)
	}
	if out.Data.CachePath != filepath.Join(home, "update-check.json") {
		t.Fatalf("cache path = %q", out.Data.CachePath)
	}
	raw, err := os.ReadFile(filepath.Join(home, "update-check.json"))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if !bytes.Contains(raw, []byte(`"latest_version": "v0.1.0-experimental.6"`)) {
		t.Fatalf("cache missing latest version: %s", string(raw))
	}
}

func TestUpdateCommandInstallsWhenUpdateAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dist-tags": {"experimental": "0.1.0-experimental.6"},
			"versions": {
				"0.1.0-experimental.5": {},
				"0.1.0-experimental.6": {}
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_UPDATE_REGISTRY_URL", server.URL)
	oldVersion := version.Version
	version.Version = "v0.1.0-experimental.5"
	oldRunner := runUpdateInstaller
	var gotArgs []string
	runUpdateInstaller = func(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		gotArgs = append([]string{}, args...)
		_, _ = stdout.Write([]byte("installer stdout\n"))
		_, _ = stderr.Write([]byte("installer stderr\n"))
		return nil
	}
	t.Cleanup(func() {
		version.Version = oldVersion
		runUpdateInstaller = oldRunner
	})

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"update"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("update failed: %v stderr=%s", err, stderr.String())
	}
	wantArgs := []string{"npx", "--yes", "@tmcopilot/cli@experimental", "update"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("installer args = %#v, want %#v", gotArgs, wantArgs)
	}
	var out struct {
		OK   bool `json:"ok"`
		Data struct {
			InstallAttempted bool   `json:"install_attempted"`
			Installed        bool   `json:"installed"`
			InstallCommand   string `json:"install_command"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("decode output: %v output=%s", err, stdout.String())
	}
	if !out.Data.InstallAttempted || !out.Data.Installed {
		t.Fatalf("install result mismatch: %#v", out.Data)
	}
	if out.Data.InstallCommand != "npx --yes @tmcopilot/cli@experimental update" {
		t.Fatalf("install command = %q", out.Data.InstallCommand)
	}
	if bytes.Contains(stdout.Bytes(), []byte("installer stdout")) || bytes.Contains(stdout.Bytes(), []byte("installer stderr")) {
		t.Fatalf("installer output leaked to stdout: %s", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("installer stdout")) || !bytes.Contains(stderr.Bytes(), []byte("installer stderr")) {
		t.Fatalf("installer output missing from stderr: %s", stderr.String())
	}
}

func TestExecuteNonInteractiveReportsLightweightUpdateOnSuccess(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dist-tags": {"experimental": "0.1.0-experimental.15"},
			"versions": {
				"0.1.0-experimental.14": {},
				"0.1.0-experimental.15": {}
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_UPDATE_REGISTRY_URL", server.URL)
	oldVersion := version.Version
	version.Version = "v0.1.0-experimental.14"
	oldRunner := runUpdateInstaller
	runUpdateInstaller = func(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		t.Fatal("non-interactive lightweight update check should not install")
		return nil
	}
	t.Cleanup(func() {
		version.Version = oldVersion
		runUpdateInstaller = oldRunner
	})

	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{"version"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d stderr=%s", exitCode, stderr.String())
	}
	if requests != 1 {
		t.Fatalf("update check requests = %d, want 1", requests)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"version":"v0.1.0-experimental.14"`)) {
		t.Fatalf("stdout missing version payload: %s", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Update available: tmc v0.1.0-experimental.14 -> v0.1.0-experimental.15")) {
		t.Fatalf("stderr missing update notice: %s", stderr.String())
	}
	if bytes.Contains(stderr.Bytes(), []byte("Installing:")) {
		t.Fatalf("non-interactive update check attempted install: %s", stderr.String())
	}
}

func TestVersionForcesUpdateCheckWhenCacheIsFresh(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dist-tags": {"experimental": "0.1.0-experimental.16"},
			"versions": {
				"0.1.0-experimental.15": {},
				"0.1.0-experimental.16": {}
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_UPDATE_REGISTRY_URL", server.URL)
	if err := saveUpdateCheckCache(updateCheckCache{
		CheckedAt:            time.Now().UTC().Format(time.RFC3339),
		CurrentVersion:       "v0.1.0-experimental.15",
		LatestVersion:        "v0.1.0-experimental.15",
		UpdateCheckSupported: true,
		Channel:              "experimental",
	}); err != nil {
		t.Fatalf("save cache: %v", err)
	}
	oldVersion := version.Version
	version.Version = "v0.1.0-experimental.15"
	t.Cleanup(func() { version.Version = oldVersion })

	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{"version"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d stderr=%s", exitCode, stderr.String())
	}
	if requests != 1 {
		t.Fatalf("update check requests = %d, want 1", requests)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Update available: tmc v0.1.0-experimental.15 -> v0.1.0-experimental.16")) {
		t.Fatalf("stderr missing forced update notice: %s", stderr.String())
	}
}

func TestExecuteNonInteractiveReportsLightweightUpdateAfterParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dist-tags": {"experimental": "0.1.0-experimental.15"},
			"versions": {
				"0.1.0-experimental.14": {},
				"0.1.0-experimental.15": {}
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_UPDATE_REGISTRY_URL", server.URL)
	oldVersion := version.Version
	version.Version = "v0.1.0-experimental.14"
	t.Cleanup(func() { version.Version = oldVersion })

	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{
		"search", "uspto-document",
		"--profile", "test",
		"--serial-number", "88418692",
		"--document-date", "2025-06-23",
		"--document-type", "xml",
		"--document-page-id",
	}, nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1 stderr=%s", exitCode, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout should be empty on parse error: %s", stdout.String())
	}
	output := stderr.String()
	if !strings.Contains(output, `"ok":false`) || !strings.Contains(output, "flag needs an argument: --document-page-id") {
		t.Fatalf("stderr missing structured parse error: %s", output)
	}
	if !strings.Contains(output, "Update available: tmc v0.1.0-experimental.14 -> v0.1.0-experimental.15") {
		t.Fatalf("stderr missing update notice: %s", output)
	}
	if strings.Index(output, `"ok":false`) > strings.Index(output, "Update available:") {
		t.Fatalf("update notice should follow the structured error: %s", output)
	}
}
