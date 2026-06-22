package tmc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/huski-inc/tmcopilot-cli/internal/config"
)

func TestSearchTrademarksCommandBuildsOpenAPIRequest(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method mismatch: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/trademark/search" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization header mismatch: %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"items":[]}}`))
	}))
	defer server.Close()

	home := t.TempDir()
	outFile := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--output", outFile,
		"search", "trademarks",
		"--name", "Nike",
		"--class", "25,35",
		"--owner", "Nike Inc",
		"--limit", "5",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}

	if !reflect.DeepEqual(gotBody["name"], []any{"Nike"}) {
		t.Fatalf("name body mismatch: %#v", gotBody["name"])
	}
	if !reflect.DeepEqual(gotBody["class"], []any{"25", "35"}) {
		t.Fatalf("class body mismatch: %#v", gotBody["class"])
	}
	if !reflect.DeepEqual(gotBody["owners"], []any{"Nike Inc"}) {
		t.Fatalf("owners body mismatch: %#v", gotBody["owners"])
	}
	if gotBody["limit"] != float64(5) {
		t.Fatalf("limit body mismatch: %#v", gotBody["limit"])
	}
	raw, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatalf("decode output file: %v", err)
	}
	if output["ok"] != true {
		t.Fatalf("output ok mismatch: %#v", output)
	}
}

func TestConfigInitUsesPersistentEndpointOverride(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_ENDPOINT", "")
	t.Setenv("TMC_ENDPOINT", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", "https://api.example.test/", "config", "init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Profiles[config.DefaultProfile].Endpoint; got != "https://api.example.test" {
		t.Fatalf("endpoint = %q", got)
	}
}

