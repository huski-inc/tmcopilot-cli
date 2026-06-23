package openapi

import (
	"bytes"
	"testing"
)

func TestGeneratedCatalogContainsTypedEndpoints(t *testing.T) {
	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/auth/me"},
		{"POST", "/trademark/search"},
		{"GET", "/portfolio/trademarks/search"},
		{"GET", "/competitors"},
		{"POST", "/gap-analyses"},
		{"POST", "/trademark/wide-table/lawsuits"},
		{"GET", "/trademark/wide-table/lawsuits/{caseNumber}"},
		{"GET", "/trademark/wide-table/lawyers/{graphId}"},
		{"POST", "/trademark/wide-table/lawyers/{graphId}/trademarks"},
	}
	for _, tt := range tests {
		endpoint, ok := FindEndpoint(tt.method, tt.path)
		if !ok {
			t.Fatalf("endpoint missing: %s %s", tt.method, tt.path)
		}
		if endpoint.Coverage != "typed" {
			t.Fatalf("coverage for %s %s = %q", tt.method, tt.path, endpoint.Coverage)
		}
	}
}

func TestAuthDTOBackedEndpointsHaveSwaggerSchemas(t *testing.T) {
	tests := []struct {
		method      string
		path        string
		definitions []string
	}{
		{
			method:      "POST",
			path:        "/auth/login",
			definitions: []string{"internal_protocol_rest_handler.loginRequest", "internal_protocol_rest_handler.loginResponse", "tmcopilot-api_internal_domain_auth.TokenPair"},
		},
		{
			method:      "POST",
			path:        "/auth/api-keys",
			definitions: []string{"internal_protocol_rest_handler.createAPIKeyRequest", "internal_protocol_rest_handler.createAPIKeyResponse", "tmcopilot-api_internal_usecase_api_dto.APIKeyResponse"},
		},
	}
	for _, tt := range tests {
		schema, ok := FindEndpointSchema(tt.method, tt.path)
		if !ok {
			t.Fatalf("schema missing: %s %s", tt.method, tt.path)
		}
		for _, definition := range tt.definitions {
			if _, ok := schema.Definitions[definition]; !ok {
				t.Fatalf("definition %q missing for %s %s", definition, tt.method, tt.path)
			}
		}
		if !bytes.Contains(schema.Parameters, []byte(`"body"`)) {
			t.Fatalf("body parameter missing for %s %s: %s", tt.method, tt.path, string(schema.Parameters))
		}
	}
}
