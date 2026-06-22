package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	Endpoint     string
	APIKey       string
	WorkspaceID  string
	HTTPClient   *http.Client
	UserAgent    string
	ExtraHeaders map[string]string
}

type ResponseEnvelope struct {
	Code    int             `json:"code"`
	Message Message         `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type Message struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func (m *Message) UnmarshalJSON(raw []byte) error {
	var object struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	}
	if err := json.Unmarshal(raw, &object); err == nil {
		m.Title = object.Title
		m.Text = object.Text
		return nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return err
	}
	m.Text = text
	return nil
}

type APIResponse struct {
	StatusCode int
	Headers    http.Header
	Envelope   *ResponseEnvelope
	Raw        json.RawMessage
}

type APIError struct {
	StatusCode int
	Code       int
	Title      string
	Text       string
	TraceID    string
	Raw        string
}

func (e *APIError) Error() string {
	parts := make([]string, 0, 3)
	if e.StatusCode > 0 {
		parts = append(parts, fmt.Sprintf("http %d", e.StatusCode))
	}
	if e.Code > 0 {
		parts = append(parts, fmt.Sprintf("code %d", e.Code))
	}
	if e.Text != "" {
		parts = append(parts, e.Text)
	} else if e.Title != "" {
		parts = append(parts, e.Title)
	} else if e.Raw != "" {
		parts = append(parts, e.Raw)
	}
	if len(parts) == 0 {
		return "api request failed"
	}
	return strings.Join(parts, ": ")
}

func New(endpoint, apiKey, workspaceID, userAgent string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		Endpoint:    strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		APIKey:      strings.TrimSpace(apiKey),
		WorkspaceID: strings.TrimSpace(workspaceID),
		HTTPClient:  &http.Client{Timeout: timeout},
		UserAgent:   strings.TrimSpace(userAgent),
	}
}

func (c *Client) Do(ctx context.Context, method, apiPath string, query url.Values, body any) (*APIResponse, error) {
	if c.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	reqURL, err := c.buildURL(apiPath, query)
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), reqURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
		req.Header.Set("X-TMCopilot-CLI-Version", c.UserAgent)
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	if c.WorkspaceID != "" {
		req.Header.Set("X-TMCopilot-Workspace-ID", c.WorkspaceID)
	}
	for key, value := range c.ExtraHeaders {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	apiResp := &APIResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Raw:        append(json.RawMessage(nil), raw...),
	}
	if envelope, ok := decodeEnvelope(raw); ok {
		apiResp.Envelope = envelope
	}
	if resp.StatusCode >= 400 || (apiResp.Envelope != nil && apiResp.Envelope.Code != 0) {
		return apiResp, apiResp.errorFromResponse()
	}
	return apiResp, nil
}

func decodeEnvelope(raw []byte) (*ResponseEnvelope, bool) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, false
	}
	_, hasCode := fields["code"]
	_, hasMessage := fields["message"]
	_, hasData := fields["data"]
	if !hasCode && !hasMessage && !hasData {
		return nil, false
	}
	var envelope ResponseEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, false
	}
	return &envelope, true
}

func (r *APIResponse) DataOrRaw(rawEnvelope bool) any {
	if rawEnvelope {
		if r.Envelope != nil {
			return r.Envelope
		}
		return json.RawMessage(r.Raw)
	}
	if r.Envelope != nil && r.Envelope.Data != nil {
		return json.RawMessage(r.Envelope.Data)
	}
	if len(r.Raw) == 0 {
		return map[string]any{}
	}
	return json.RawMessage(r.Raw)
}

func (r *APIResponse) DecodeData(out any) error {
	if r.Envelope == nil {
		return json.Unmarshal(r.Raw, out)
	}
	return json.Unmarshal(r.Envelope.Data, out)
}

func (r *APIResponse) errorFromResponse() error {
	traceID := r.Headers.Get("X-Trace-ID")
	if r.Envelope != nil {
		return &APIError{
			StatusCode: r.StatusCode,
			Code:       r.Envelope.Code,
			Title:      r.Envelope.Message.Title,
			Text:       r.Envelope.Message.Text,
			TraceID:    traceID,
			Raw:        string(r.Raw),
		}
	}
	return &APIError{
		StatusCode: r.StatusCode,
		TraceID:    traceID,
		Raw:        string(r.Raw),
	}
}

func (c *Client) buildURL(apiPath string, query url.Values) (string, error) {
	apiPath = strings.TrimSpace(apiPath)
	if apiPath == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.HasPrefix(apiPath, "http://") || strings.HasPrefix(apiPath, "https://") {
		u, err := url.Parse(apiPath)
		if err != nil {
			return "", err
		}
		if len(query) > 0 {
			q := u.Query()
			for key, values := range query {
				for _, value := range values {
					q.Add(key, value)
				}
			}
			u.RawQuery = q.Encode()
		}
		return u.String(), nil
	}
	if !strings.HasPrefix(apiPath, "/") {
		apiPath = "/" + apiPath
	}
	if !strings.HasPrefix(apiPath, "/api/") {
		apiPath = "/api/v1" + apiPath
	}
	u, err := url.Parse(c.Endpoint + apiPath)
	if err != nil {
		return "", err
	}
	if len(query) > 0 {
		q := u.Query()
		for key, values := range query {
			for _, value := range values {
				q.Add(key, value)
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}