func TestAuthImportKeyReadsStdinAndUsesPersistentEndpoint(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(" tmc_stdin_key \n"))
	cmd.SetArgs([]string{
		"--endpoint", "https://api.example.test/",
		"auth", "import-key",
		"--name", "local",
		"--api-key-stdin",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Profiles["local"].Endpoint; got != "https://api.example.test" {
		t.Fatalf("endpoint = %q", got)
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles["local"].APIKey; got != "tmc_stdin_key" {
		t.Fatalf("api key = %q", got)
	}
}

func TestSetupStoresAPIKeyAndChecks(t *testing.T) {
	var sawAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tmc_setup_key" {
			t.Fatalf("authorization header mismatch: %q", got)
		}
		sawAuth = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"email":"setup@example.com"}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(" tmc_setup_key \n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"setup",
		"--api-key-stdin",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup failed: %v stderr=%s", err, stderr.String())
	}
	if !sawAuth {
		t.Fatal("auth check was not called")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Profiles[config.DefaultProfile].Endpoint; got != server.URL {
		t.Fatalf("endpoint = %q", got)
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles[config.DefaultProfile].APIKey; got != "tmc_setup_key" {
		t.Fatalf("api key = %q", got)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"auth_method":"api_key"`)) {
		t.Fatalf("stdout missing setup result: %s", stdout.String())
	}
}

func TestAuthLoginAuthorizesInBrowserAndStoresAPIKey(t *testing.T) {
	var sawCreateAuthorization, sawPoll, sawCheck bool
	var gotDeviceUUID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/api-key-authorizations":
			sawCreateAuthorization = true
			if got := r.Header.Get("Authorization"); got != "" {
				t.Fatalf("authorization create Authorization header = %q", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode authorization body: %v", err)
			}
			gotDeviceUUID, _ = body["device_uuid"].(string)
			if gotDeviceUUID == "" {
				t.Fatalf("device_uuid was not sent: %#v", body)
			}
			if body["device_name"] != "Codex CLI" {
				t.Fatalf("device_name = %#v", body["device_name"])
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"authorization_id":"akreq_1","authorization_url":"https://app.example.test/api-key-authorize?request_id=akreq_1","poll_token":"poll_token_1","authorization_expires_in":60,"interval":0}}`))
		case "/api/v1/auth/api-key-authorizations/akreq_1/result":
			sawPoll = true
			if got := r.Header.Get("Authorization"); got != "Bearer poll_token_1" {
				t.Fatalf("poll Authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"status":"approved","api_key":"tmc_authorized_key","key":{"id":"key_1","name":"Codex CLI","bound_device_uuid":"` + gotDeviceUUID + `","bound_device_name":"Codex CLI","key_prefix":"tmc_aut","created_at":1710000000}}}`))
		case "/api/v1/auth/me":
			sawCheck = true
			if got := r.Header.Get("Authorization"); got != "Bearer tmc_authorized_key" {
				t.Fatalf("check Authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"email":"user@example.com"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--no-browser",
		"--device-name", "Codex CLI",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login failed: %v stderr=%s", err, stderr.String())
	}
	if !sawCreateAuthorization || !sawPoll || !sawCheck {
		t.Fatalf("expected create/poll/check calls, got create=%v poll=%v check=%v", sawCreateAuthorization, sawPoll, sawCheck)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Profiles[config.DefaultProfile].DeviceUUID; got == "" || got != gotDeviceUUID {
		t.Fatalf("stored device uuid = %q, sent %q", got, gotDeviceUUID)
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles[config.DefaultProfile].APIKey; got != "tmc_authorized_key" {
		t.Fatalf("stored api key = %q", got)
	}
	for _, leaked := range [][]byte{
		[]byte("tmc_authorized_key"),
		[]byte("poll_token_1"),
		[]byte("key_1"),
		[]byte("tmc_aut"),
		[]byte(gotDeviceUUID),
		[]byte("user@example.com"),
		[]byte(`"key_prefix"`),
		[]byte(`"bound_device_uuid"`),
	} {
		if bytes.Contains(stdout.Bytes(), leaked) {
			t.Fatalf("stdout leaked secret %q: %s", leaked, stdout.String())
		}
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"auth_method":"api_key_authorization"`)) {
		t.Fatalf("stdout missing authorization result: %s", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`https://app.example.test/api-key-authorize?request_id=akreq_1`)) {
		t.Fatalf("stderr missing authorization URL: %s", stderr.String())
	}
}

func TestAuthLoginNoWaitStoresPendingAuthorizationOnly(t *testing.T) {
	var sawCreateAuthorization bool
	var gotDeviceUUID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/api-key-authorizations":
			sawCreateAuthorization = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode authorization body: %v", err)
			}
			gotDeviceUUID, _ = body["device_uuid"].(string)
			if gotDeviceUUID == "" {
				t.Fatalf("device_uuid was not sent: %#v", body)
			}
			if body["device_name"] != "Codex CLI" {
				t.Fatalf("device_name = %#v", body["device_name"])
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"authorization_id":"akreq_1","authorization_url":"https://app.example.test/api-key-authorize?request_id=akreq_1","poll_token":"poll_token_1","authorization_expires_in":60,"interval":0}}`))
		case "/api/v1/auth/api-key-authorizations/akreq_1/result", "/api/v1/auth/me":
			t.Fatalf("no-wait should not poll or check credentials, got path: %s", r.URL.Path)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--no-wait",
		"--device-name", "Codex CLI",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login --no-wait failed: %v stderr=%s", err, stderr.String())
	}
	if !sawCreateAuthorization {
		t.Fatal("authorization request was not created")
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("credentials should not be written during no-wait, stat err=%v", err)
	}
	store, err := loadPendingAuthorizationStore()
	if err != nil {
		t.Fatalf("load pending store: %v", err)
	}
	pending, ok := store.Authorizations["akreq_1"]
	if !ok {
		t.Fatalf("pending authorization not stored: %#v", store.Authorizations)
	}
	if pending.PollToken != "poll_token_1" || pending.DeviceUUID != gotDeviceUUID || pending.DeviceName != "Codex CLI" {
		t.Fatalf("pending authorization mismatch: %#v", pending)
	}

	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			Stored        bool   `json:"stored"`
			ResumeCommand string `json:"resume_command"`
			SetupResume   string `json:"setup_resume"`
			Authorization struct {
				ID               string `json:"id"`
				Status           string `json:"status"`
				AuthorizationURL string `json:"authorization_url"`
			} `json:"authorization"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode no-wait result: %v output=%s", err, stdout.String())
	}
	if result.Data.Stored || result.Data.Authorization.ID != "akreq_1" || result.Data.Authorization.Status != "pending" {
		t.Fatalf("no-wait result mismatch: %#v", result.Data)
	}
	if result.Data.ResumeCommand != "tmc auth login --request-id akreq_1" || result.Data.SetupResume != "tmc setup --request-id akreq_1" {
		t.Fatalf("resume commands mismatch: %#v", result.Data)
	}
	if result.Data.Authorization.AuthorizationURL != "https://app.example.test/api-key-authorize?request_id=akreq_1" {
		t.Fatalf("authorization url = %q", result.Data.Authorization.AuthorizationURL)
	}
	for _, leaked := range [][]byte{
		[]byte("poll_token_1"),
		[]byte(gotDeviceUUID),
	} {
		if bytes.Contains(stdout.Bytes(), leaked) || bytes.Contains(stderr.Bytes(), leaked) {
			t.Fatalf("no-wait output leaked secret %q: stdout=%s stderr=%s", leaked, stdout.String(), stderr.String())
		}
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`tmc auth login --request-id akreq_1`)) {
		t.Fatalf("stderr missing resume command: %s", stderr.String())
	}
}

func TestAuthLoginRequestIDResumesPendingAuthorization(t *testing.T) {
	var createCalls, pollCalls, checkCalls int
	var gotDeviceUUID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/api-key-authorizations":
			createCalls++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode authorization body: %v", err)
			}
			gotDeviceUUID, _ = body["device_uuid"].(string)
			if gotDeviceUUID == "" {
				t.Fatalf("device_uuid was not sent: %#v", body)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"authorization_id":"akreq_1","authorization_url":"https://app.example.test/api-key-authorize?request_id=akreq_1","poll_token":"poll_token_1","authorization_expires_in":60,"interval":0}}`))
		case "/api/v1/auth/api-key-authorizations/akreq_1/result":
			pollCalls++
			if got := r.Header.Get("Authorization"); got != "Bearer poll_token_1" {
				t.Fatalf("poll Authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"status":"approved","api_key":"tmc_authorized_key","key":{"id":"key_1","name":"Codex CLI","bound_device_uuid":"` + gotDeviceUUID + `","bound_device_name":"Codex CLI","key_prefix":"tmc_aut","created_at":1710000000}}}`))
		case "/api/v1/auth/me":
			checkCalls++
			if got := r.Header.Get("Authorization"); got != "Bearer tmc_authorized_key" {
				t.Fatalf("check Authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"email":"user@example.com"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--no-wait",
		"--device-name", "Codex CLI",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login --no-wait failed: %v stderr=%s", err, stderr.String())
	}
	if createCalls != 1 || pollCalls != 0 || checkCalls != 0 {
		t.Fatalf("unexpected calls after no-wait: create=%d poll=%d check=%d", createCalls, pollCalls, checkCalls)
	}
	for _, leaked := range [][]byte{[]byte("poll_token_1"), []byte("tmc_authorized_key")} {
		if bytes.Contains(stdout.Bytes(), leaked) || bytes.Contains(stderr.Bytes(), leaked) {
			t.Fatalf("no-wait output leaked secret %q: stdout=%s stderr=%s", leaked, stdout.String(), stderr.String())
		}
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--request-id", "akreq_1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login --request-id failed: %v stderr=%s", err, stderr.String())
	}
	if createCalls != 1 || pollCalls != 1 || checkCalls != 1 {
		t.Fatalf("unexpected calls after resume: create=%d poll=%d check=%d", createCalls, pollCalls, checkCalls)
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles[config.DefaultProfile].APIKey; got != "tmc_authorized_key" {
		t.Fatalf("stored api key = %q", got)
	}
	store, err := loadPendingAuthorizationStore()
	if err != nil {
		t.Fatalf("load pending store: %v", err)
	}
	if _, ok := store.Authorizations["akreq_1"]; ok {
		t.Fatalf("pending authorization was not removed: %#v", store.Authorizations)
	}
	for _, leaked := range [][]byte{
		[]byte("tmc_authorized_key"),
		[]byte("poll_token_1"),
		[]byte("key_1"),
		[]byte("tmc_aut"),
		[]byte(gotDeviceUUID),
		[]byte("user@example.com"),
		[]byte(`"key_prefix"`),
		[]byte(`"bound_device_uuid"`),
	} {
		if bytes.Contains(stdout.Bytes(), leaked) || bytes.Contains(stderr.Bytes(), leaked) {
			t.Fatalf("resume output leaked secret %q: stdout=%s stderr=%s", leaked, stdout.String(), stderr.String())
		}
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"stored":true`)) ||
		!bytes.Contains(stdout.Bytes(), []byte(`"id":"akreq_1"`)) ||
		!bytes.Contains(stdout.Bytes(), []byte(`"verified":true`)) {
		t.Fatalf("resume result missing status: %s", stdout.String())
	}
}

func TestSetupNoWaitPrintsSetupResumeCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/api-key-authorizations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"authorization_id":"akreq_setup","authorization_url":"https://app.example.test/api-key-authorize?request_id=akreq_setup","poll_token":"poll_token_setup","authorization_expires_in":60,"interval":0}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"setup",
		"--no-wait",
		"--device-name", "Codex CLI",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup --no-wait failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`tmc setup --request-id akreq_setup`)) {
		t.Fatalf("stderr missing setup resume command: %s", stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"setup_resume":"tmc setup --request-id akreq_setup"`)) {
		t.Fatalf("stdout missing setup resume command: %s", stdout.String())
	}
	for _, leaked := range [][]byte{[]byte("poll_token_setup")} {
		if bytes.Contains(stdout.Bytes(), leaked) || bytes.Contains(stderr.Bytes(), leaked) {
			t.Fatalf("setup no-wait output leaked secret %q: stdout=%s stderr=%s", leaked, stdout.String(), stderr.String())
		}
	}
}

