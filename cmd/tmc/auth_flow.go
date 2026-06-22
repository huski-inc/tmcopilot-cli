package tmc

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/huski-inc/tmcopilot-cli/internal/client"
	"github.com/huski-inc/tmcopilot-cli/internal/config"
	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
	"github.com/huski-inc/tmcopilot-cli/internal/version"
)

type loginCredentialOptions struct {
	Email             string
	PasswordStdin     bool
	TurnstileResponse string
	KeyName           string
	DeviceName        string
	ExpiresIn         int64
	NoBrowser         bool
	NoWait            bool
	RequestID         string
	Check             bool
}

type storedCredential struct {
	ProfileName string
	Profile     config.Profile
}

func detectAPIKeyInputSource(apiKey string, apiKeyStdin bool, allowEnv bool) (string, bool) {
	switch {
	case strings.TrimSpace(apiKey) != "":
		return "flag", true
	case apiKeyStdin:
		return "stdin", true
	case allowEnv:
		if _, ok := config.EnvAPIKey(); ok {
			return "env", true
		}
	}
	return "", false
}

func readAPIKeyInput(cmd *cobra.Command, apiKey string, apiKeyStdin bool, allowEnv bool) (string, bool, error) {
	if strings.TrimSpace(apiKey) != "" && apiKeyStdin {
		return "", false, fmt.Errorf("use only one of --api-key or --api-key-stdin")
	}
	if apiKeyStdin {
		raw, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", false, err
		}
		apiKey = string(raw)
	}
	if strings.TrimSpace(apiKey) == "" && allowEnv {
		if envKey, ok := config.EnvAPIKey(); ok {
			apiKey = envKey
		}
	}
	apiKey = strings.TrimSpace(apiKey)
	return apiKey, apiKey != "", nil
}

func writeLoginSetupDryRun(cmd *cobra.Command, opts *globalOptions, loginOpts loginCredentialOptions) error {
	rt, err := commandRuntime(cmd, opts, false)
	if err != nil {
		return err
	}
	if err := validateAuthorizationOptions(loginOpts); err != nil {
		return err
	}
	plan, err := buildLoginSetupPlan(rt, loginOpts)
	if err != nil {
		return err
	}
	plan["dry_run"] = true
	return writeAuthFlowPlan(cmd, opts, rt, plan)
}

func buildLoginSetupPlan(rt *runtimeContext, loginOpts loginCredentialOptions) (map[string]any, error) {
	if strings.TrimSpace(loginOpts.RequestID) != "" {
		return buildResumeAuthorizationPlan(rt, loginOpts), nil
	}
	if usesLegacyPasswordLogin(loginOpts) {
		return buildLegacyLoginSetupPlan(rt, loginOpts), nil
	}
	if loginOpts.ExpiresIn > 0 {
		return nil, fmt.Errorf("--expires-in is only supported with --email/--password-stdin legacy login; choose API key expiry in the browser authorization page")
	}
	deviceUUID := strings.TrimSpace(rt.Profile.DeviceUUID)
	if deviceUUID == "" {
		deviceUUID = "<generated_device_uuid>"
	}
	deviceName := resolveAuthorizationDeviceName(loginOpts)
	actions := []map[string]any{
		{
			"type":   "http_request",
			"method": "POST",
			"path":   "/auth/api-key-authorizations",
			"body": map[string]any{
				"device_uuid": deviceUUID,
				"device_name": deviceName,
			},
		},
		{
			"type": "open_browser",
			"url":  "<authorization_url>",
			"skip": loginOpts.NoBrowser,
		},
		{
			"type":        "http_poll",
			"method":      "GET",
			"path":        "/auth/api-key-authorizations/<authorization_id>/result",
			"auth_source": "<poll_token>",
		},
		{
			"type":    "write_local_config",
			"profile": rt.ProfileName,
		},
		{
			"type":       "write_local_credentials",
			"profile":    rt.ProfileName,
			"credential": "<authorized_api_key>",
		},
	}
	if loginOpts.NoWait {
		actions = actions[:1]
		actions = append(actions, map[string]any{
			"type":       "write_pending_authorization",
			"profile":    rt.ProfileName,
			"request_id": "<authorization_id>",
			"path":       mustPendingAuthorizationsPath(),
		}, map[string]any{
			"type":           "print_authorization_url",
			"url":            "<authorization_url>",
			"resume_command": "tmc setup --request-id <authorization_id>",
		})
	} else if loginOpts.Check {
		actions = append(actions, map[string]any{
			"type":        "http_request",
			"method":      "GET",
			"path":        "/auth/me",
			"auth_source": "authorized_api_key",
		})
	}
	return map[string]any{
		"profile":      rt.ProfileName,
		"endpoint":     rt.Profile.Endpoint,
		"workspace_id": rt.Profile.WorkspaceID,
		"auth_method":  "api_key_authorization",
		"device_uuid":  deviceUUID,
		"device_name":  deviceName,
		"no_wait":      loginOpts.NoWait,
		"actions":      actions,
	}, nil
}

