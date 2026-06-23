package tmc

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/output"
)

type pageResult struct {
	Items      []map[string]any `json:"items"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

type listOptions struct {
	page         int
	pageSize     int
	pageAll      bool
	all          bool
	maxPages     int
	params       []string
	fields       string
	maxRows      int
	manifestPath string
	progress     bool
	sortField    string
	sortDir      string
	sortParam    string
	sortDirParam string
	filters      []queryFlag
}

type listCommandSpec struct {
	Use          string
	Short        string
	Path         string
	SortParam    string
	SortDirParam string
	Filters      []queryFlagSpec
}

type queryFlagSpec struct {
	Flag        string
	Param       string
	Default     string
	Description string
}

type queryFlag struct {
	flag        string
	param       string
	value       string
	defaultVal  string
	description string
}

func newPagedListCommand(rootOpts *globalOptions, spec listCommandSpec) *cobra.Command {
	list := &listOptions{
		page:         1,
		pageSize:     20,
		sortParam:    defaultString(spec.SortParam, "sort"),
		sortDirParam: defaultString(spec.SortDirParam, "sort_dir"),
		filters:      make([]queryFlag, len(spec.Filters)),
	}
	for i, filter := range spec.Filters {
		param := filter.Param
		if param == "" {
			param = strings.ReplaceAll(filter.Flag, "-", "_")
		}
		list.filters[i] = queryFlag{
			flag:        filter.Flag,
			param:       param,
			defaultVal:  filter.Default,
			description: filter.Description,
		}
	}
	cmd := &cobra.Command{
		Use:   spec.Use,
		Short: spec.Short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, rootOpts, true)
				if err != nil {
					return err
				}
				query, err := buildListQuery(list)
				if err != nil {
					return err
				}
				if list.pageAll || list.all {
					return streamPages(cmd, rt, spec.Path, query, list)
				}
				resp, err := rt.Client.Do(cmd.Context(), "GET", spec.Path, query, nil)
				if err != nil {
					return err
				}
				return writeResult(rt, resp.DataOrRaw(rt.RawEnvelope), map[string]any{
					"status_code": resp.StatusCode,
					"trace_id":    resp.Headers.Get("X-Trace-ID"),
				})
			})
		},
	}
	addListFlags(cmd, list)
	return cmd
}

func addListFlags(cmd *cobra.Command, opts *listOptions) {
	cmd.Flags().IntVar(&opts.page, "page", 1, "page number")
	cmd.Flags().IntVar(&opts.pageSize, "page-size", 20, "page size")
	cmd.Flags().BoolVar(&opts.pageAll, "page-all", false, "fetch all pages by paging through the API")
	cmd.Flags().BoolVar(&opts.all, "all", false, "alias for --page-all")
	cmd.Flags().IntVar(&opts.maxPages, "max-pages", 0, "maximum pages to fetch with --page-all")
	cmd.Flags().StringArrayVar(&opts.params, "param", nil, "additional query parameter key=value; repeatable")
	cmd.Flags().StringVar(&opts.fields, "fields", "", "comma-separated output fields for csv/ndjson/json streaming")
	cmd.Flags().IntVar(&opts.maxRows, "max-rows", 0, "maximum rows to stream with --page-all")
	cmd.Flags().StringVar(&opts.manifestPath, "manifest", "", "write export manifest JSON to a file")
	cmd.Flags().BoolVar(&opts.progress, "progress", false, "write page progress to stderr while streaming")
	cmd.Flags().StringVar(&opts.sortField, "sort", "", "sort field")
	cmd.Flags().StringVar(&opts.sortDir, "sort-dir", "", "sort direction")
	for i := range opts.filters {
		filter := &opts.filters[i]
		cmd.Flags().StringVar(&filter.value, filter.flag, filter.defaultVal, filter.description)
	}
}

func buildListQuery(opts *listOptions) (url.Values, error) {
	query, err := parseParams(opts.params)
	if err != nil {
		return nil, err
	}
	if opts.page < 1 {
		opts.page = 1
	}
	if opts.pageSize < 1 {
		opts.pageSize = 20
	}
	query.Set("page", strconv.Itoa(opts.page))
	query.Set("page_size", strconv.Itoa(opts.pageSize))
	if opts.sortField != "" {
		query.Set(opts.sortParam, opts.sortField)
	}
	if opts.sortDir != "" {
		query.Set(opts.sortDirParam, opts.sortDir)
	}
	for _, filter := range opts.filters {
		if strings.TrimSpace(filter.value) != "" {
			query.Set(filter.param, filter.value)
		}
	}
	return query, nil
}

func streamPages(cmd *cobra.Command, rt *runtimeContext, apiPath string, query url.Values, opts *listOptions) error {
	startedAt := time.Now().UTC()
	format := strings.ToLower(rt.Format)
	if format == "" || format == "pretty" || format == "raw" {
		format = "json"
	}
	if format != "json" && format != "csv" && format != "ndjson" {
		return fmt.Errorf("--page-all supports --format json, csv, or ndjson")
	}
	var writer io.Writer = cmd.OutOrStdout()
	outputPath := rt.OutputPath
	tempPath := ""
	if rt.OutputPath != "" {
		tempPath = rt.OutputPath + ".tmp"
		file, err := output.CreateFile(tempPath)
		if err != nil {
			return err
		}
		defer os.Remove(tempPath)
		writer = file
		outputPath = tempPath
	}

	fields := splitFields(opts.fields)
	summary := map[string]any{
		"path":        rt.OutputPath,
		"format":      format,
		"pages":       0,
		"rows":        0,
		"total":       0,
		"total_pages": 0,
		"started_at":  startedAt.Format(time.RFC3339),
	}
	var err error
	switch format {
	case "json":
		err = streamJSONPages(cmd, rt, writer, apiPath, query, opts, fields, summary)
	case "ndjson":
		err = streamNDJSONPages(cmd, rt, writer, apiPath, query, opts, fields, summary)
	case "csv":
		err = streamCSVPages(cmd, rt, writer, apiPath, query, opts, fields, summary)
	}
	if err != nil {
		return err
	}
	summary["completed_at"] = time.Now().UTC().Format(time.RFC3339)
	if closer, ok := writer.(io.Closer); ok && outputPath != "" {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	if tempPath != "" {
		if err := os.Rename(tempPath, rt.OutputPath); err != nil {
			return err
		}
	}
	if opts.manifestPath != "" {
		if err := writeExportManifest(opts.manifestPath, apiPath, query, fields, summary); err != nil {
			return err
		}
	}
	if rt.OutputPath != "" {
		return output.WriteTo(cmd.OutOrStdout(), "json", "", summary, nil)
	}
	return nil
}

func streamJSONPages(cmd *cobra.Command, rt *runtimeContext, writer io.Writer, apiPath string, query url.Values, opts *listOptions, fields []string, summary map[string]any) error {
	if _, err := writer.Write([]byte("[\n")); err != nil {
		return err
	}
	wrote := false
	err := eachPage(cmd, rt, apiPath, query, opts, func(page pageResult) error {
		updateSummary(summary, page)
		for _, item := range page.Items {
			if reachedMaxRows(summary, opts) {
				return errStopRows
			}
			projected := projectItem(item, fields)
			raw, err := json.Marshal(projected)
			if err != nil {
				return err
			}
			if wrote {
				if _, err := writer.Write([]byte(",\n")); err != nil {
					return err
				}
			}
			if _, err := writer.Write(raw); err != nil {
				return err
			}
			wrote = true
			summary["rows"] = summary["rows"].(int) + 1
		}
		return nil
	})
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte("\n]\n"))
	return err
}

func streamNDJSONPages(cmd *cobra.Command, rt *runtimeContext, writer io.Writer, apiPath string, query url.Values, opts *listOptions, fields []string, summary map[string]any) error {
	enc := json.NewEncoder(writer)
	return eachPage(cmd, rt, apiPath, query, opts, func(page pageResult) error {
		updateSummary(summary, page)
		for _, item := range page.Items {
			if reachedMaxRows(summary, opts) {
				return errStopRows
			}
			if err := enc.Encode(projectItem(item, fields)); err != nil {
				return err
			}
			summary["rows"] = summary["rows"].(int) + 1
		}
		return nil
	})
}

func streamCSVPages(cmd *cobra.Command, rt *runtimeContext, writer io.Writer, apiPath string, query url.Values, opts *listOptions, fields []string, summary map[string]any) error {
	cw := csv.NewWriter(writer)
	wroteHeader := false
	err := eachPage(cmd, rt, apiPath, query, opts, func(page pageResult) error {
		updateSummary(summary, page)
		for _, item := range page.Items {
			if reachedMaxRows(summary, opts) {
				return errStopRows
			}
			if len(fields) == 0 {
				fields = inferCSVFields(item)
			}
			if !wroteHeader {
				if err := cw.Write(fields); err != nil {
					return err
				}
				wroteHeader = true
			}
			row := make([]string, 0, len(fields))
			for _, field := range fields {
				row = append(row, csvValue(item[field]))
			}
			if err := cw.Write(row); err != nil {
				return err
			}
			summary["rows"] = summary["rows"].(int) + 1
		}
		return nil
	})
	cw.Flush()
	if flushErr := cw.Error(); flushErr != nil {
		return flushErr
	}
	return err
}

func eachPage(cmd *cobra.Command, rt *runtimeContext, apiPath string, query url.Values, opts *listOptions, visit func(page pageResult) error) error {
	pageNo := opts.page
	for {
		query.Set("page", strconv.Itoa(pageNo))
		resp, err := rt.Client.Do(cmd.Context(), "GET", apiPath, query, nil)
		if err != nil {
			return err
		}
		var page pageResult
		if err := resp.DecodeData(&page); err != nil {
			return err
		}
		page.Items = normalizePageItems(page.Items)
		if err := visit(page); err != nil {
			if errors.Is(err, errStopRows) {
				return nil
			}
			return err
		}
		if opts.progress {
			fmt.Fprintf(cmd.ErrOrStderr(), "page=%d rows=%d total=%d\n", pageNo, pageRows(page), page.Total)
		}
		if page.TotalPages <= 0 || pageNo >= page.TotalPages {
			return nil
		}
		if opts.maxPages > 0 && pageNo-opts.page+1 >= opts.maxPages {
			return nil
		}
		pageNo++
	}
}

func normalizePageItems(items []map[string]any) []map[string]any {
	if len(items) == 0 {
		return items
	}
	out := make([]map[string]any, len(items))
	for i, item := range items {
		normalized, ok := normalizeResponseValue("", item).(map[string]any)
		if ok {
			out[i] = normalized
		} else {
			out[i] = item
		}
	}
	return out
}

var errStopRows = errors.New("maximum streamed rows reached")

func reachedMaxRows(summary map[string]any, opts *listOptions) bool {
	if opts == nil || opts.maxRows <= 0 {
		return false
	}
	rows, _ := summary["rows"].(int)
	return rows >= opts.maxRows
}

func pageRows(page pageResult) int {
	return len(page.Items)
}

func writeExportManifest(path string, apiPath string, query url.Values, fields []string, summary map[string]any) error {
	manifest := map[string]any{
		"api_path": apiPath,
		"query":    query,
		"fields":   fields,
		"summary":  summary,
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return output.WriteRawFile(path, raw)
}

func updateSummary(summary map[string]any, page pageResult) {
	summary["pages"] = summary["pages"].(int) + 1
	summary["total"] = page.Total
	summary["total_pages"] = page.TotalPages
}

func splitFields(value string) []string {
	parts := strings.Split(value, ",")
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			fields = append(fields, part)
		}
	}
	return fields
}

func projectItem(item map[string]any, fields []string) map[string]any {
	if len(fields) == 0 {
		return item
	}
	projected := make(map[string]any, len(fields))
	for _, field := range fields {
		projected[field] = item[field]
	}
	return projected
}

func inferCSVFields(item map[string]any) []string {
	fields := make([]string, 0, len(item))
	for key, value := range item {
		switch value.(type) {
		case string, float64, bool, nil:
			fields = append(fields, key)
		}
	}
	sort.Strings(fields)
	return fields
}

func csvValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(raw)
	}
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