func TestAuthLoginAuthorizationRequestOutAndIdempotencyHeader(t *testing.T) {
	var sawCreateAuthorization bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/api-key-authorizations":
			sawCreateAuthorization = true
			if got := r.Header.Get("Idempotency-Key"); got != "idem-1" {
				t.Fatalf("Idempotency-Key = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"authorization_id":"akreq_1","authorization_url":"https://app.example.test/api-key-authorize?request_id=akreq_1","poll_token":"poll_token_1","authorization_expires_in":60,"interval":0}}`))
		case "/api/v1/auth/api-key-authorizations/akreq_1/result":
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"status":"approved","api_key":"tmc_authorized_key","key":{"id":"key_1","name":"Codex CLI","key_prefix":"tmc_aut"}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")
	requestOut := filepath.Join(t.TempDir(), "auth-plan.json")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--request-out", requestOut,
		"--idempotency-key", "idem-1",
		"auth", "login",
		"--no-browser",
		"--device-name", "Codex CLI",
		"--check=false",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login failed: %v stderr=%s", err, stderr.String())
	}
	if !sawCreateAuthorization {
		t.Fatal("authorization request was not created")
	}
	raw, err := os.ReadFile(requestOut)
	if err != nil {
		t.Fatalf("read request out: %v", err)
	}
	if !bytes.Contains(raw, []byte(`/auth/api-key-authorizations`)) ||
		!bytes.Contains(raw, []byte(`"auth_method": "api_key_authorization"`)) {
		t.Fatalf("request out missing authorization plan: %s", string(raw))
	}
	for _, leaked := range [][]byte{[]byte("tmc_authorized_key"), []byte("poll_token_1")} {
		if bytes.Contains(raw, leaked) || bytes.Contains(stdout.Bytes(), leaked) {
			t.Fatalf("secret %q leaked: request_out=%s stdout=%s", leaked, string(raw), stdout.String())
		}
	}
}

func TestAuthLoginLegacyPasswordCreatesAndStoresCLIAPIKey(t *testing.T) {
	var sawLogin, sawCreateKey, sawCheck bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			sawLogin = true
			if got := r.Header.Get("Authorization"); got != "" {
				t.Fatalf("login authorization header = %q", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode login body: %v", err)
			}
			if body["email"] != "user@example.com" || body["password"] != "secret" {
				t.Fatalf("login body mismatch: %#v", body)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"tokens":{"access_token":"jwt_login_token","expires_in":3600},"user":{"email":"user@example.com"}}}`))
		case "/api/v1/auth/api-keys":
			sawCreateKey = true
			if got := r.Header.Get("Authorization"); got != "Bearer jwt_login_token" {
				t.Fatalf("api key authorization header = %q", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode api key body: %v", err)
			}
			if body["name"] != "codex cli" {
				t.Fatalf("api key name = %#v", body["name"])
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"raw_key":"tmc_created_key","key":{"id":"key_1","name":"codex cli","key_prefix":"tmc_cre"}}}`))
		case "/api/v1/auth/me":
			sawCheck = true
			if got := r.Header.Get("Authorization"); got != "Bearer tmc_created_key" {
				t.Fatalf("check authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"email":"user@example.com"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("secret\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--email", "user@example.com",
		"--password-stdin",
		"--key-name", "codex cli",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login failed: %v stderr=%s", err, stderr.String())
	}
	if !sawLogin || !sawCreateKey || !sawCheck {
		t.Fatalf("expected login/create/check calls, got login=%v create=%v check=%v", sawLogin, sawCreateKey, sawCheck)
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles[config.DefaultProfile].APIKey; got != "tmc_created_key" {
		t.Fatalf("stored api key = %q", got)
	}
	if bytes.Contains(stdout.Bytes(), []byte("tmc_created_key")) {
		t.Fatalf("stdout leaked raw api key: %s", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("secret")) {
		t.Fatalf("stdout leaked password: %s", stdout.String())
	}
	for _, leaked := range [][]byte{
		[]byte("key_1"),
		[]byte("tmc_cre"),
		[]byte("user@example.com"),
		[]byte(`"key_prefix"`),
	} {
		if bytes.Contains(stdout.Bytes(), leaked) {
			t.Fatalf("stdout leaked credential metadata %q: %s", leaked, stdout.String())
		}
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"auth_method":"login_api_key"`)) {
		t.Fatalf("stdout missing login setup result: %s", stdout.String())
	}
}

func TestSetupDryRunDoesNotCallAPIOrWriteCredentials(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("tmc_dryrun_key\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--dry-run",
		"setup",
		"--api-key-stdin",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup dry-run failed: %v stderr=%s", err, stderr.String())
	}
	if called {
		t.Fatal("server was called during setup dry-run")
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("credentials should not be written during dry-run, stat err=%v", err)
	}
	if bytes.Contains(stdout.Bytes(), []byte("tmc_dryrun_key")) {
		t.Fatalf("dry-run leaked api key: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"dry_run":true`)) {
		t.Fatalf("dry-run output missing marker: %s", stdout.String())
	}
}

func TestSetupDryRunDoesNotImportEnvAPIKeyByDefault(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "tmc_env_key")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dry-run", "setup"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup dry-run failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"auth_method":"api_key_authorization"`)) ||
		!bytes.Contains(stdout.Bytes(), []byte(`"/auth/api-key-authorizations"`)) {
		t.Fatalf("setup should plan API key authorization flow instead of env API key import: %s", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte(`"api_key_source":"env"`)) {
		t.Fatalf("setup should not import env API key by default: %s", stdout.String())
	}
}

func TestAuthLoginLegacyPasswordDryRunDoesNotCallAPIOrReadPassword(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("secret\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--dry-run",
		"auth", "login",
		"--email", "user@example.com",
		"--password-stdin",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login dry-run failed: %v stderr=%s", err, stderr.String())
	}
	if called {
		t.Fatal("server was called during auth login dry-run")
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("credentials should not be written during dry-run, stat err=%v", err)
	}
	if bytes.Contains(stdout.Bytes(), []byte("secret")) {
		t.Fatalf("dry-run leaked password: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"/auth/login"`)) || !bytes.Contains(stdout.Bytes(), []byte(`redacted`)) {
		t.Fatalf("dry-run output missing redacted login plan: %s", stdout.String())
	}
}

func TestAuthLogoutRemovesLocalCredential(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cfg := config.DefaultConfig()
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := config.SaveCredentials(&config.Credentials{Profiles: map[string]config.Credential{
		config.DefaultProfile: {APIKey: "tmc_logout_key"},
	}}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"auth", "logout"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth logout failed: %v stderr=%s", err, stderr.String())
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if _, ok := creds.Profiles[config.DefaultProfile]; ok {
		t.Fatalf("credential was not removed: %#v", creds.Profiles)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"removed":1`)) {
		t.Fatalf("logout output missing removal count: %s", stdout.String())
	}
}

func TestAuthLoginLegacyPasswordWritesRedactedRequestOutAndIdempotencyHeader(t *testing.T) {
	var sawCreateKey bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"tokens":{"access_token":"jwt_login_token"}}}`))
		case "/api/v1/auth/api-keys":
			sawCreateKey = true
			if got := r.Header.Get("Idempotency-Key"); got != "idem-1" {
				t.Fatalf("Idempotency-Key = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"raw_key":"tmc_created_key","key":{"id":"key_1","name":"codex cli","key_prefix":"tmc_cre"}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")
	requestOut := filepath.Join(t.TempDir(), "auth-plan.json")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("secret\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--request-out", requestOut,
		"--idempotency-key", "idem-1",
		"auth", "login",
		"--email", "user@example.com",
		"--password-stdin",
		"--key-name", "codex cli",
		"--check=false",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login failed: %v stderr=%s", err, stderr.String())
	}
	if !sawCreateKey {
		t.Fatal("api key creation was not called")
	}
	raw, err := os.ReadFile(requestOut)
	if err != nil {
		t.Fatalf("read request out: %v", err)
	}
	if !bytes.Contains(raw, []byte(`/auth/login`)) || !bytes.Contains(raw, []byte(`redacted`)) {
		t.Fatalf("request out missing redacted login plan: %s", string(raw))
	}
	for _, leaked := range [][]byte{[]byte("secret"), []byte("tmc_created_key")} {
		if bytes.Contains(raw, leaked) {
			t.Fatalf("request out leaked secret %q: %s", leaked, string(raw))
		}
	}
}

func TestAuthLoginLegacyPasswordCheckFailureStoresCredentialAsUnverified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"tokens":{"access_token":"jwt_login_token"}}}`))
		case "/api/v1/auth/api-keys":
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"raw_key":"tmc_unverified_key","key":{"id":"key_1","name":"codex cli","key_prefix":"tmc_unv"}}}`))
		case "/api/v1/auth/me":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"code":50300,"message":{"title":"Unavailable","text":"temporary failure"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("secret\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--email", "user@example.com",
		"--password-stdin",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login should keep generated key as unverified instead of failing: %v stderr=%s", err, stderr.String())
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles[config.DefaultProfile].APIKey; got != "tmc_unverified_key" {
		t.Fatalf("stored api key = %q", got)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"verified":false`)) {
		t.Fatalf("stdout missing unverified marker: %s", stdout.String())
	}
}

func TestAuthImportKeyCheckFailureDoesNotStoreCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":40100,"message":{"title":"Unauthorized","text":"invalid key"}}`))
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("bad-key\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "import-key",
		"--api-key-stdin",
		"--check",
	})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected auth import-key check failure")
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("credentials should not be written after failed check, stat err=%v", err)
	}
}

