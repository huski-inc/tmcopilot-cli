package openapi

import (
	"encoding/json"
	"strings"
)

type Endpoint struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	OperationID string   `json:"operation_id,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	HasBody     bool     `json:"has_body"`
	ParamCount  int      `json:"param_count"`
	Coverage    string   `json:"coverage"`
}

type EndpointSchema struct {
	Endpoint    Endpoint                   `json:"endpoint"`
	Parameters  json.RawMessage            `json:"parameters,omitempty"`
	Responses   json.RawMessage            `json:"responses,omitempty"`
	Definitions map[string]json.RawMessage `json:"definitions,omitempty"`
}

func EndpointKey(method string, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}

func FindEndpoint(method string, path string) (Endpoint, bool) {
	key := EndpointKey(method, path)
	for _, endpoint := range Endpoints {
		if EndpointKey(endpoint.Method, endpoint.Path) == key {
			return endpoint, true
		}
	}
	return Endpoint{}, false
}

func FindEndpointSchema(method string, path string) (EndpointSchema, bool) {
	key := EndpointKey(method, path)
	schema, ok := EndpointSchemas[key]
	return schema, ok
}

func FilterEndpoints(tag string, coverage string) []Endpoint {
	tag = strings.TrimSpace(strings.ToLower(tag))
	coverage = strings.TrimSpace(strings.ToLower(coverage))
	out := make([]Endpoint, 0, len(Endpoints))
	for _, endpoint := range Endpoints {
		if coverage != "" && strings.ToLower(endpoint.Coverage) != coverage {
			continue
		}
		if tag != "" && !endpointHasTag(endpoint, tag) {
			continue
		}
		out = append(out, endpoint)
	}
	return out
}

func endpointHasTag(endpoint Endpoint, tag string) bool {
	for _, value := range endpoint.Tags {
		if strings.ToLower(value) == tag {
			return true
		}
	}
	return false
}
