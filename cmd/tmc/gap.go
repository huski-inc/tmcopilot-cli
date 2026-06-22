package tmc

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

func newGapCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gap",
		Short: "Work with gap analyses",
	}
	cmd.AddCommand(newGapListCommand(opts))
	cmd.AddCommand(newGapCreateCommand(opts))
	cmd.AddCommand(newGapGetCommand(opts))
	cmd.AddCommand(newGapDeleteCommand(opts))
	cmd.AddCommand(newGapRunCommand(opts))
	cmd.AddCommand(newGapWaitCommand(opts))
	cmd.AddCommand(newGapResultsCommand(opts))
	cmd.AddCommand(newGapReportsCommand(opts))
	cmd.AddCommand(newGapGenerateReportCommand(opts))
	cmd.AddCommand(newGapSharesCommand(opts))
	return cmd
}

func newGapListCommand(opts *globalOptions) *cobra.Command {
	var search string
	var status string
	var limit int
	var offset int
	var params []string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List gap analyses",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, err := parseParams(params)
			if err != nil {
				return err
			}
			setQuery(query, "search", search)
			setQuery(query, "status", status)
			if limit > 0 {
				query.Set("limit", strconv.Itoa(limit))
			}
			if offset > 0 {
				query.Set("offset", strconv.Itoa(offset))
			}
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/gap-analyses", query, nil)
		},
	}
	cmd.Flags().StringVar(&search, "search", "", "search keyword")
	cmd.Flags().StringVar(&status, "status", "", "status filter")
	cmd.Flags().IntVar(&limit, "limit", 50, "items per page")
	cmd.Flags().IntVar(&offset, "offset", 0, "offset")
	cmd.Flags().StringArrayVar(&params, "param", nil, "additional query parameter key=value; repeatable")
	return cmd
}

func newGapCreateCommand(opts *globalOptions) *cobra.Command {
	var data string
	var title, baseCompany, baseSource, benchmarkCompany, benchmarkSource string
	var competitorID, businessContext, productFocus, reportAudience string
	var baseAliases, benchmarkAliases, niceClasses, statusFilter, targetMarkets []string
	var includeLive, includePending, includeAbandoned, runImmediately bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a gap analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.GapCreateRequest{
					Title:                 strings.TrimSpace(title),
					BaseCompanyName:       strings.TrimSpace(baseCompany),
					BaseSourceType:        strings.TrimSpace(baseSource),
					BenchmarkCompanyName:  strings.TrimSpace(benchmarkCompany),
					BenchmarkSourceType:   strings.TrimSpace(benchmarkSource),
					CompetitorID:          strings.TrimSpace(competitorID),
					BusinessContext:       strings.TrimSpace(businessContext),
					ProductFocus:          strings.TrimSpace(productFocus),
					ReportAudience:        strings.TrimSpace(reportAudience),
					BaseOwnerAliases:      splitStringValues(baseAliases),
					BenchmarkOwnerAliases: splitStringValues(benchmarkAliases),
					NiceClasses:           splitStringValues(niceClasses),
					StatusFilter:          splitStringValues(statusFilter),
					TargetMarkets:         splitStringValues(targetMarkets),
				}
				if cmd.Flags().Changed("include-live") {
					req.IncludeLive = &includeLive
				}
				if cmd.Flags().Changed("include-pending") {
					req.IncludePending = &includePending
				}
				if cmd.Flags().Changed("include-abandoned") {
					req.IncludeAbandoned = &includeAbandoned
				}
				if cmd.Flags().Changed("run-immediately") {
					req.RunImmediately = &runImmediately
				}
				if req.Empty() {
					return nil, fmt.Errorf("gap create requires --data or at least one input flag")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/gap-analyses", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&title, "title", "", "analysis title")
	cmd.Flags().StringVar(&baseCompany, "base-company-name", "", "base company name")
	cmd.Flags().StringVar(&baseSource, "base-source-type", "", "base source type")
	cmd.Flags().StringVar(&benchmarkCompany, "benchmark-company-name", "", "benchmark company name")
	cmd.Flags().StringVar(&benchmarkSource, "benchmark-source-type", "", "benchmark source type")
	cmd.Flags().StringVar(&competitorID, "competitor-id", "", "competitor ID")
	cmd.Flags().StringVar(&businessContext, "business-context", "", "business context")
	cmd.Flags().StringVar(&productFocus, "product-focus", "", "product focus")
	cmd.Flags().StringVar(&reportAudience, "report-audience", "", "report audience")
	cmd.Flags().StringArrayVar(&baseAliases, "base-owner-alias", nil, "base owner alias; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&benchmarkAliases, "benchmark-owner-alias", nil, "benchmark owner alias; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&niceClasses, "nice-class", nil, "Nice class; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&statusFilter, "status-filter", nil, "status filter; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&targetMarkets, "target-market", nil, "target market; repeatable or comma-separated")
	cmd.Flags().BoolVar(&includeLive, "include-live", false, "include live trademarks")
	cmd.Flags().BoolVar(&includePending, "include-pending", false, "include pending trademarks")
	cmd.Flags().BoolVar(&includeAbandoned, "include-abandoned", false, "include abandoned trademarks")
	cmd.Flags().BoolVar(&runImmediately, "run-immediately", false, "run the analysis immediately")
	return cmd
}

func newGapGetCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a gap analysis",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/gap-analyses/"+url.PathEscape(args[0]), nil, nil)
		},
	}
}

func newGapDeleteCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a gap analysis",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, http.MethodDelete, "/gap-analyses/"+url.PathEscape(args[0]), nil, nil)
		},
	}
}

func newGapRunCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "run <id>",
		Short: "Run a gap analysis",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/gap-analyses/"+url.PathEscape(args[0])+"/run", nil, nil)
		},
	}
}

func newGapWaitCommand(opts *globalOptions) *cobra.Command {
	var pollInterval time.Duration
	var waitTimeout time.Duration
	cmd := &cobra.Command{
		Use:   "wait <id>",
		Short: "Poll a gap analysis until it reaches a terminal status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				if pollInterval <= 0 {
					return fmt.Errorf("--poll-interval must be positive")
				}
				rt, err := commandRuntime(cmd, opts, true)
				if err != nil {
					return err
				}
				deadline := time.Now().Add(waitTimeout)
				id := url.PathEscape(args[0])
				polls := 0
				for {
					polls++
					resp, err := rt.Client.Do(cmd.Context(), "GET", "/gap-analyses/"+id, nil, nil)
					if err != nil {
						return err
					}
					var data map[string]any
					if err := resp.DecodeData(&data); err != nil {
						return err
					}
					status := strings.ToLower(strings.TrimSpace(statusFromMap(data)))
					if isTerminalStatus(status) {
						meta := map[string]any{
							"status_code": resp.StatusCode,
							"trace_id":    resp.Headers.Get("X-Trace-ID"),
							"polls":       polls,
							"status":      status,
						}
						if err := writeResult(rt, data, meta); err != nil {
							return err
						}
						if isFailedStatus(status) {
							return fmt.Errorf("gap analysis reached failed status: %s", status)
						}
						return nil
					}
					if waitTimeout > 0 && time.Now().Add(pollInterval).After(deadline) {
						return fmt.Errorf("timed out waiting for gap analysis; last status: %s", status)
					}
					time.Sleep(pollInterval)
				}
			})
		},
	}
	cmd.Flags().DurationVar(&pollInterval, "poll-interval", 5*time.Second, "poll interval")
	cmd.Flags().DurationVar(&waitTimeout, "wait-timeout", 10*time.Minute, "maximum wait time")
	return cmd
}

func statusFromMap(data map[string]any) string {
	for _, key := range []string{"status", "state", "analysis_status", "run_status"} {
		if value, ok := data[key].(string); ok {
			return value
		}
	}
	return ""
}

func isTerminalStatus(status string) bool {
	switch status {
	case "completed", "complete", "done", "succeeded", "success", "failed", "failure", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func isFailedStatus(status string) bool {
	switch status {
	case "failed", "failure", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func newGapResultsCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "results <id>",
		Short: "Get gap analysis results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/gap-analyses/"+url.PathEscape(args[0])+"/results", nil, nil)
		},
	}
}

func newGapReportsCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "reports <id>",
		Short: "List gap analysis reports",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/gap-analyses/"+url.PathEscape(args[0])+"/reports", nil, nil)
		},
	}
}

func newGapGenerateReportCommand(opts *globalOptions) *cobra.Command {
	var data string
	var selectedClasses []string
	cmd := &cobra.Command{
		Use:   "generate-report <id>",
		Short: "Generate a gap analysis report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				return openapi.GapGenerateReportRequest{SelectedClasses: splitStringValues(selectedClasses)}, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/gap-analyses/"+url.PathEscape(args[0])+"/reports/generate", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringArrayVar(&selectedClasses, "selected-class", nil, "selected class; repeatable or comma-separated")
	return cmd
}

func newGapSharesCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shares",
		Short: "Work with gap analysis shares",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "create <id>",
		Short: "Create a gap analysis share",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/gap-analyses/"+url.PathEscape(args[0])+"/share", nil, nil)
		},
	})
	var limit int
	list := &cobra.Command{
		Use:   "list <id>",
		Short: "List gap analysis shares",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			if limit > 0 {
				query.Set("limit", strconv.Itoa(limit))
			}
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/gap-analyses/"+url.PathEscape(args[0])+"/shares", query, nil)
		},
	}
	list.Flags().IntVar(&limit, "limit", 50, "maximum shares")
	cmd.AddCommand(list)
	cmd.AddCommand(&cobra.Command{
		Use:   "revoke <id> <token>",
		Short: "Revoke a gap analysis share",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := url.PathEscape(args[0])
			token := url.PathEscape(args[1])
			return callAPIAndWrite(cmd, opts, http.MethodDelete, "/gap-analyses/"+id+"/shares/"+token, nil, nil)
		},
	})
	get := &cobra.Command{
		Use:   "get <token>",
		Short: "Get a gap analysis share by token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token := strings.TrimSpace(args[0])
			if token == "" {
				return fmt.Errorf("token is required")
			}
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/gap-analyses/shares/"+url.PathEscape(token), nil, nil)
		},
	}
	cmd.AddCommand(get)
	return cmd
}
