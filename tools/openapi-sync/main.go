package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type swaggerSpec struct {
	Swagger     string                          `json:"swagger"`
	Info        map[string]any                  `json:"info"`
	Paths       map[string]map[string]operation `json:"paths"`
	Definitions map[string]json.RawMessage      `json:"definitions"`
}

type operation struct {
	OperationID string                     `json:"operationId"`
	Summary     string                     `json:"summary"`
	Tags        []string                   `json:"tags"`
	Parameters  []parameterSpec            `json:"parameters"`
	Responses   map[string]json.RawMessage `json:"responses"`
}

type parameterSpec struct {
	Name        string          `json:"name"`
	In          string          `json:"in"`
	Type        string          `json:"type,omitempty"`
	Description string          `json:"description,omitempty"`
	Required    bool            `json:"required,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
}

type endpoint struct {
	Method      string
	Path        string
	OperationID string
	Summary     string
	Tags        []string
	HasBody     bool
	ParamCount  int
	Coverage    string
	Parameters  json.RawMessage
	Responses   json.RawMessage
	Definitions map[string]json.RawMessage
}

func main() {
	var specPath string
	var outGo string
	var outInventory string
	var check bool
	flag.StringVar(&specPath, "spec", defaultSpecPath(), "swagger.json path")
	flag.StringVar(&outGo, "out", "internal/openapi/catalog_generated.go", "generated Go catalog path")
	flag.StringVar(&outInventory, "inventory", "plans/openapi-endpoint-inventory.md", "endpoint inventory markdown path")
	flag.BoolVar(&check, "check", false, "check generated files without writing")
	flag.Parse()

	raw, err := os.ReadFile(specPath)
	must(err)
	var spec swaggerSpec
	must(json.Unmarshal(raw, &spec))

	hashRaw := sha256.Sum256(raw)
	hash := hex.EncodeToString(hashRaw[:])
	endpoints := collectEndpoints(spec)

	goFile := renderGo(hash, specPath, endpoints)
	inventory := renderInventory(hash, specPath, endpoints)

	if check {
		checkFile(outGo, goFile)
		checkFile(outInventory, inventory)
		return
	}
	must(writeFile(outGo, goFile))
	must(writeFile(outInventory, inventory))
}

func defaultSpecPath() string {
	return filepath.Join("..", "tmcopilot-project", "backend", "docs", "swagger", "swagger.json")
}

func collectEndpoints(spec swaggerSpec) []endpoint {
	endpoints := make([]endpoint, 0, len(spec.Paths))
	for path, methods := range spec.Paths {
		for method, op := range methods {
			method = strings.ToUpper(method)
			if !isHTTPMethod(method) {
				continue
			}
			item := endpoint{
				Method:      method,
				Path:        path,
				OperationID: strings.TrimSpace(op.OperationID),
				Summary:     strings.TrimSpace(op.Summary),
				Tags:        append([]string{}, op.Tags...),
				ParamCount:  len(op.Parameters),
				Coverage:    endpointCoverage(method, path),
			}
			item.Parameters = mustMarshal(op.Parameters)
			item.Responses = mustMarshal(op.Responses)
			for _, parameter := range op.Parameters {
				if parameter.In == "body" {
					item.HasBody = true
					break
				}
			}
			item.Definitions = referencedDefinitions(spec.Definitions, item.Parameters, item.Responses)
			endpoints = append(endpoints, item)
		}
	}
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path == endpoints[j].Path {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})
	return endpoints
}

func isHTTPMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func endpointCoverage(method string, path string) string {
	key := method + " " + path
	if typedEndpoints[key] {
		return "typed"
	}
	if rawReadyEndpoints[key] {
		return "raw-ready"
	}
	return "raw"
}

var typedEndpoints = map[string]bool{
	"GET /auth/api-keys":                                       true,
	"POST /auth/api-keys":                                      true,
	"DELETE /auth/api-keys/{id}":                               true,
	"GET /auth/collaborators":                                  true,
	"POST /auth/collaborators/invitations":                     true,
	"DELETE /auth/collaborators/invitations/{id}":              true,
	"POST /auth/collaborators/invitations/{token}/accept":      true,
	"DELETE /auth/collaborators/{id}":                          true,
	"PUT /auth/collaborators/{id}/role":                        true,
	"GET /auth/me":                                             true,
	"POST /auth/logout":                                        true,
	"GET /auth/notification-preferences":                       true,
	"PUT /auth/notification-preferences":                       true,
	"GET /auth/ui-settings":                                    true,
	"GET /auth/workspaces":                                     true,
	"POST /common-law/max-similarity":                          true,
	"POST /common-law/search/app-store":                        true,
	"POST /common-law/search/ecommerce/handle":                 true,
	"POST /common-law/search/google/text":                      true,
	"POST /common-law/search/social/handle":                    true,
	"POST /common-law/search/social/text":                      true,
	"GET /competitors":                                         true,
	"GET /competitors/activities":                              true,
	"GET /competitors/reports":                                 true,
	"POST /domain/max-similarity":                              true,
	"POST /domain/search":                                      true,
	"GET /files":                                               true,
	"POST /files/presign":                                      true,
	"GET /gap-analyses":                                        true,
	"POST /gap-analyses":                                       true,
	"GET /gap-analyses/shares/{token}":                         true,
	"DELETE /gap-analyses/{id}":                                true,
	"GET /gap-analyses/{id}":                                   true,
	"GET /gap-analyses/{id}/reports":                           true,
	"POST /gap-analyses/{id}/reports/generate":                 true,
	"GET /gap-analyses/{id}/results":                           true,
	"POST /gap-analyses/{id}/run":                              true,
	"POST /gap-analyses/{id}/share":                            true,
	"GET /gap-analyses/{id}/shares":                            true,
	"DELETE /gap-analyses/{id}/shares/{token}":                 true,
	"GET /portfolio/actions/cbp":                               true,
	"GET /portfolio/actions/cbp/summary":                       true,
	"GET /portfolio/actions/conflict":                          true,
	"GET /portfolio/actions/conflict/summary":                  true,
	"GET /portfolio/actions/office":                            true,
	"GET /portfolio/actions/office/summary":                    true,
	"GET /portfolio/activity":                                  true,
	"GET /portfolio/tasks":                                     true,
	"GET /portfolio/tasks/latest-sync":                         true,
	"GET /portfolio/tasks/stats":                               true,
	"GET /portfolio/tasks/{taskId}":                            true,
	"GET /portfolio/trademark-groups":                          true,
	"PUT /portfolio/trademark-groups/{groupId}/monitor/toggle": true,
	"PUT /portfolio/trademark-monitor":                         true,
	"PUT /portfolio/trademark-monitor/toggle":                  true,
	"GET /portfolio/trademarks/counts":                         true,
	"POST /portfolio/trademarks/import":                        true,
	"POST /portfolio/trademarks/import/preview":                true,
	"GET /portfolio/trademarks/monitored":                      true,
	"GET /portfolio/trademarks/search":                         true,
	"GET /portfolio/trademarks/{trademarkId}":                  true,
	"PUT /portfolio/trademarks/{trademarkId}":                  true,
	"GET /portfolio/trademarks/{trademarkId}/metadata":         true,
	"PUT /portfolio/trademarks/{trademarkId}/metadata":         true,
	"PUT /portfolio/trademarks/{trademarkId}/monitor":          true,
	"POST /trademark/detail":                                   true,
	"POST /trademark/image/task":                               true,
	"POST /trademark/image/task/result":                        true,
	"GET /trademark/image/task/{id}/result":                    true,
	"GET /trademark/lawyer/contact":                            true,
	"GET /trademark/lawyer/ranking":                            true,
	"GET /trademark/lawyer/search":                             true,
	"POST /trademark/office-action/search":                     true,
	"GET /trademark/office-action/uspto/document":              true,
	"GET /trademark/owner/ranking":                             true,
	"GET /trademark/owner/search":                              true,
	"POST /trademark/search":                                   true,
	"POST /trademark/search/summary":                           true,
	"GET /trademark/search/tips":                               true,
	"POST /trademark/ttab/search":                              true,
	"GET /trademark/ttab/{case_number}":                        true,
	"POST /upload/presign":                                     true,
}

var rawReadyEndpoints = map[string]bool{
	"POST /auth/login": true,
}

func renderGo(hash string, specPath string, endpoints []endpoint) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "package openapi\n\n")
	fmt.Fprintf(&buf, "import \"encoding/json\"\n\n")
	fmt.Fprintf(&buf, "const SourceHash = %q\n\n", hash)
	fmt.Fprintf(&buf, "const SourcePath = %q\n\n", filepath.ToSlash(specPath))
	fmt.Fprintf(&buf, "var Endpoints = []Endpoint{\n")
	for _, endpoint := range endpoints {
		fmt.Fprintf(&buf, "{Method:%q, Path:%q, OperationID:%q, Summary:%q, Tags:%#v, HasBody:%t, ParamCount:%d, Coverage:%q},\n",
			endpoint.Method,
			endpoint.Path,
			endpoint.OperationID,
			endpoint.Summary,
			endpoint.Tags,
			endpoint.HasBody,
			endpoint.ParamCount,
			endpoint.Coverage,
		)
	}
	fmt.Fprintf(&buf, "}\n")
	fmt.Fprintf(&buf, "\nvar EndpointSchemas = map[string]EndpointSchema{\n")
	for i, endpoint := range endpoints {
		fmt.Fprintf(&buf, "%q: {Endpoint: Endpoints[%d], Parameters: json.RawMessage(%q), Responses: json.RawMessage(%q)",
			endpoint.Method+" "+endpoint.Path,
			i,
			string(endpoint.Parameters),
			string(endpoint.Responses),
		)
		if len(endpoint.Definitions) > 0 {
			fmt.Fprintf(&buf, ", Definitions: map[string]json.RawMessage{")
			names := sortedDefinitionNames(endpoint.Definitions)
			for _, name := range names {
				fmt.Fprintf(&buf, "%q: json.RawMessage(%q),", name, string(endpoint.Definitions[name]))
			}
			fmt.Fprintf(&buf, "}")
		}
		fmt.Fprintf(&buf, "},\n")
	}
	fmt.Fprintf(&buf, "}\n")
	formatted, err := format.Source(buf.Bytes())
	must(err)
	return formatted
}

func mustMarshal(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	must(err)
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}

func referencedDefinitions(definitions map[string]json.RawMessage, raws ...json.RawMessage) map[string]json.RawMessage {
	seen := map[string]bool{}
	queue := []string{}
	for _, raw := range raws {
		collectRefs(raw, seen, &queue)
	}
	out := map[string]json.RawMessage{}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		raw, ok := definitions[name]
		if !ok {
			continue
		}
		out[name] = raw
		collectRefs(raw, seen, &queue)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func collectRefs(raw json.RawMessage, seen map[string]bool, queue *[]string) {
	if len(raw) == 0 {
		return
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return
	}
	collectRefsValue(value, seen, queue)
}

func collectRefsValue(value any, seen map[string]bool, queue *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		if ref, ok := typed["$ref"].(string); ok {
			const prefix = "#/definitions/"
			if strings.HasPrefix(ref, prefix) {
				name := strings.TrimPrefix(ref, prefix)
				if name != "" && !seen[name] {
					seen[name] = true
					*queue = append(*queue, name)
				}
			}
		}
		for _, child := range typed {
			collectRefsValue(child, seen, queue)
		}
	case []any:
		for _, child := range typed {
			collectRefsValue(child, seen, queue)
		}
	}
}

func sortedDefinitionNames(definitions map[string]json.RawMessage) []string {
	names := make([]string, 0, len(definitions))
	for name := range definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func renderInventory(hash string, specPath string, endpoints []endpoint) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# OpenAPI Endpoint Inventory\n\n")
	fmt.Fprintf(&buf, "- Source: `%s`\n", filepath.ToSlash(specPath))
	fmt.Fprintf(&buf, "- SHA256: `%s`\n", hash)
	fmt.Fprintf(&buf, "- Endpoints: `%d`\n\n", len(endpoints))
	fmt.Fprintf(&buf, "| Coverage | Method | Path | Tags | Summary |\n")
	fmt.Fprintf(&buf, "|---|---|---|---|---|\n")
	for _, endpoint := range endpoints {
		fmt.Fprintf(&buf, "| %s | %s | `%s` | %s | %s |\n",
			escapeCell(endpoint.Coverage),
			escapeCell(endpoint.Method),
			escapeCell(endpoint.Path),
			escapeCell(strings.Join(endpoint.Tags, ",")),
			escapeCell(endpoint.Summary),
		)
	}
	return buf.Bytes()
}

func escapeCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}

func checkFile(path string, want []byte) {
	got, err := os.ReadFile(path)
	must(err)
	if !bytes.Equal(got, want) {
		fmt.Fprintf(os.Stderr, "%s is out of date; run make openapi-sync\n", path)
		os.Exit(1)
	}
}

func writeFile(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
