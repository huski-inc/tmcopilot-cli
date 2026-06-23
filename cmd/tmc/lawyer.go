package tmc

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

type lawyerTrademarkSortOptions struct {
	FilingAt     string
	Index        string
	LawsuitCount string
	Mark         string
	SerialNumber string
	Status       string
}

type lawyerLawFirmSortOptions struct {
	Name           string
	Rank           string
	TrademarkCount string
	LawsuitCount   string
	LawyerCount    string
}

func newLawyersCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lawyers",
		Aliases: []string{"lawyer", "attorneys", "attorney"},
		Short:   "Search and inspect trademark lawyers",
	}
	cmd.AddCommand(newLawyerSearchTopCommand(opts))
	cmd.AddCommand(newLawyerRankingTopCommand(opts))
	cmd.AddCommand(newLawyerContactTopCommand(opts))
	cmd.AddCommand(newLawyerGetCommand(opts))
	cmd.AddCommand(newLawyerTrademarksCommand(opts))
	cmd.AddCommand(newLawyerLawFirmsCommand(opts))
	cmd.AddCommand(newWideTableLawsuitListCommand(opts, "lawsuits <graph-id>", "List lawyer lawsuits", "/trademark/wide-table/lawyers/%s/lawsuits"))
	return cmd
}

func newLawyerSearchCommand(opts *globalOptions) *cobra.Command {
	return newLawyerSearchCommandFor(opts, "lawyers", []string{"attorneys", "attorney", "lawyer"})
}

func newLawyerSearchTopCommand(opts *globalOptions) *cobra.Command {
	return newLawyerSearchCommandFor(opts, "search", []string{"find"})
}

func newLawyerSearchCommandFor(opts *globalOptions, use string, aliases []string) *cobra.Command {
	var name, city, state, zipCode, emailName, emailDomain, emailAddress string
	var page int
	var limit int
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Search trademark lawyers and attorneys",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			setQuery(query, "name", name)
			setQuery(query, "city", city)
			setQuery(query, "state", state)
			setQuery(query, "zip_code", zipCode)
			setQuery(query, "email_name", emailName)
			setQuery(query, "email_domain", emailDomain)
			setQuery(query, "email_address", emailAddress)
			if page > 0 {
				query.Set("page", strconv.Itoa(page))
			}
			if limit > 0 {
				query.Set("limit", strconv.Itoa(limit))
			}
			if len(query) == 0 {
				return fmt.Errorf("at least one lawyer search flag is required")
			}
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/trademark/lawyer/search", query, nil)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "lawyer name to search")
	cmd.Flags().StringVar(&city, "city", "", "correspondent city filter")
	cmd.Flags().StringVar(&state, "state", "", "correspondent state filter")
	cmd.Flags().StringVar(&zipCode, "zip-code", "", "correspondent ZIP code filter")
	cmd.Flags().StringVar(&emailName, "email-name", "", "email local-part filter")
	cmd.Flags().StringVar(&emailDomain, "email-domain", "", "email domain filter")
	cmd.Flags().StringVar(&emailAddress, "email-address", "", "exact email address filter")
	cmd.Flags().IntVar(&page, "page", 0, "page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "items per page")
	return cmd
}

func newLawyerRankingCommand(opts *globalOptions) *cobra.Command {
	return newLawyerRankingCommandFor(opts, "lawyer-ranking", []string{"attorney-ranking"})
}

func newLawyerRankingTopCommand(opts *globalOptions) *cobra.Command {
	return newLawyerRankingCommandFor(opts, "ranking", nil)
}

func newLawyerRankingCommandFor(opts *globalOptions, use string, aliases []string) *cobra.Command {
	var rankingType string
	var limit int
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Get trademark lawyer ranking",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			setQuery(query, "type", rankingType)
			if limit > 0 {
				query.Set("limit", strconv.Itoa(limit))
			}
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/trademark/lawyer/ranking", query, nil)
		},
	}
	cmd.Flags().StringVar(&rankingType, "type", "", "ranking type")
	cmd.Flags().IntVar(&limit, "limit", 0, "max number of results")
	return cmd
}

func newLawyerContactCommand(opts *globalOptions) *cobra.Command {
	return newLawyerContactCommandFor(opts, "lawyer-contact", []string{"attorney-contact"})
}

func newLawyerContactTopCommand(opts *globalOptions) *cobra.Command {
	return newLawyerContactCommandFor(opts, "contact", nil)
}

func newLawyerContactCommandFor(opts *globalOptions, use string, aliases []string) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Get lawyer contact information",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required")
			}
			query := url.Values{}
			query.Set("name", name)
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/trademark/lawyer/contact", query, nil)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "lawyer name exact match")
	return cmd
}

func newLawyerGetCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:     "get <graph-id>",
		Aliases: []string{"detail"},
		Short:   "Get lawyer wide-table info",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/trademark/wide-table/lawyers/"+url.PathEscape(args[0]), nil, nil)
		},
	}
}

func newLawyerTrademarksCommand(opts *globalOptions) *cobra.Command {
	var data string
	var limit, page, status int
	var sorts lawyerTrademarkSortOptions
	cmd := &cobra.Command{
		Use:   "trademarks <graph-id>",
		Short: "List lawyer trademarks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				return buildLawyerTrademarkListRequest(cmd, limit, page, status, sorts), nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/trademark/wide-table/lawyers/"+url.PathEscape(args[0])+"/trademarks", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().IntVar(&limit, "limit", 0, "result limit")
	cmd.Flags().IntVar(&page, "page", 0, "result page")
	cmd.Flags().IntVar(&status, "status", 0, "trademark status")
	addLawyerTrademarkSortFlags(cmd, &sorts)
	return cmd
}

func newLawyerLawFirmsCommand(opts *globalOptions) *cobra.Command {
	var data string
	var limit, page int
	var sorts lawyerLawFirmSortOptions
	cmd := &cobra.Command{
		Use:     "law-firms <graph-id>",
		Aliases: []string{"lawfirms", "firms"},
		Short:   "List lawyer law firms",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				return buildLawyerLawFirmListRequest(limit, page, sorts), nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/trademark/wide-table/lawyers/"+url.PathEscape(args[0])+"/law-firms", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().IntVar(&limit, "limit", 0, "result limit")
	cmd.Flags().IntVar(&page, "page", 0, "result page")
	addLawyerLawFirmSortFlags(cmd, &sorts)
	return cmd
}

func buildLawyerTrademarkListRequest(cmd *cobra.Command, limit int, page int, status int, sorts lawyerTrademarkSortOptions) openapi.WideTableTrademarkListRequest {
	return openapi.WideTableTrademarkListRequest{
		Limit:            limit,
		Page:             page,
		Status:           changedIntPtr(cmd, "status", status),
		SortFilingAt:     strings.TrimSpace(sorts.FilingAt),
		SortIndex:        strings.TrimSpace(sorts.Index),
		SortLawsuitCount: strings.TrimSpace(sorts.LawsuitCount),
		SortMark:         strings.TrimSpace(sorts.Mark),
		SortSerialNumber: strings.TrimSpace(sorts.SerialNumber),
		SortStatus:       strings.TrimSpace(sorts.Status),
	}
}

func buildLawyerLawFirmListRequest(limit int, page int, sorts lawyerLawFirmSortOptions) openapi.WideTableLawFirmListRequest {
	return openapi.WideTableLawFirmListRequest{
		Limit:              limit,
		Page:               page,
		SortName:           strings.TrimSpace(sorts.Name),
		SortRank:           strings.TrimSpace(sorts.Rank),
		SortTrademarkCount: strings.TrimSpace(sorts.TrademarkCount),
		SortLawsuitCount:   strings.TrimSpace(sorts.LawsuitCount),
		SortLawyerCount:    strings.TrimSpace(sorts.LawyerCount),
	}
}

func addLawyerTrademarkSortFlags(cmd *cobra.Command, sorts *lawyerTrademarkSortOptions) {
	cmd.Flags().StringVar(&sorts.FilingAt, "sort-filing-at", "", "sort filing date: asc or desc")
	cmd.Flags().StringVar(&sorts.Index, "sort-index", "", "sort index: asc or desc")
	cmd.Flags().StringVar(&sorts.LawsuitCount, "sort-lawsuit-count", "", "sort lawsuit count: asc or desc")
	cmd.Flags().StringVar(&sorts.Mark, "sort-mark", "", "sort mark: asc or desc")
	cmd.Flags().StringVar(&sorts.SerialNumber, "sort-serial-number", "", "sort serial number: asc or desc")
	cmd.Flags().StringVar(&sorts.Status, "sort-status", "", "sort status: asc or desc")
}

func addLawyerLawFirmSortFlags(cmd *cobra.Command, sorts *lawyerLawFirmSortOptions) {
	cmd.Flags().StringVar(&sorts.Name, "sort-name", "", "sort name: asc or desc")
	cmd.Flags().StringVar(&sorts.Rank, "sort-rank", "", "sort rank: asc or desc")
	cmd.Flags().StringVar(&sorts.TrademarkCount, "sort-trademark-count", "", "sort trademark count: asc or desc")
	cmd.Flags().StringVar(&sorts.LawsuitCount, "sort-lawsuit-count", "", "sort lawsuit count: asc or desc")
	cmd.Flags().StringVar(&sorts.LawyerCount, "sort-lawyer-count", "", "sort lawyer count: asc or desc")
}