func buildResumeAuthorizationPlan(rt *runtimeContext, loginOpts loginCredentialOptions) map[string]any {
	requestID := strings.TrimSpace(loginOpts.RequestID)
	actions := []map[string]any{
		{
			"type":       "read_pending_authorization",
			"profile":    rt.ProfileName,
			"request_id": requestID,
			"path":       mustPendingAuthorizationsPath(),
		},
		{
			"type":        "http_poll",
			"method":      "GET",
			"path":        "/auth/api-key-authorizations/" + requestID + "/result",
			"auth_source": "local_pending_poll_token",
		},
		{
			"type":    "write_local_config",
			"profile": rt.ProfileName,
		},
		{
			"type":       "write_local_credentials",
			"profile":    rt.ProfileName,
			"credential": "<authorized_api_key>",
		},
		{
			"type":       "delete_pending_authorization",
			"request_id": requestID,
			"path":       mustPendingAuthorizationsPath(),
		},
	}
	if loginOpts.Check {
		actions = append(actions, map[string]any{
			"type":        "http_request",
			"method":      "GET",
			"path":        "/auth/me",
			"auth_source": "authorized_api_key",
		})
	}
	return map[string]any{
		"profile":      rt.ProfileName,
		"endpoint":     rt.Profile.Endpoint,
		"workspace_id": rt.Profile.WorkspaceID,
		"auth_method":  "api_key_authorization",
		"request_id":   requestID,
		"resume":       true,
		"actions":      actions,
	}
}

func buildLegacyLoginSetupPlan(rt *runtimeContext, loginOpts loginCredentialOptions) map[string]any {
	body := map[string]any{
		"email":    strings.TrimSpace(loginOpts.Email),
		"password": "<redacted>",
	}
	if strings.TrimSpace(loginOpts.TurnstileResponse) != "" {
		body["cf_turnstile_response"] = "<redacted>"
	}
	createKeyBody := map[string]any{
		"name": defaultCLIAPIKeyName(),
	}
	if strings.TrimSpace(loginOpts.KeyName) != "" {
		createKeyBody["name"] = strings.TrimSpace(loginOpts.KeyName)
	}
	if loginOpts.ExpiresIn > 0 {
		createKeyBody["expires_in"] = loginOpts.ExpiresIn
	}
	actions := []map[string]any{
		{
			"type":   "http_request",
			"method": "POST",
			"path":   "/auth/login",
			"body":   body,
		},
		{
			"type":        "http_request",
			"method":      "POST",
			"path":        "/auth/api-keys",
			"auth_source": "login_access_token",
			"body":        createKeyBody,
		},
		{
			"type":    "write_local_config",
			"profile": rt.ProfileName,
		},
		{
			"type":       "write_local_credentials",
			"profile":    rt.ProfileName,
			"credential": "<generated_api_key>",
		},
	}
	if loginOpts.Check {
		actions = append(actions, map[string]any{
			"type":        "http_request",
			"method":      "GET",
			"path":        "/auth/me",
			"auth_source": "generated_api_key",
		})
	}
	return map[string]any{
		"profile":      rt.ProfileName,
		"endpoint":     rt.Profile.Endpoint,
		"workspace_id": rt.Profile.WorkspaceID,
		"auth_method":  "login_api_key",
		"actions":      actions,
	}
}

