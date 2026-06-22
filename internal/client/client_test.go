package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientDoUnwrapsEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tmc_test" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.URL.Path; got != "/api/v1/auth/me" {
			t.Fatalf("path = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": map[string]any{"title": "Success", "text": "ok"},
			"data":    map[string]any{"email": "user@example.com"},
		})
	}))
	defer server.Close()

	c := New(server.URL, "tmc_test", "", "test", time.Second)
	resp, err := c.Do(context.Background(), "GET", "/auth/me", nil, nil)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	var data map[string]string
	if err := resp.DecodeData(&data); err != nil {
		t.Fatalf("DecodeData() error = %v", err)
	}
	if data["email"] != "user@example.com" {
		t.Fatalf("email = %q", data["email"])
	}
}

func TestClientDoDetectsEnvelopeWithStringMessageAndNullData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": "ok",
			"data":    nil,
		})
	}))
	defer server.Close()

	c := New(server.URL, "", "", "test", time.Second)
	resp, err := c.Do(context.Background(), "GET", "/version", nil, nil)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.Envelope == nil {
		t.Fatal("Envelope = nil")
	}
	if resp.Envelope.Message.Text != "ok" {
		t.Fatalf("message text = %q", resp.Envelope.Message.Text)
	}
	if string(resp.Envelope.Data) != "null" {
		t.Fatalf("data = %q", resp.Envelope.Data)
	}
}

func TestClientDoReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Trace-ID", "trace-1")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    40100,
			"message": map[string]any{"title": "Unauthorized", "text": "missing authentication"},
		})
	}))
	defer server.Close()

	c := New(server.URL, "", "", "test", time.Second)
	_, err := c.Do(context.Background(), "GET", "/auth/me", nil, nil)
	if err == nil {
		t.Fatal("Do() error = nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized || apiErr.Code != 40100 || apiErr.TraceID != "trace-1" {
		t.Fatalf("apiErr = %+v", apiErr)
	}
}