func TestPathArgumentsAreEscaped(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
	}{
		{
			name:     "portfolio trademark get",
			args:     []string{"portfolio", "trademarks", "get", "abc/def"},
			wantPath: "/api/v1/portfolio/trademarks/abc%2Fdef",
		},
		{
			name:     "auth api key revoke",
			args:     []string{"--yes", "auth", "api-keys", "revoke", "key/1"},
			wantPath: "/api/v1/auth/api-keys/key%2F1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.EscapedPath()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"ok":true}}`))
			}))
			defer server.Close()

			t.Setenv("TMCOPILOT_HOME", t.TempDir())
			t.Setenv("TMCOPILOT_API_KEY", "test-key")

			cmd := NewRootCommand()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			args := append([]string{"--endpoint", server.URL}, tt.args...)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

func TestDryRunDoesNotCallAPIAndWritesRequestOut(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	requestOut := filepath.Join(t.TempDir(), "request.json")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--dry-run",
		"--request-out", requestOut,
		"api", "POST", "/auth/api-keys",
		"--data", `{"name":"dry"}`,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	if called {
		t.Fatal("server was called during dry-run")
	}
	raw, err := os.ReadFile(requestOut)
	if err != nil {
		t.Fatalf("read request out: %v", err)
	}
	if !bytes.Contains(raw, []byte(`"method": "POST"`)) || !bytes.Contains(raw, []byte(`"path": "/auth/api-keys"`)) {
		t.Fatalf("request out mismatch: %s", raw)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"dry"`)) {
		t.Fatalf("stdout missing dry-run body: %s", stdout.String())
	}
}

func TestDestructiveRequestRequiresYes(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", "https://api.example.test", "api", "DELETE", "/gap-analyses/id-1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("DELETE without --yes error = nil")
	}
	if !strings.Contains(stderr.String(), `"type":"cli_error"`) {
		t.Fatalf("stderr missing typed cli error: %s", stderr.String())
	}
}

func TestAPICatalogFiltersEndpoints(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"api", "catalog", "--tag", "auth", "--coverage", "typed"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`/auth/me`)) {
		t.Fatalf("catalog output missing auth typed endpoint: %s", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte(`/competitors`)) {
		t.Fatalf("catalog output includes non-auth endpoint: %s", stdout.String())
	}
}

func TestSchemaShowsEndpointMetadata(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "search", "trademarks"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema failed: %v stderr=%s", err, stderr.String())
	}
	for _, want := range []string{
		`"command":"tmc search trademarks"`,
		`"path":"/trademark/search"`,
		`"--class"`,
	} {
		if !bytes.Contains(stdout.Bytes(), []byte(want)) {
			t.Fatalf("schema output missing %q: %s", want, stdout.String())
		}
	}
	if bytes.Contains(stdout.Bytes(), []byte(`"definitions"`)) {
		t.Fatalf("schema output should not include raw OpenAPI definitions by default: %s", stdout.String())
	}
}

func TestSchemaIncludesAgentSafetyMetadata(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "search", "trademarks"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema failed: %v stderr=%s", err, stderr.String())
	}
	var searchSchema struct {
		OK   bool `json:"ok"`
		Data struct {
			Safety struct {
				ReadOnly    bool `json:"read_only"`
				SideEffect  bool `json:"side_effect"`
				Destructive bool `json:"destructive"`
			} `json:"safety"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &searchSchema); err != nil {
		t.Fatalf("decode schema: %v output=%s", err, stdout.String())
	}
	if !searchSchema.Data.Safety.ReadOnly || searchSchema.Data.Safety.SideEffect || searchSchema.Data.Safety.Destructive {
		t.Fatalf("search safety metadata mismatch: %#v", searchSchema.Data.Safety)
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "gap", "delete"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete schema failed: %v stderr=%s", err, stderr.String())
	}
	var deleteSchema struct {
		OK   bool `json:"ok"`
		Data struct {
			Safety struct {
				SideEffect  bool `json:"side_effect"`
				Destructive bool `json:"destructive"`
				RequiresYes bool `json:"requires_yes"`
			} `json:"safety"`
			Examples []string `json:"examples"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &deleteSchema); err != nil {
		t.Fatalf("decode delete schema: %v output=%s", err, stdout.String())
	}
	if !deleteSchema.Data.Safety.SideEffect || !deleteSchema.Data.Safety.Destructive || !deleteSchema.Data.Safety.RequiresYes {
		t.Fatalf("delete safety metadata mismatch: %#v", deleteSchema.Data.Safety)
	}
	if len(deleteSchema.Data.Examples) == 0 {
		t.Fatalf("delete schema missing examples: %s", stdout.String())
	}
}

func TestSchemaIncludesPaginationMetadata(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "portfolio", "trademarks", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema failed: %v stderr=%s", err, stderr.String())
	}
	var schema struct {
		OK   bool `json:"ok"`
		Data struct {
			Pagination struct {
				SupportsPageAll   bool     `json:"supports_page_all"`
				SupportsFields    bool     `json:"supports_fields"`
				SupportsManifest  bool     `json:"supports_manifest"`
				RecommendedFormat string   `json:"recommended_format"`
				Flags             []string `json:"flags"`
			} `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("decode schema: %v output=%s", err, stdout.String())
	}
	if !schema.Data.Pagination.SupportsPageAll || !schema.Data.Pagination.SupportsFields || !schema.Data.Pagination.SupportsManifest {
		t.Fatalf("pagination metadata mismatch: %#v", schema.Data.Pagination)
	}
	if schema.Data.Pagination.RecommendedFormat != "ndjson" {
		t.Fatalf("recommended format = %q", schema.Data.Pagination.RecommendedFormat)
	}
}

func TestAgentBootstrapReturnsMachineReadableGuidance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization header mismatch: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"email":"agent@example.com"}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", server.URL, "agent", "bootstrap", "--check"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent bootstrap failed: %v stderr=%s", err, stderr.String())
	}
	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			CLI struct {
				Commands []string `json:"commands"`
			} `json:"cli"`
			Auth struct {
				Configured bool `json:"configured"`
				Verified   bool `json:"verified"`
			} `json:"auth"`
			Skills []struct {
				Name string `json:"name"`
			} `json:"skills"`
			Discovery struct {
				Schema string `json:"schema"`
			} `json:"discovery"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode bootstrap: %v output=%s", err, stdout.String())
	}
	if !result.Data.Auth.Configured || !result.Data.Auth.Verified {
		t.Fatalf("auth metadata mismatch: %#v", result.Data.Auth)
	}
	if !reflect.DeepEqual(result.Data.CLI.Commands, []string{"tmc", "tmcopilot"}) {
		t.Fatalf("commands = %#v", result.Data.CLI.Commands)
	}
	if len(result.Data.Skills) == 0 {
		t.Fatalf("bootstrap missing skills: %s", stdout.String())
	}
	if result.Data.Discovery.Schema != "tmc schema <command...>" {
		t.Fatalf("schema discovery = %q", result.Data.Discovery.Schema)
	}
	if bytes.Contains(stdout.Bytes(), []byte("agent@example.com")) {
		t.Fatalf("bootstrap leaked user profile data: %s", stdout.String())
	}
}

func TestSchemaOpenAPIFlagIncludesRawDefinitions(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "--openapi", "search", "trademarks"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema --openapi failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"definitions"`)) {
		t.Fatalf("schema --openapi output missing definitions: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"internal_protocol_rest_handler.searchByTextRequest"`)) {
		t.Fatalf("schema --openapi output missing request definition: %s", stdout.String())
	}
}