func usesLegacyPasswordLogin(loginOpts loginCredentialOptions) bool {
	return strings.TrimSpace(loginOpts.Email) != "" ||
		loginOpts.PasswordStdin ||
		strings.TrimSpace(loginOpts.TurnstileResponse) != ""
}

func resolveAuthorizationDeviceName(loginOpts loginCredentialOptions) string {
	for _, value := range []string{loginOpts.DeviceName, loginOpts.KeyName} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return defaultCLIAPIKeyName()
}

func writeAPIKeySetupDryRun(cmd *cobra.Command, opts *globalOptions, apiKeySource string, check bool, profileOverride string) error {
	rt, err := commandRuntimeForProfile(cmd, opts, profileOverride, false)
	if err != nil {
		return err
	}
	plan := buildAPIKeySetupPlan(rt, apiKeySource, check)
	plan["dry_run"] = true
	return writeAuthFlowPlan(cmd, opts, rt, plan)
}

func buildAPIKeySetupPlan(rt *runtimeContext, apiKeySource string, check bool) map[string]any {
	actions := []map[string]any{
		{
			"type":    "write_local_config",
			"profile": rt.ProfileName,
		},
		{
			"type":           "write_local_credentials",
			"profile":        rt.ProfileName,
			"credential":     "<provided_api_key>",
			"api_key_source": apiKeySource,
		},
	}
	if check {
		actions = append(actions, map[string]any{
			"type":        "http_request",
			"method":      "GET",
			"path":        "/auth/me",
			"auth_source": "provided_api_key",
		})
	}
	return map[string]any{
		"profile":          rt.ProfileName,
		"endpoint":         rt.Profile.Endpoint,
		"workspace_id":     rt.Profile.WorkspaceID,
		"auth_method":      "api_key",
		"api_key_source":   apiKeySource,
		"credential_store": mustCredentialsPath(),
		"config_path":      mustConfigPath(),
		"actions":          actions,
	}
}

func writeAuthFlowPlan(cmd *cobra.Command, opts *globalOptions, rt *runtimeContext, plan map[string]any) error {
	if opts != nil && opts.requestOut != "" {
		if err := writeJSONPlan(opts.requestOut, plan); err != nil {
			return err
		}
	}
	return writeResult(rt, plan, nil)
}

func storeAPIKeyForRuntime(rt *runtimeContext, apiKey string) (storedCredential, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return storedCredential{}, fmt.Errorf("api key is required")
	}
	cfg := rt.Config
	cfg.Profiles[rt.ProfileName] = rt.Profile
	cfg.CurrentProfile = rt.ProfileName
	if err := config.Save(cfg); err != nil {
		return storedCredential{}, err
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return storedCredential{}, err
	}
	creds.Profiles[rt.ProfileName] = config.Credential{APIKey: apiKey}
	if err := config.SaveCredentials(creds); err != nil {
		return storedCredential{}, err
	}
	return storedCredential{ProfileName: rt.ProfileName, Profile: rt.Profile}, nil
}

