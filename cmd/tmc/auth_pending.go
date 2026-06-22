package tmc

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huski-inc/tmcopilot-cli/internal/config"
	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

type pendingAuthorizationStore struct {
	Authorizations map[string]pendingAuthorization `json:"authorizations"`
}

type pendingAuthorization struct {
	AuthorizationID  string `json:"authorization_id"`
	AuthorizationURL string `json:"authorization_url,omitempty"`
	PollToken        string `json:"poll_token"`
	ProfileName      string `json:"profile"`
	Endpoint         string `json:"endpoint"`
	WorkspaceID      string `json:"workspace_id,omitempty"`
	DeviceUUID       string `json:"device_uuid,omitempty"`
	DeviceName       string `json:"device_name,omitempty"`
	CreatedAt        string `json:"created_at"`
	ExpiresAt        string `json:"expires_at,omitempty"`
	Interval         int64  `json:"interval,omitempty"`
}

func pendingAuthorizationsPath() (string, error) {
	home, err := config.HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "pending-authorizations.json"), nil
}

func mustPendingAuthorizationsPath() string {
	path, err := pendingAuthorizationsPath()
	if err != nil {
		return ""
	}
	return path
}

func loadPendingAuthorizationStore() (pendingAuthorizationStore, error) {
	path, err := pendingAuthorizationsPath()
	if err != nil {
		return pendingAuthorizationStore{}, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return pendingAuthorizationStore{Authorizations: map[string]pendingAuthorization{}}, nil
	}
	if err != nil {
		return pendingAuthorizationStore{}, err
	}
	var store pendingAuthorizationStore
	if err := json.Unmarshal(raw, &store); err != nil {
		return pendingAuthorizationStore{}, fmt.Errorf("parse pending authorizations %s: %w", path, err)
	}
	if store.Authorizations == nil {
		store.Authorizations = map[string]pendingAuthorization{}
	}
	return store, nil
}

func savePendingAuthorizationStore(store pendingAuthorizationStore) error {
	if store.Authorizations == nil {
		store.Authorizations = map[string]pendingAuthorization{}
	}
	path, err := pendingAuthorizationsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o600)
}

func savePendingAuthorization(auth pendingAuthorization) error {
	auth.AuthorizationID = strings.TrimSpace(auth.AuthorizationID)
	auth.PollToken = strings.TrimSpace(auth.PollToken)
	if auth.AuthorizationID == "" {
		return fmt.Errorf("authorization id is required")
	}
	if auth.PollToken == "" {
		return fmt.Errorf("poll token is required")
	}
	store, err := loadPendingAuthorizationStore()
	if err != nil {
		return err
	}
	store.Authorizations[auth.AuthorizationID] = auth
	return savePendingAuthorizationStore(store)
}

func loadPendingAuthorization(requestID string) (pendingAuthorization, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return pendingAuthorization{}, fmt.Errorf("request id is required")
	}
	store, err := loadPendingAuthorizationStore()
	if err != nil {
		return pendingAuthorization{}, err
	}
	auth, ok := store.Authorizations[requestID]
	if !ok {
		return pendingAuthorization{}, fmt.Errorf("pending authorization %q not found; run `tmc setup --no-wait` first", requestID)
	}
	if strings.TrimSpace(auth.PollToken) == "" {
		return pendingAuthorization{}, fmt.Errorf("pending authorization %q is missing local poll token; restart with `tmc setup --no-wait`", requestID)
	}
	if expiresAt := strings.TrimSpace(auth.ExpiresAt); expiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			return pendingAuthorization{}, fmt.Errorf("pending authorization %q has invalid expires_at %q; restart with `tmc setup --no-wait`", requestID, expiresAt)
		}
		if !time.Now().Before(parsed) {
			return pendingAuthorization{}, fmt.Errorf("pending authorization %q expired at %s; restart with `tmc setup --no-wait`", requestID, expiresAt)
		}
	}
	return auth, nil
}

func removePendingAuthorization(requestID string) error {
	store, err := loadPendingAuthorizationStore()
	if err != nil {
		return err
	}
	delete(store.Authorizations, strings.TrimSpace(requestID))
	return savePendingAuthorizationStore(store)
}

func pendingFromCreateResponse(rt *runtimeContext, deviceUUID string, deviceName string, resp openapi.APIKeyAuthorizationCreateResponse) pendingAuthorization {
	now := time.Now().UTC()
	expiresAt := ""
	if resp.AuthorizationExpiresIn > 0 {
		expiresAt = now.Add(time.Duration(resp.AuthorizationExpiresIn) * time.Second).Format(time.RFC3339)
	}
	return pendingAuthorization{
		AuthorizationID:  strings.TrimSpace(resp.AuthorizationID),
		AuthorizationURL: strings.TrimSpace(resp.AuthorizationURL),
		PollToken:        strings.TrimSpace(resp.PollToken),
		ProfileName:      rt.ProfileName,
		Endpoint:         rt.Profile.Endpoint,
		WorkspaceID:      rt.Profile.WorkspaceID,
		DeviceUUID:       deviceUUID,
		DeviceName:       deviceName,
		CreatedAt:        now.Format(time.RFC3339),
		ExpiresAt:        expiresAt,
		Interval:         resp.Interval,
	}
}

func createResponseFromPending(auth pendingAuthorization) openapi.APIKeyAuthorizationCreateResponse {
	expiresIn := int64(0)
	if strings.TrimSpace(auth.ExpiresAt) != "" {
		if expiresAt, err := time.Parse(time.RFC3339, auth.ExpiresAt); err == nil {
			remaining := time.Until(expiresAt)
			if remaining > 0 {
				expiresIn = int64(remaining.Seconds())
				if expiresIn <= 0 {
					expiresIn = 1
				}
			}
		}
	}
	return openapi.APIKeyAuthorizationCreateResponse{
		AuthorizationID:        auth.AuthorizationID,
		AuthorizationURL:       auth.AuthorizationURL,
		PollToken:              auth.PollToken,
		AuthorizationExpiresIn: expiresIn,
		Interval:               auth.Interval,
	}
}

func validatePendingAuthorizationForRuntime(auth pendingAuthorization, rt *runtimeContext) error {
	if strings.TrimSpace(auth.ProfileName) != "" && auth.ProfileName != rt.ProfileName {
		return fmt.Errorf("pending authorization %q belongs to profile %q; rerun with `--profile %s`", auth.AuthorizationID, auth.ProfileName, auth.ProfileName)
	}
	if endpoint := config.NormalizeEndpoint(auth.Endpoint); endpoint != "" && endpoint != rt.Profile.Endpoint {
		return fmt.Errorf("pending authorization %q was created for endpoint %q; rerun with `--endpoint %s`", auth.AuthorizationID, endpoint, endpoint)
	}
	return nil
}
