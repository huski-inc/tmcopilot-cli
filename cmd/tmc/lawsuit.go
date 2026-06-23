package tmc

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

type lawsuitSortOptions struct {
	CaseAt                string
	CaseName              string
	CaseNumberCode        string
	Index                 string
	LawFirmCount          string
	LawsuitDefendantCount string
	LawsuitPlaintiffCount string
	LawyerCount           string
}

func newLawsuitCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lawsuits",
		Aliases: []string{"lawsuit", "litigation"},
		Short:   "Search and fetch trademark lawsuits",
	}
	cmd.AddCommand(newLawsuitSearchCommand(opts))
	cmd.AddCommand(newLawsuitGetCommand(opts))
	cmd.AddCommand(newWideTableLawsuitListCommand(opts, "brand-owner <graph-id>", "List brand owner lawsuits", "/trademark/wide-table/brand-owners/%s/lawsuits"))
	cmd.AddCommand(newWideTableLawsuitListCommand(opts, "lawyer <graph-id>", "List lawyer lawsuits", "/trademark/wide-table/lawyers/%s/lawsuits"))
	return cmd
}

func newLawsuitSearchCommand(opts *globalOptions) *cobra.Command {
	return newLawsuitSearchCommandFor(opts, "search", nil)
}

func newLawsuitSearchLegacyCommand(opts *globalOptions) *cobra.Command {
	return newLawsuitSearchCommandFor(opts, "lawsuits", nil)
}

func newLawsuitSearchCommandFor(opts *globalOptions, use string, aliases []string) *cobra.Command {
	var data string
	var caseAt, caseClosedAt, caseNames, caseNumberCodes []string
	var parties, plaintiffs, defendants, lawyers, lawFirms, trademarks []string
	var usageID string
	var limit, page int
	var sorts lawsuitSortOptions
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Search lawsuits",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.LawsuitSearchRequest{
					CaseAt:                    splitStringValues(caseAt),
					CaseClosedAt:              splitStringValues(caseClosedAt),
					CaseName:                  splitStringValues(caseNames),
					CaseNumberCode:            splitStringValues(caseNumberCodes),
					PartyName:                 splitStringValues(parties),
					PlaintiffName:             splitStringValues(plaintiffs),
					DefendantName:             splitStringValues(defendants),
					LawyerName:                splitStringValues(lawyers),
					LawFirmName:               splitStringValues(lawFirms),
					Trademark:                 splitStringValues(trademarks),
					UsageIdempotencyKey:       strings.TrimSpace(usageID),
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
				if req.Empty() {
					return nil, fmt.Errorf("lawsuits search requires --data or at least one search/filter/sort flag")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/trademark/wide-table/lawsuits", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringArrayVar(&caseAt, "case-at", nil, "case filing date or timestamp; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&caseClosedAt, "case-closed-at", nil, "case closed date or timestamp; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&caseNames, "case-name", nil, "case name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&caseNumberCodes, "case-number-code", nil, "case number code; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&parties, "party", nil, "party name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&plaintiffs, "plaintiff", nil, "plaintiff name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&defendants, "defendant", nil, "defendant name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&lawyers, "lawyer", nil, "lawyer name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&lawFirms, "law-firm", nil, "law firm name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&trademarks, "trademark", nil, "trademark text or serial; repeatable or comma-separated")
	cmd.Flags().StringVar(&usageID, "usage-idempotency-key", "", "usage idempotency key")
	cmd.Flags().IntVar(&limit, "limit", 0, "result limit")
	cmd.Flags().IntVar(&page, "page", 0, "result page")
	addLawsuitSortFlags(cmd, &sorts)
	return cmd
}

func newLawsuitGetCommand(opts *globalOptions) *cobra.Command {
	return newLawsuitGetCommandFor(opts, "get <case-number>", []string{"case", "detail"})
}

func newLawsuitGetLegacyCommand(opts *globalOptions) *cobra.Command {
	return newLawsuitGetCommandFor(opts, "lawsuit <case-number>", []string{"lawsuit-case"})
}

func newLawsuitGetCommandFor(opts *globalOptions, use string, aliases []string) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Get a lawsuit by case number",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/trademark/wide-table/lawsuits/"+url.PathEscape(args[0]), nil, nil)
		},
	}
}

func addLawsuitSortFlags(cmd *cobra.Command, sorts *lawsuitSortOptions) {
	cmd.Flags().StringVar(&sorts.CaseAt, "sort-case-at", "", "sort case filing date: asc or desc")
	cmd.Flags().StringVar(&sorts.CaseName, "sort-case-name", "", "sort case name: asc or desc")
	cmd.Flags().StringVar(&sorts.CaseNumberCode, "sort-case-number-code", "", "sort case number code: asc or desc")
	cmd.Flags().StringVar(&sorts.Index, "sort-index", "", "sort index: asc or desc")
	cmd.Flags().StringVar(&sorts.LawFirmCount, "sort-law-firm-count", "", "sort law firm count: asc or desc")
	cmd.Flags().StringVar(&sorts.LawsuitDefendantCount, "sort-lawsuit-defendant-count", "", "sort lawsuit defendant count: asc or desc")
	cmd.Flags().StringVar(&sorts.LawsuitPlaintiffCount, "sort-lawsuit-plaintiff-count", "", "sort lawsuit plaintiff count: asc or desc")
	cmd.Flags().StringVar(&sorts.LawyerCount, "sort-lawyer-count", "", "sort lawyer count: asc or desc")
}