func loginAndStoreAPIKey(cmd *cobra.Command, opts *globalOptions, loginOpts loginCredentialOptions) error {
	rt, err := commandRuntime(cmd, opts, false)
	if err != nil {
		return err
	}
	if err := validateAuthorizationOptions(loginOpts); err != nil {
		return err
	}
	if !usesLegacyPasswordLogin(loginOpts) {
		if strings.TrimSpace(loginOpts.RequestID) != "" {
			return resumeAndStoreAPIKeyAuthorization(cmd, opts, rt, loginOpts)
		}
		if loginOpts.ExpiresIn > 0 {
			return fmt.Errorf("--expires-in is only supported with --email/--password-stdin legacy login; choose API key expiry in the browser authorization page")
		}
		return authorizeAndStoreAPIKey(cmd, opts, rt, loginOpts)
	}
	email, password, err := readLoginCredentials(cmd, loginOpts)
	if err != nil {
		return err
	}
	if opts.requestOut != "" {
		planOpts := loginOpts
		planOpts.Email = email
		plan, err := buildLoginSetupPlan(rt, planOpts)
		if err != nil {
			return err
		}
		if err := writeJSONPlan(opts.requestOut, plan); err != nil {
			return err
		}
	}
	loginClient := newFlowClient(rt.Profile, "", cmd.CommandPath(), opts)
	loginResp, err := loginWithPassword(cmd, loginClient, email, password, loginOpts.TurnstileResponse)
	if err != nil {
		return err
	}
	if strings.TrimSpace(loginResp.Tokens.AccessToken) == "" {
		return fmt.Errorf("login response did not include an access token")
	}
	apiKeyClient := newFlowClient(rt.Profile, loginResp.Tokens.AccessToken, cmd.CommandPath(), opts)
	apiKeyResp, err := createCLIAPIKey(cmd, apiKeyClient, loginOpts.KeyName, loginOpts.ExpiresIn)
	if err != nil {
		return err
	}
	if strings.TrimSpace(apiKeyResp.RawKey) == "" {
		return fmt.Errorf("api key creation response did not include raw_key")
	}
	stored, err := storeAPIKeyForRuntime(rt, apiKeyResp.RawKey)
	if err != nil {
		return err
	}
	result := map[string]any{
		"profile":          stored.ProfileName,
		"endpoint":         stored.Profile.Endpoint,
		"workspace_id":     stored.Profile.WorkspaceID,
		"auth_method":      "login_api_key",
		"credential_store": mustCredentialsPath(),
		"config_path":      mustConfigPath(),
		"stored":           true,
	}
	if loginOpts.Check {
		check := attemptStoredCredentialCheck(cmd, opts, stored.Profile, apiKeyResp.RawKey)
		result["check"] = check
		result["verified"] = checkOK(check)
	}
	return writeResult(rt, result, nil)
}

func validateAuthorizationOptions(loginOpts loginCredentialOptions) error {
	if loginOpts.NoWait && strings.TrimSpace(loginOpts.RequestID) != "" {
		return fmt.Errorf("use only one of --no-wait or --request-id")
	}
	if loginOpts.NoWait && usesLegacyPasswordLogin(loginOpts) {
		return fmt.Errorf("--no-wait is only supported with browser authorization; do not combine it with --email, --password-stdin, or --turnstile-response")
	}
	if strings.TrimSpace(loginOpts.RequestID) != "" && usesLegacyPasswordLogin(loginOpts) {
		return fmt.Errorf("--request-id is only supported with browser authorization; do not combine it with --email, --password-stdin, or --turnstile-response")
	}
	if strings.TrimSpace(loginOpts.RequestID) != "" && loginOpts.ExpiresIn > 0 {
		return fmt.Errorf("--expires-in is only supported with --email/--password-stdin legacy login; it cannot be used with --request-id")
	}
	return nil
}

