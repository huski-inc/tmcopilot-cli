package tmc

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

func newDomainCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "Search domain name evidence",
	}
	cmd.AddCommand(newDomainSearchCommand(opts))
	cmd.AddCommand(newDomainMaxSimilarityCommand(opts))
	return cmd
}

func newDomainSearchCommand(opts *globalOptions) *cobra.Command {
	var data string
	var keyword string
	var limit int
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search domain names by keyword",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.DomainSearchRequest{
					Keyword: strings.TrimSpace(keyword),
				}
				if limit > 0 {
					req.Limit = limit
				}
				if req.Keyword == "" {
					return nil, fmt.Errorf("--keyword is required")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/domain/search", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&keyword, "keyword", "", "keyword to search")
	cmd.Flags().IntVar(&limit, "limit", 0, "result limit")
	return cmd
}

func newDomainMaxSimilarityCommand(opts *globalOptions) *cobra.Command {
	var data string
	var keyword string
	cmd := &cobra.Command{
		Use:   "max-similarity",
		Short: "Get max domain name similarity for a keyword",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.DomainMaxSimilarityRequest{
					Keyword: strings.TrimSpace(keyword),
				}
				if req.Keyword == "" {
					return nil, fmt.Errorf("--keyword is required")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/domain/max-similarity", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&keyword, "keyword", "", "keyword to check")
	return cmd
}
