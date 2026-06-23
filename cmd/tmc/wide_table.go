package tmc

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

func newWideTableLawsuitListCommand(opts *globalOptions, use string, short string, pathFormat string) *cobra.Command {
	var data string
	var limit, page int
	var sorts lawsuitSortOptions
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				return buildLawsuitListRequest(limit, page, sorts), nil
			})
			if err != nil {
				return err
			}
			path := fmt.Sprintf(pathFormat, url.PathEscape(args[0]))
			return callAPIAndWrite(cmd, opts, http.MethodPost, path, nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().IntVar(&limit, "limit", 0, "result limit")
	cmd.Flags().IntVar(&page, "page", 0, "result page")
	addLawsuitSortFlags(cmd, &sorts)
	return cmd
}

func buildLawsuitListRequest(limit int, page int, sorts lawsuitSortOptions) openapi.LawsuitListRequest {
	return openapi.LawsuitListRequest{
		Limit:                     limit,
		Page:                      page,
		SortCaseAt:                strings.TrimSpace(sorts.CaseAt),
		SortCaseName:              strings.TrimSpace(sorts.CaseName),
		SortCaseNumberCode:        strings.TrimSpace(sorts.CaseNumberCode),
		SortIndex:                 strings.TrimSpace(sorts.Index),
		SortLawFirmCount:          strings.TrimSpace(sorts.LawFirmCount),
		SortLawsuitDefendantCount: strings.TrimSpace(sorts.LawsuitDefendantCount),
		SortLawsuitPlaintiffCount: strings.TrimSpace(sorts.LawsuitPlaintiffCount),
		SortLawyerCount:           strings.TrimSpace(sorts.LawyerCount),
	}
}