func TestAPIEndpointSchemaShowsRawEndpointMetadata(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"api", "schema", "POST", "/trademark/search"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("api schema failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"path":"/trademark/search"`)) {
		t.Fatalf("api schema output missing endpoint: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"internal_protocol_rest_handler.searchByTextRequest"`)) {
		t.Fatalf("api schema output missing definition: %s", stdout.String())
	}
}

func TestSchemaRejectsRawEndpointFormWithHint(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "POST", "/trademark/search"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected raw endpoint schema form to fail")
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`tmc schema expects a CLI command path`)) {
		t.Fatalf("stderr missing command-path hint: %s", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`tmc api schema POST /trademark/search`)) {
		t.Fatalf("stderr missing api schema hint: %s", stderr.String())
	}
}

func TestSchemaAuthLogoutIsLocalOnly(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "auth", "logout"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema auth logout failed: %v stderr=%s", err, stderr.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte(`/auth/logout`)) {
		t.Fatalf("local logout schema should not expose backend logout endpoint: %s", stdout.String())
	}
}

func TestSearchAliasesBuildRequests(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dry-run", "search", "trademark", "--name", "Nike", "--class", "25"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("alias dry-run failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"/trademark/search"`)) {
		t.Fatalf("alias dry-run output missing endpoint: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"class":["25"]`)) {
		t.Fatalf("alias dry-run output missing class body: %s", stdout.String())
	}
}

