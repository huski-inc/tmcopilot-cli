package tmc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/output"
)

func callAPIAndWrite(cmd *cobra.Command, opts *globalOptions, method string, path string, query url.Values, body any) error {
	return handleCommand(cmd, func() error {
		return executeAPIAndWrite(cmd, opts, method, path, query, body)
	})
}

func executeAPIAndWrite(cmd *cobra.Command, opts *globalOptions, method string, path string, query url.Values, body any) error {
	return executeAPIAndWriteWithHeaders(cmd, opts, method, path, query, body, nil)
}

func executeAPIAndWriteWithHeaders(cmd *cobra.Command, opts *globalOptions, method string, path string, query url.Values, body any, headers map[string]string) error {
	method = strings.ToUpper(method)
	if err := ensureWriteAllowed(opts, method, path); err != nil {
		return err
	}
	rt, err := commandRuntime(cmd, opts, !opts.dryRun)
	if err != nil {
		return err
	}
	for key, value := range headers {
		rt.Client.ExtraHeaders[key] = value
	}
	plan := buildAPIRequestPlan(method, path, query, body, opts, rt, headers)
	if opts.requestOut != "" {
		if err := writeRequestPlan(opts.requestOut, plan); err != nil {
			return err
		}
	}
	if opts.dryRun {
		return writeResult(rt, plan, nil)
	}
	resp, err := rt.Client.Do(cmd.Context(), method, path, query, body)
	if err != nil {
		return err
	}
	return writeResult(rt, resp.DataOrRaw(rt.RawEnvelope), map[string]any{
		"status_code": resp.StatusCode,
		"trace_id":    resp.Headers.Get("X-Trace-ID"),
	})
}

type apiRequestPlan struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   url.Values        `json:"query,omitempty"`
	Body    any               `json:"body,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

func buildAPIRequestPlan(method string, path string, query url.Values, body any, opts *globalOptions, rt *runtimeContext, extraHeaders map[string]string) apiRequestPlan {
	headers := map[string]string{
		"Accept":                  "application/json",
		"X-TMCopilot-CLI-Command": cmdPathFromRuntime(rt),
	}
	if body != nil {
		headers["Content-Type"] = "application/json"
	}
	if rt != nil && rt.Profile.WorkspaceID != "" {
		headers["X-TMCopilot-Workspace-ID"] = rt.Profile.WorkspaceID
	}
	if opts != nil && strings.TrimSpace(opts.idempotencyKey) != "" {
		headers["Idempotency-Key"] = strings.TrimSpace(opts.idempotencyKey)
	}
	for key, value := range extraHeaders {
		headers[key] = value
	}
	return apiRequestPlan{
		Method:  method,
		Path:    path,
		Query:   query,
		Body:    body,
		Headers: headers,
	}
}

func cmdPathFromRuntime(rt *runtimeContext) string {
	if rt == nil || rt.Client == nil {
		return ""
	}
	return rt.Client.ExtraHeaders["X-TMCopilot-CLI-Command"]
}

func writeRequestPlan(path string, plan apiRequestPlan) error {
	return writeJSONPlan(path, plan)
}

func writeJSONPlan(path string, plan any) error {
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return output.WriteRawFile(path, raw)
}

func ensureWriteAllowed(opts *globalOptions, method string, path string) error {
	if opts != nil && opts.dryRun {
		return nil
	}
	if !isDestructiveRequest(method, path) {
		return nil
	}
	if opts != nil && opts.yes {
		return nil
	}
	return fmt.Errorf("destructive request %s %s requires --yes or --dry-run", method, path)
}

func isDestructiveRequest(method string, path string) bool {
	if method == http.MethodDelete {
		return true
	}
	lowerPath := strings.ToLower(path)
	return strings.Contains(lowerPath, "/delete") || strings.Contains(lowerPath, "/revoke")
}

func setQuery(query url.Values, key string, value string) {
	if strings.TrimSpace(value) != "" {
		query.Set(key, value)
	}
}

func bodyFromDataOrBuilder(data string, builder func() (any, error)) (any, error) {
	if strings.TrimSpace(data) != "" {
		return readDataArg(data)
	}
	return builder()
}

func splitIntValues(values []string, field string) ([]int, error) {
	split := splitStringValues(values)
	if len(split) == 0 {
		return nil, nil
	}
	ints := make([]int, 0, len(split))
	for _, value := range split {
		num, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid integer for %s: %q", field, value)
		}
		ints = append(ints, num)
	}
	return ints, nil
}

func splitStringValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}