func authorizeAndStoreAPIKey(cmd *cobra.Command, opts *globalOptions, rt *runtimeContext, loginOpts loginCredentialOptions) error {
	deviceUUID, err := ensureRuntimeDeviceUUID(rt)
	if err != nil {
		return err
	}
	deviceName := resolveAuthorizationDeviceName(loginOpts)
	if strings.TrimSpace(deviceName) == "" {
		return fmt.Errorf("device name is required")
	}
	if opts.requestOut != "" {
		plan, err := buildLoginSetupPlan(rt, loginOpts)
		if err != nil {
			return err
		}
		if err := writeJSONPlan(opts.requestOut, plan); err != nil {
			return err
		}
	}
	flowClient := newFlowClient(rt.Profile, "", cmd.CommandPath(), opts)
	createResp, err := createAPIKeyAuthorization(cmd, flowClient, deviceUUID, deviceName)
	if err != nil {
		return err
	}
	if strings.TrimSpace(createResp.AuthorizationID) == "" {
		return fmt.Errorf("authorization response did not include authorization_id")
	}
	if strings.TrimSpace(createResp.AuthorizationURL) == "" {
		return fmt.Errorf("authorization response did not include authorization_url")
	}
	if strings.TrimSpace(createResp.PollToken) == "" {
		return fmt.Errorf("authorization response did not include poll_token")
	}
	if loginOpts.NoWait {
		pending := pendingFromCreateResponse(rt, deviceUUID, deviceName, createResp)
		if err := savePendingAuthorization(pending); err != nil {
			return err
		}
		writeNoWaitAuthorizationInstructions(cmd, createResp.AuthorizationURL, createResp.AuthorizationID)
		return writePendingAuthorizationResult(rt, pending)
	}
	writeAuthorizationInstructions(cmd, createResp.AuthorizationURL, loginOpts.NoBrowser)
	if !loginOpts.NoBrowser {
		if err := openBrowser(cmd, createResp.AuthorizationURL); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Could not open browser automatically: %v\n", err)
			fmt.Fprintf(cmd.ErrOrStderr(), "Open this URL to authorize the CLI:\n%s\n", createResp.AuthorizationURL)
		}
	}
	pollClient := newFlowClient(rt.Profile, createResp.PollToken, cmd.CommandPath(), opts)
	resultResp, err := pollAPIKeyAuthorizationResult(cmd, pollClient, createResp)
	if err != nil {
		return err
	}
	if strings.TrimSpace(resultResp.APIKey) == "" {
		return fmt.Errorf("authorization result did not include api_key")
	}
	stored, err := storeAPIKeyForRuntime(rt, resultResp.APIKey)
	if err != nil {
		return err
	}
	result := map[string]any{
		"profile":          stored.ProfileName,
		"endpoint":         stored.Profile.Endpoint,
		"workspace_id":     stored.Profile.WorkspaceID,
		"auth_method":      "api_key_authorization",
		"credential_store": mustCredentialsPath(),
		"config_path":      mustConfigPath(),
		"authorization": map[string]any{
			"status": resultResp.Status,
		},
		"device": map[string]any{
			"name": deviceName,
		},
		"stored": true,
	}
	if loginOpts.Check {
		check := attemptStoredCredentialCheck(cmd, opts, stored.Profile, resultResp.APIKey)
		result["check"] = check
		result["verified"] = checkOK(check)
	}
	return writeResult(rt, result, nil)
}

func writePendingAuthorizationResult(rt *runtimeContext, pending pendingAuthorization) error {
	result := map[string]any{
		"profile":      rt.ProfileName,
		"endpoint":     rt.Profile.Endpoint,
		"workspace_id": rt.Profile.WorkspaceID,
		"auth_method":  "api_key_authorization",
		"authorization": map[string]any{
			"id":                pending.AuthorizationID,
			"status":            "pending",
			"authorization_url": pending.AuthorizationURL,
			"expires_at":        pending.ExpiresAt,
		},
		"device": map[string]any{
			"name": pending.DeviceName,
		},
		"pending_store":   mustPendingAuthorizationsPath(),
		"stored":          false,
		"resume_command":  "tmc auth login --request-id " + pending.AuthorizationID,
		"setup_resume":    "tmc setup --request-id " + pending.AuthorizationID,
		"check_deferred":  true,
		"open_in_browser": pending.AuthorizationURL,
	}
	return writeResult(rt, result, nil)
}

func resumeAndStoreAPIKeyAuthorization(cmd *cobra.Command, opts *globalOptions, rt *runtimeContext, loginOpts loginCredentialOptions) error {
	requestID := strings.TrimSpace(loginOpts.RequestID)
	if opts.requestOut != "" {
		plan := buildResumeAuthorizationPlan(rt, loginOpts)
		if err := writeJSONPlan(opts.requestOut, plan); err != nil {
			return err
		}
	}
	pending, err := loadPendingAuthorization(requestID)
	if err != nil {
		return err
	}
	if err := validatePendingAuthorizationForRuntime(pending, rt); err != nil {
		return err
	}
	auth := createResponseFromPending(pending)
	pollClient := newFlowClient(rt.Profile, pending.PollToken, cmd.CommandPath(), opts)
	resultResp, err := pollAPIKeyAuthorizationResult(cmd, pollClient, auth)
	if err != nil {
		return err
	}
	if strings.TrimSpace(resultResp.APIKey) == "" {
		return fmt.Errorf("authorization result did not include api_key")
	}
	stored, err := storeAPIKeyForRuntime(rt, resultResp.APIKey)
	if err != nil {
		return err
	}
	if err := removePendingAuthorization(requestID); err != nil {
		return err
	}
	result := map[string]any{
		"profile":          stored.ProfileName,
		"endpoint":         stored.Profile.Endpoint,
		"workspace_id":     stored.Profile.WorkspaceID,
		"auth_method":      "api_key_authorization",
		"credential_store": mustCredentialsPath(),
		"config_path":      mustConfigPath(),
		"authorization": map[string]any{
			"id":     requestID,
			"status": resultResp.Status,
		},
		"stored": true,
	}
	if loginOpts.Check {
		check := attemptStoredCredentialCheck(cmd, opts, stored.Profile, resultResp.APIKey)
		result["check"] = check
		result["verified"] = checkOK(check)
	}
	return writeResult(rt, result, nil)
}