func TestExecuteUnknownCommandWritesStructuredSuggestion(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{"search", "trademarkz", "--name", "Nike"}, nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`"type":"validation_error"`)) {
		t.Fatalf("stderr missing validation error: %s", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`trademarks`)) {
		t.Fatalf("stderr missing suggestion: %s", stderr.String())
	}
}

func TestExecuteUnknownFlagWritesStructuredSuggestion(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{"search", "trademarks", "--clas", "25"}, nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`"type":"validation_error"`)) {
		t.Fatalf("stderr missing validation error: %s", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`--class`)) {
		t.Fatalf("stderr missing flag suggestion: %s", stderr.String())
	}
}

func TestAPIDownloadWritesRawResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test"); got != "yes" {
			t.Fatalf("X-Test header = %q", got)
		}
		_, _ = w.Write([]byte("raw-body"))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")
	outFile := filepath.Join(t.TempDir(), "downloads", "body.txt")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--output", outFile,
		"api", "download", "GET", "/files/raw",
		"--header", "X-Test=yes",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	raw, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(raw) != "raw-body" {
		t.Fatalf("download content = %q", raw)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"bytes":8`)) {
		t.Fatalf("summary missing bytes: %s", stdout.String())
	}
}

func TestSkillsListAndRead(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	listCmd := NewRootCommand()
	var listOut, listErr bytes.Buffer
	listCmd.SetOut(&listOut)
	listCmd.SetErr(&listErr)
	listCmd.SetArgs([]string{"skills", "list"})
	if err := listCmd.Execute(); err != nil {
		t.Fatalf("skills list failed: %v stderr=%s", err, listErr.String())
	}
	if !bytes.Contains(listOut.Bytes(), []byte(`tmc-trademark-search`)) {
		t.Fatalf("skills list missing trademark skill: %s", listOut.String())
	}

	readCmd := NewRootCommand()
	var readOut, readErr bytes.Buffer
	readCmd.SetOut(&readOut)
	readCmd.SetErr(&readErr)
	readCmd.SetArgs([]string{"skills", "read", "tmc-trademark-search"})
	if err := readCmd.Execute(); err != nil {
		t.Fatalf("skills read failed: %v stderr=%s", err, readErr.String())
	}
	if !bytes.Contains(readOut.Bytes(), []byte("tmc search trademarks")) {
		t.Fatalf("skills read missing command guidance: %s", readOut.String())
	}
}

func TestSkillsReadReferenceAsJSON(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"skills", "read", "tmc-trademark-search/references/search-fields.md", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("skills read reference failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"path":"references/search-fields.md"`)) {
		t.Fatalf("json output missing reference path: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`--class 25,35`)) {
		t.Fatalf("json output missing reference content: %s", stdout.String())
	}
}

