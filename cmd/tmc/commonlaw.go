package tmc

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

func newCommonLawCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "common-law",
		Aliases: []string{"commonlaw"},
		Short:   "Search common-law sources",
	}
	cmd.AddCommand(newCommonLawSearchCommand(opts))
	cmd.AddCommand(newCommonLawMaxSimilarityCommand(opts))
	return cmd
}

func newCommonLawSearchCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search common-law evidence sources",
	}
	cmd.AddCommand(newCommonLawSearchEndpointCommand(opts, "app-store", "Search app stores for common-law evidence", "/common-law/search/app-store"))
	cmd.AddCommand(newCommonLawSearchEndpointCommand(opts, "ecommerce-handle", "Search ecommerce handles for common-law evidence", "/common-law/search/ecommerce/handle"))
	cmd.AddCommand(newCommonLawSearchEndpointCommand(opts, "google-text", "Search Google text results for common-law evidence", "/common-law/search/google/text"))
	cmd.AddCommand(newCommonLawSearchEndpointCommand(opts, "social-handle", "Search social handles for common-law evidence", "/common-law/search/social/handle"))
	cmd.AddCommand(newCommonLawSearchEndpointCommand(opts, "social-text", "Search social text results for common-law evidence", "/common-law/search/social/text"))
	return cmd
}

func newCommonLawSearchEndpointCommand(opts *globalOptions, use string, short string, path string) *cobra.Command {
	var data string
	var names []string
	var platform, collaborationID, collaborationSharedID string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.CommonLawSearchRequest{
					Name:                  splitStringValues(names),
					Platform:              strings.TrimSpace(platform),
					CollaborationID:       strings.TrimSpace(collaborationID),
					CollaborationSharedID: strings.TrimSpace(collaborationSharedID),
				}
				if len(req.Name) == 0 {
					return nil, fmt.Errorf("--name is required")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, path, nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringArrayVar(&names, "name", nil, "name or mark text; repeatable or comma-separated")
	cmd.Flags().StringVar(&platform, "platform", "", "source platform filter")
	cmd.Flags().StringVar(&collaborationID, "collaboration-id", "", "collaboration id")
	cmd.Flags().StringVar(&collaborationSharedID, "collaboration-shared-id", "", "collaboration shared id")
	return cmd
}

func newCommonLawMaxSimilarityCommand(opts *globalOptions) *cobra.Command {
	var data string
	var keyword string
	cmd := &cobra.Command{
		Use:   "max-similarity",
		Short: "Get max common-law similarity for a keyword",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.CommonLawMaxSimilarityRequest{
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
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/common-law/max-similarity", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&keyword, "keyword", "", "keyword to check")
	return cmd
}