func saveAPIKeyAndMaybeCheck(cmd *cobra.Command, opts *globalOptions, apiKey string, check bool) error {
	return saveAPIKeyAndMaybeCheckForProfile(cmd, opts, apiKey, check, "", "provided")
}

func saveAPIKeyAndMaybeCheckForProfile(cmd *cobra.Command, opts *globalOptions, apiKey string, check bool, profileOverride string, apiKeySource string) error {
	rt, err := commandRuntimeForProfile(cmd, opts, profileOverride, false)
	if err != nil {
		return err
	}
	if opts.requestOut != "" {
		if err := writeJSONPlan(opts.requestOut, buildAPIKeySetupPlan(rt, apiKeySource, check)); err != nil {
			return err
		}
	}
	var checkResult map[string]any
	if check {
		checkResult, err = checkStoredCredential(cmd, opts, rt.Profile, apiKey)
		if err != nil {
			return err
		}
	}
	stored, err := storeAPIKeyForRuntime(rt, apiKey)
	if err != nil {
		return err
	}
	result := map[string]any{
		"profile":          stored.ProfileName,
		"endpoint":         stored.Profile.Endpoint,
		"workspace_id":     stored.Profile.WorkspaceID,
		"auth_method":      "api_key",
		"credential_store": mustCredentialsPath(),
		"config_path":      mustConfigPath(),
		"stored":           true,
	}
	if check {
		result["check"] = checkResult
		result["verified"] = true
	}
	return writeResult(rt, result, nil)
}

func readLoginCredentials(cmd *cobra.Command, opts loginCredentialOptions) (string, string, error) {
	email := strings.TrimSpace(opts.Email)
	if email == "" {
		if !canPrompt(cmd) {
			return "", "", fmt.Errorf("email is required; pass --email or run tmc setup in an interactive terminal")
		}
		value, err := promptLine(cmd, "Email: ")
		if err != nil {
			return "", "", err
		}
		email = strings.TrimSpace(value)
	}
	if email == "" {
		return "", "", fmt.Errorf("email is required")
	}

	password, err := readPassword(cmd, opts.PasswordStdin)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(password) == "" {
		return "", "", fmt.Errorf("password is required")
	}
	return email, strings.TrimSpace(password), nil
}

