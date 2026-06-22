package openapi

import "encoding/json"

type LoginRequest struct {
	Email               string `json:"email"`
	Password            string `json:"password"`
	CfTurnstileResponse string `json:"cf_turnstile_response,omitempty"`
}

type LoginResponse struct {
	Tokens TokenPair       `json:"tokens"`
	User   json.RawMessage `json:"user,omitempty"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
}

type APIKeyCreateRequest struct {
	Name      string `json:"name"`
	ExpiresIn int64  `json:"expires_in,omitempty"`
}

type APIKeyCreateResponse struct {
	RawKey string     `json:"raw_key"`
	Key    APIKeyInfo `json:"key"`
}

type APIKeyInfo struct {
	ID              string `json:"id,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	Name            string `json:"name,omitempty"`
	BoundDeviceUUID string `json:"bound_device_uuid,omitempty"`
	BoundDeviceName string `json:"bound_device_name,omitempty"`
	KeyPrefix       string `json:"key_prefix,omitempty"`
	LastUsedAt      *int64 `json:"last_used_at,omitempty"`
	ExpiresAt       *int64 `json:"expires_at,omitempty"`
	CreatedAt       int64  `json:"created_at,omitempty"`
}

type APIKeyAuthorizationCreateRequest struct {
	DeviceUUID string `json:"device_uuid"`
	DeviceName string `json:"device_name"`
}

type APIKeyAuthorizationCreateResponse struct {
	AuthorizationID        string `json:"authorization_id"`
	AuthorizationURL       string `json:"authorization_url"`
	PollToken              string `json:"poll_token"`
	AuthorizationExpiresIn int64  `json:"authorization_expires_in"`
	Interval               int64  `json:"interval"`
}

type APIKeyAuthorizationResultResponse struct {
	Status string     `json:"status"`
	APIKey string     `json:"api_key,omitempty"`
	Key    APIKeyInfo `json:"key,omitempty"`
}
