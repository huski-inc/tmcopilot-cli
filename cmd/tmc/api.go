package tmc

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
	"github.com/huski-inc/tmcopilot-cli/internal/output"
)

func newAPICommand(opts *globalOptions) *cobra.Command {
	var params []string
	var headers []string
	var data string
	var bodyFile string
	cmd := &cobra.Command{
		Use:   "api <method> <path>",
		Short: "Call a TMCopilot REST API endpoint",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				method := strings.ToUpper(args[0])
				switch method {
				case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				default:
					return fmt.Errorf("unsupported method %q", method)
				}
				query, err := parseParams(params)
				if err != nil {
					return err
				}
				if strings.TrimSpace(data) != "" && strings.TrimSpace(bodyFile) != "" {
					return fmt.Errorf("use only one of --data or --body-file")
				}
				bodyInput := data
				if strings.TrimSpace(bodyFile) != "" {
					bodyInput = "@" + bodyFile
				}
				body, err := readDataArg(bodyInput)
				if err != nil {
					return err
				}
				headerMap, err := parseHeaders(headers)
				if err != nil {
					return err
				}
				return executeAPIAndWriteWithHeaders(cmd, opts, method, args[1], query, body, headerMap)
			})
		},
	}
	cmd.Flags().StringArrayVar(&params, "param", nil, "query parameter key=value; repeatable")
	cmd.Flags().StringArrayVar(&headers, "header", nil, "HTTP header key=value; repeatable")
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "JSON request body file")
	cmd.AddCommand(newAPICatalogCommand(opts))
	cmd.AddCommand(newAPIEndpointCommand(opts))
	cmd.AddCommand(newAPIEndpointSchemaCommand(opts))
	cmd.AddCommand(newAPIDownloadCommand(opts))
	return cmd
}

func newAPICatalogCommand(opts *globalOptions) *cobra.Command {
	var tag string
	var coverage string
	var search string
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "List Swagger endpoints known to this CLI build",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				items, hiddenInternal := openapi.FilterEndpointsForCatalog(tag, coverage)
				search = strings.TrimSpace(strings.ToLower(search))
				if search != "" {
					filtered := make([]openapi.Endpoint, 0, len(items))
					for _, item := range items {
						haystack := strings.ToLower(item.Method + " " + item.Path + " " + item.Summary + " " + strings.Join(item.Tags, " "))
						if strings.Contains(haystack, search) {
							filtered = append(filtered, item)
						}
					}
					items = filtered
				}
				return writeResult(rt, map[string]any{
					"source_hash":           openapi.SourceHash,
					"source_path":           openapi.SourcePath,
					"count":                 len(items),
					"hidden_internal_count": hiddenInternal,
					"items":                 items,
				}, nil)
			})
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "filter by Swagger tag")
	cmd.Flags().StringVar(&coverage, "coverage", "", "filter by coverage: typed, raw-ready, raw")
	cmd.Flags().StringVar(&search, "search", "", "search method, path, tag, or summary")
	return cmd
}

func newAPIEndpointCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "endpoint <method> <path>",
		Short: "Show Swagger metadata for one endpoint",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				endpoint, ok := openapi.FindEndpoint(args[0], normalizeCatalogPath(args[1]))
				if !ok {
					return fmt.Errorf("endpoint not found in catalog: %s %s", strings.ToUpper(args[0]), args[1])
				}
				if openapi.IsInternalEndpoint(endpoint) {
					return fmt.Errorf("endpoint not found in catalog")
				}
				return writeResult(rt, endpoint, map[string]any{
					"source_hash": openapi.SourceHash,
					"source_path": openapi.SourcePath,
				})
			})
		},
	}
}

func normalizeCatalogPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/api/v1")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func parseHeaders(values []string) (map[string]string, error) {
	headers := map[string]string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key, val, ok := strings.Cut(value, "=")
		if !ok {
			return nil, fmt.Errorf("invalid header %q, expected key=value", value)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" {
			return nil, fmt.Errorf("invalid header %q, key is empty", value)
		}
		headers[key] = val
	}
	return headers, nil
}

func newAPIDownloadCommand(opts *globalOptions) *cobra.Command {
	var params []string
	var data string
	var bodyFile string
	var headers []string
	cmd := &cobra.Command{
		Use:   "download <method> <path>",
		Short: "Call an API endpoint and write the raw response body to --output",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				if strings.TrimSpace(opts.output) == "" {
					return fmt.Errorf("--output is required for api download")
				}
				method := strings.ToUpper(args[0])
				switch method {
				case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				default:
					return fmt.Errorf("unsupported method %q", method)
				}
				query, err := parseParams(params)
				if err != nil {
					return err
				}
				if strings.TrimSpace(data) != "" && strings.TrimSpace(bodyFile) != "" {
					return fmt.Errorf("use only one of --data or --body-file")
				}
				bodyInput := data
				if strings.TrimSpace(bodyFile) != "" {
					bodyInput = "@" + bodyFile
				}
				body, err := readDataArg(bodyInput)
				if err != nil {
					return err
				}
				headerMap, err := parseHeaders(headers)
				if err != nil {
					return err
				}
				return executeAPIDownloadAndWrite(cmd, opts, method, args[1], query, body, headerMap)
			})
		},
	}
	cmd.Flags().StringArrayVar(&params, "param", nil, "query parameter key=value; repeatable")
	cmd.Flags().StringArrayVar(&headers, "header", nil, "HTTP header key=value; repeatable")
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "JSON request body file")
	return cmd
}

func executeAPIDownloadAndWrite(cmd *cobra.Command, opts *globalOptions, method string, path string, query url.Values, body any, headers map[string]string) error {
	if opts == nil || strings.TrimSpace(opts.output) == "" {
		return fmt.Errorf("--output is required for download")
	}
	if isRawAPIFallbackCommand(cmd) {
		if err := ensurePublicAPIFallbackAllowed(method, path); err != nil {
			return err
		}
	}
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
	if err := output.WriteRawFile(opts.output, resp.Raw); err != nil {
		return err
	}
	return outputDownloadSummary(cmd, rt, opts.output, resp.StatusCode, resp.Headers.Get("X-Trace-ID"), len(resp.Raw))
}

func outputDownloadSummary(cmd *cobra.Command, rt *runtimeContext, path string, statusCode int, traceID string, bytes int) error {
	return output.WriteTo(cmd.OutOrStdout(), "json", "", map[string]any{
		"path":  path,
		"bytes": bytes,
	}, map[string]any{
		"status_code": statusCode,
		"trace_id":    traceID,
	})
}