func readPassword(cmd *cobra.Command, fromStdin bool) (string, error) {
	if fromStdin {
		raw, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
	file, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", fmt.Errorf("password is required; pass --password-stdin or run in an interactive terminal")
	}
	fmt.Fprint(cmd.ErrOrStderr(), "Password: ")
	raw, err := term.ReadPassword(int(file.Fd()))
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func canPrompt(cmd *cobra.Command) bool {
	file, ok := cmd.InOrStdin().(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func promptLine(cmd *cobra.Command, prompt string) (string, error) {
	fmt.Fprint(cmd.ErrOrStderr(), prompt)
	value, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return value, nil
}

func loginWithPassword(cmd *cobra.Command, c *client.Client, email string, password string, turnstileResponse string) (openapi.LoginResponse, error) {
	body := openapi.LoginRequest{
		Email:               email,
		Password:            password,
		CfTurnstileResponse: strings.TrimSpace(turnstileResponse),
	}
	resp, err := c.Do(cmd.Context(), "POST", "/auth/login", nil, body)
	if err != nil {
		return openapi.LoginResponse{}, err
	}
	var out openapi.LoginResponse
	if err := resp.DecodeData(&out); err != nil {
		return openapi.LoginResponse{}, fmt.Errorf("decode login response: %w", err)
	}
	return out, nil
}

func createCLIAPIKey(cmd *cobra.Command, c *client.Client, keyName string, expiresIn int64) (openapi.APIKeyCreateResponse, error) {
	keyName = strings.TrimSpace(keyName)
	if keyName == "" {
		keyName = defaultCLIAPIKeyName()
	}
	resp, err := c.Do(cmd.Context(), "POST", "/auth/api-keys", nil, openapi.APIKeyCreateRequest{
		Name:      keyName,
		ExpiresIn: expiresIn,
	})
	if err != nil {
		return openapi.APIKeyCreateResponse{}, err
	}
	var out openapi.APIKeyCreateResponse
	if err := resp.DecodeData(&out); err != nil {
		return openapi.APIKeyCreateResponse{}, fmt.Errorf("decode api key response: %w", err)
	}
	return out, nil
}

func createAPIKeyAuthorization(cmd *cobra.Command, c *client.Client, deviceUUID string, deviceName string) (openapi.APIKeyAuthorizationCreateResponse, error) {
	resp, err := c.Do(cmd.Context(), "POST", "/auth/api-key-authorizations", nil, openapi.APIKeyAuthorizationCreateRequest{
		DeviceUUID: strings.TrimSpace(deviceUUID),
		DeviceName: strings.TrimSpace(deviceName),
	})
	if err != nil {
		return openapi.APIKeyAuthorizationCreateResponse{}, err
	}
	var out openapi.APIKeyAuthorizationCreateResponse
	if err := resp.DecodeData(&out); err != nil {
		return openapi.APIKeyAuthorizationCreateResponse{}, fmt.Errorf("decode api key authorization response: %w", err)
	}
	return out, nil
}

func pollAPIKeyAuthorizationResult(cmd *cobra.Command, c *client.Client, auth openapi.APIKeyAuthorizationCreateResponse) (openapi.APIKeyAuthorizationResultResponse, error) {
	interval := time.Duration(auth.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	expiresIn := time.Duration(auth.AuthorizationExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = 10 * time.Minute
	}
	deadline := time.Now().Add(expiresIn)
	path := "/auth/api-key-authorizations/" + url.PathEscape(auth.AuthorizationID) + "/result"
	for {
		resp, err := c.Do(cmd.Context(), "GET", path, nil, nil)
		if err != nil {
			return openapi.APIKeyAuthorizationResultResponse{}, err
		}
		var out openapi.APIKeyAuthorizationResultResponse
		if err := resp.DecodeData(&out); err != nil {
			return openapi.APIKeyAuthorizationResultResponse{}, fmt.Errorf("decode api key authorization result: %w", err)
		}
		switch strings.TrimSpace(out.Status) {
		case "approved":
			if strings.TrimSpace(out.APIKey) == "" {
				return openapi.APIKeyAuthorizationResultResponse{}, fmt.Errorf("authorization was approved but api_key was not returned")
			}
			return out, nil
		case "pending":
		case "denied":
			return openapi.APIKeyAuthorizationResultResponse{}, fmt.Errorf("authorization was denied")
		case "expired":
			return openapi.APIKeyAuthorizationResultResponse{}, fmt.Errorf("authorization expired")
		case "consumed":
			return openapi.APIKeyAuthorizationResultResponse{}, fmt.Errorf("authorization result was already consumed")
		default:
			if strings.TrimSpace(out.Status) == "" {
				return openapi.APIKeyAuthorizationResultResponse{}, fmt.Errorf("authorization result did not include status")
			}
			return openapi.APIKeyAuthorizationResultResponse{}, fmt.Errorf("unsupported authorization status %q", out.Status)
		}
		wait := interval
		if remaining := time.Until(deadline); remaining <= 0 {
			return openapi.APIKeyAuthorizationResultResponse{}, fmt.Errorf("authorization expired")
		} else if wait > remaining {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		select {
		case <-cmd.Context().Done():
			timer.Stop()
			return openapi.APIKeyAuthorizationResultResponse{}, cmd.Context().Err()
		case <-timer.C:
		}
	}
}

func writeAuthorizationInstructions(cmd *cobra.Command, authorizationURL string, noBrowser bool) {
	if noBrowser {
		fmt.Fprintf(cmd.ErrOrStderr(), "Open this URL to authorize the CLI:\n%s\n", authorizationURL)
	} else {
		fmt.Fprintln(cmd.ErrOrStderr(), "Opening browser for TMCopilot CLI authorization.")
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Waiting for authorization...")
}

func writeNoWaitAuthorizationInstructions(cmd *cobra.Command, authorizationURL string, requestID string) {
	fmt.Fprintf(cmd.ErrOrStderr(), "Open this URL to authorize the CLI:\n%s\n", authorizationURL)
	fmt.Fprintf(cmd.ErrOrStderr(), "After approval, resume with:\n%s\n", noWaitResumeCommand(cmd, requestID))
}

func noWaitResumeCommand(cmd *cobra.Command, requestID string) string {
	if cmd != nil && strings.HasSuffix(cmd.CommandPath(), " setup") {
		return "tmc setup --request-id " + requestID
	}
	return "tmc auth login --request-id " + requestID
}

func openBrowser(cmd *cobra.Command, rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("authorization URL is empty")
	}
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name = "open"
		args = []string{rawURL}
	case "windows":
		name = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", rawURL}
	default:
		name = "xdg-open"
		args = []string{rawURL}
	}
	command := exec.CommandContext(cmd.Context(), name, args...)
	if err := command.Start(); err != nil {
		return err
	}
	go func() {
		_ = command.Wait()
	}()
	return nil
}

func ensureRuntimeDeviceUUID(rt *runtimeContext) (string, error) {
	if rt == nil {
		return "", fmt.Errorf("runtime context is required")
	}
	deviceUUID := strings.TrimSpace(rt.Profile.DeviceUUID)
	if deviceUUID != "" {
		return deviceUUID, nil
	}
	generated, err := randomUUID()
	if err != nil {
		return "", err
	}
	profile := rt.Profile
	profile.DeviceUUID = generated
	rt.Profile = profile
	rt.Config.Profiles[rt.ProfileName] = profile
	if err := config.Save(rt.Config); err != nil {
		return "", err
	}
	return generated, nil
}

func randomUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	hexed := make([]byte, 32)
	hex.Encode(hexed, b[:])
	s := string(hexed)
	return fmt.Sprintf("%s-%s-%s-%s-%s", s[0:8], s[8:12], s[12:16], s[16:20], s[20:32]), nil
}

func checkStoredCredential(cmd *cobra.Command, opts *globalOptions, profile config.Profile, apiKey string) (map[string]any, error) {
	c := newFlowClient(profile, apiKey, cmd.CommandPath(), opts)
	resp, err := c.Do(cmd.Context(), "GET", "/auth/me", nil, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":          true,
		"status_code": resp.StatusCode,
	}, nil
}

func attemptStoredCredentialCheck(cmd *cobra.Command, opts *globalOptions, profile config.Profile, apiKey string) map[string]any {
	check, err := checkStoredCredential(cmd, opts, profile, apiKey)
	if err == nil {
		return check
	}
	return map[string]any{
		"ok":      false,
		"message": err.Error(),
	}
}

func checkOK(check map[string]any) bool {
	if check == nil {
		return false
	}
	ok, _ := check["ok"].(bool)
	return ok
}

func newFlowClient(profile config.Profile, credential string, commandPath string, opts *globalOptions) *client.Client {
	c := client.New(
		profile.Endpoint,
		credential,
		profile.WorkspaceID,
		"tmcopilot-cli/"+version.Version,
		opts.timeout,
	)
	c.ExtraHeaders = map[string]string{
		"X-TMCopilot-CLI-Command": commandPath,
	}
	if opts != nil && strings.TrimSpace(opts.idempotencyKey) != "" {
		c.ExtraHeaders["Idempotency-Key"] = strings.TrimSpace(opts.idempotencyKey)
	}
	return c
}

func commandRuntimeForProfile(cmd *cobra.Command, opts *globalOptions, profileOverride string, needAuth bool) (*runtimeContext, error) {
	if strings.TrimSpace(profileOverride) == "" {
		return commandRuntime(cmd, opts, needAuth)
	}
	optsCopy := *opts
	optsCopy.profile = strings.TrimSpace(profileOverride)
	return commandRuntime(cmd, &optsCopy, needAuth)
}

func defaultCLIAPIKeyName() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return "tmc cli"
	}
	return "tmc cli " + strings.TrimSpace(hostname)
}