func TestSkillsReadSupportsOutputFile(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	outFile := filepath.Join(t.TempDir(), "skill.md")
	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--output", outFile, "skills", "read", "tmc-trademark-search"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("skills read output failed: %v stderr=%s", err, stderr.String())
	}
	raw, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !bytes.Contains(raw, []byte("tmc search trademarks")) {
		t.Fatalf("output file missing skill content: %s", string(raw))
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"bytes":`)) {
		t.Fatalf("stdout missing output summary: %s", stdout.String())
	}
}

func TestSkillsRejectsInvalidTarget(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"skills", "read", "../tmc-shared"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected invalid skill target to fail")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("invalid skill name")) {
		t.Fatalf("stderr missing invalid name error: %s", stderr.String())
	}
}

func TestDoctorAuthFailsWithoutAPIKey(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "auth"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("doctor auth error = nil")
	}
	if !strings.Contains(err.Error(), "doctor auth failed") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"ok":false`) {
		t.Fatalf("stdout missing failed auth result: %s", stdout.String())
	}
}

func TestDoctorNetworkFailsOnBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":50000,"message":{"title":"Error","text":"broken"}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", server.URL, "doctor", "network"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("doctor network error = nil")
	}
	if !strings.Contains(err.Error(), "doctor network failed") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"network":{"message":"http 500`) {
		t.Fatalf("stdout missing failed network result: %s", stdout.String())
	}
}
