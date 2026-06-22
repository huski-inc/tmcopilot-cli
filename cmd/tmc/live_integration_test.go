package tmc

import (
	"bytes"
	"os"
	"testing"
)

func TestLiveDoctorNetwork(t *testing.T) {
	if os.Getenv("TMCOPILOT_LIVE") != "1" {
		t.Skip("set TMCOPILOT_LIVE=1 to run live integration tests")
	}
	endpoint := os.Getenv("TMCOPILOT_LIVE_ENDPOINT")
	if endpoint == "" {
		t.Fatal("TMCOPILOT_LIVE_ENDPOINT is required")
	}

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", endpoint, "doctor", "network"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor network failed: %v stderr=%s stdout=%s", err, stderr.String(), stdout.String())
	}
}

func TestLiveAuthWhoami(t *testing.T) {
	if os.Getenv("TMCOPILOT_LIVE") != "1" {
		t.Skip("set TMCOPILOT_LIVE=1 to run live integration tests")
	}
	endpoint := os.Getenv("TMCOPILOT_LIVE_ENDPOINT")
	apiKey := os.Getenv("TMCOPILOT_LIVE_API_KEY")
	if endpoint == "" || apiKey == "" {
		t.Fatal("TMCOPILOT_LIVE_ENDPOINT and TMCOPILOT_LIVE_API_KEY are required")
	}

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", apiKey)
	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", endpoint, "auth", "whoami"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth whoami failed: %v stderr=%s stdout=%s", err, stderr.String(), stdout.String())
	}
}
