package tmc

import (
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func newPortfolioCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "Work with portfolio resources",
	}
	trademarks := &cobra.Command{
		Use:   "trademarks",
		Short: "Work with portfolio trademarks",
	}
	trademarks.AddCommand(newPortfolioTrademarksListCommand(opts))
	trademarks.AddCommand(newPortfolioTrademarkGetCommand(opts))
	trademarks.AddCommand(newPortfolioMonitoredTrademarksCommand(opts))
	trademarks.AddCommand(newPortfolioTrademarksImportCommand(opts))
	trademarks.AddCommand(newPortfolioTrademarksImportPreviewCommand(opts))
	trademarks.AddCommand(newPortfolioTrademarkUpdateCommand(opts))
	trademarks.AddCommand(newPortfolioTrademarkMetadataCommand(opts))
	trademarks.AddCommand(newPortfolioTrademarkMonitorCommand(opts))
	cmd.AddCommand(trademarks)

	groups := &cobra.Command{
		Use:   "groups",
		Short: "Work with portfolio trademark groups",
	}
	groups.AddCommand(newPortfolioTrademarkGroupsListCommand(opts))
	groups.AddCommand(newPortfolioTrademarkGroupMonitorToggleCommand(opts))
	cmd.AddCommand(groups)

	actions := &cobra.Command{
		Use:   "actions",
		Short: "Work with portfolio action lists",
	}
	actions.AddCommand(newPagedListCommand(opts, listCommandSpec{
		Use:   "office",
		Short: "List portfolio office actions",
		Path:  "/portfolio/actions/office",
		Filters: []queryFlagSpec{
			{Flag: "keyword", Param: "keyword", Description: "keyword filter"},
			{Flag: "serial", Param: "serial", Description: "legacy serial filter"},
			{Flag: "status", Param: "status", Description: "status filter"},
		},
	}))
	actions.AddCommand(newPagedListCommand(opts, listCommandSpec{
		Use:          "conflict",
		Short:        "List portfolio conflict actions",
		Path:         "/portfolio/actions/conflict",
		SortParam:    "sort_field",
		SortDirParam: "sort_dir",
		Filters: []queryFlagSpec{
			{Flag: "keyword", Param: "keyword", Description: "keyword filter"},
			{Flag: "status", Param: "status", Description: "status filter"},
			{Flag: "risk", Param: "risk", Description: "risk filter"},
			{Flag: "jurisdiction", Param: "jurisdiction", Description: "jurisdiction filter"},
			{Flag: "reviewed", Param: "reviewed", Description: "review filter"},
			{Flag: "date-from", Param: "date_from", Description: "calendar range start timestamp"},
			{Flag: "date-to", Param: "date_to", Description: "calendar range end timestamp"},
		},
	}))
	actions.AddCommand(newPagedListCommand(opts, listCommandSpec{
		Use:          "cbp",
		Short:        "List portfolio CBP recordations",
		Path:         "/portfolio/actions/cbp",
		SortParam:    "sort_field",
		SortDirParam: "sort_dir",
		Filters: []queryFlagSpec{
			{Flag: "keyword", Param: "keyword", Description: "keyword filter"},
			{Flag: "status", Param: "status", Description: "status filter"},
		},
	}))
	actions.AddCommand(newPortfolioSummaryCommand(opts, "office-summary", "Get portfolio office action summary", "/portfolio/actions/office/summary", nil))
	actions.AddCommand(newPortfolioSummaryCommand(opts, "conflict-summary", "Get portfolio conflict action summary", "/portfolio/actions/conflict/summary", nil))
	actions.AddCommand(newPortfolioSummaryCommand(opts, "cbp-summary", "Get portfolio CBP summary", "/portfolio/actions/cbp/summary", []queryFlagSpec{
		{Flag: "keyword", Param: "keyword", Description: "keyword filter"},
		{Flag: "status", Param: "status", Description: "status filter"},
	}))
	cmd.AddCommand(actions)

	activity := &cobra.Command{
		Use:   "activity",
		Short: "Work with portfolio activity",
	}
	activity.AddCommand(newPagedListCommand(opts, listCommandSpec{
		Use:   "list",
		Short: "List portfolio activity",
		Path:  "/portfolio/activity",
		Filters: []queryFlagSpec{
			{Flag: "category", Param: "category", Description: "category filter"},
			{Flag: "action", Param: "action", Description: "action filter"},
			{Flag: "keyword", Param: "keyword", Description: "keyword filter"},
			{Flag: "user", Param: "user", Description: "user filter"},
		},
	}))
	cmd.AddCommand(activity)

	cmd.AddCommand(newPortfolioSummaryCommand(opts, "counts", "Get portfolio trademark counts", "/portfolio/trademarks/counts", nil))
	cmd.AddCommand(newPortfolioSummaryCommand(opts, "monitored-summary", "Get portfolio monitored summary", "/portfolio/trademarks/monitored/summary", nil))
	return cmd
}

func newPortfolioTrademarksListCommand(opts *globalOptions) *cobra.Command {
	return newPagedListCommand(opts, listCommandSpec{
		Use:          "list",
		Short:        "List portfolio trademarks",
		Path:         "/portfolio/trademarks/search",
		SortParam:    "sort_field",
		SortDirParam: "sort_dir",
		Filters: []queryFlagSpec{
			{Flag: "keyword", Param: "keyword", Description: "search keyword"},
			{Flag: "trademark-format", Param: "format", Description: "trademark format filter"},
			{Flag: "status", Param: "status", Description: "trademark status filter"},
			{Flag: "statuses", Param: "statuses", Description: "comma-separated trademark status filters"},
			{Flag: "country", Param: "country", Description: "country code filter"},
			{Flag: "class", Param: "class", Description: "Nice class filter"},
			{Flag: "classes", Param: "classes", Description: "comma-separated Nice class filters"},
			{Flag: "deadline-days", Param: "deadline_days", Description: "upcoming deadline window in days"},
			{Flag: "owner-name", Param: "owner_name", Description: "owner filter"},
			{Flag: "attorney-name", Param: "attorney_name", Description: "attorney filter"},
			{Flag: "registration-number", Param: "registration_number", Description: "registration number filter"},
			{Flag: "monitoring", Param: "monitoring", Description: "monitoring filter"},
			{Flag: "cbp-status", Param: "cbp_status", Description: "CBP status filter"},
			{Flag: "tag-id", Param: "tag_id", Description: "single tag filter"},
			{Flag: "tag-ids", Param: "tag_ids", Description: "comma-separated tag filters"},
			{Flag: "tag-match", Param: "tag_match", Description: "tag match mode: any or all"},
		},
	})
}

func newPortfolioTrademarkGetCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <trademark-id>",
		Short: "Get a portfolio trademark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "GET", "/portfolio/trademarks/"+url.PathEscape(args[0]), nil, nil)
		},
	}
}

func newPortfolioMonitoredTrademarksCommand(opts *globalOptions) *cobra.Command {
	var page int
	var pageSize int
	var monitorType string
	var params []string
	cmd := &cobra.Command{
		Use:   "monitored",
		Short: "List monitored portfolio trademarks",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, err := parseParams(params)
			if err != nil {
				return err
			}
			if page < 1 {
				page = 1
			}
			if pageSize < 1 {
				pageSize = 20
			}
			query.Set("page", strconv.Itoa(page))
			query.Set("page_size", strconv.Itoa(pageSize))
			setQuery(query, "monitor_type", monitorType)
			return callAPIAndWrite(cmd, opts, "GET", "/portfolio/trademarks/monitored", query, nil)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "page size")
	cmd.Flags().StringVar(&monitorType, "monitor-type", "", "monitor type filter")
	cmd.Flags().StringArrayVar(&params, "param", nil, "additional query parameter key=value; repeatable")
	return cmd
}

func newPortfolioSummaryCommand(opts *globalOptions, use string, short string, path string, filters []queryFlagSpec) *cobra.Command {
	var params []string
	filterValues := make([]queryFlag, len(filters))
	for i, filter := range filters {
		param := filter.Param
		if param == "" {
			param = filter.Flag
		}
		filterValues[i] = queryFlag{
			flag:        filter.Flag,
			param:       param,
			defaultVal:  filter.Default,
			description: filter.Description,
		}
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				query, err := parseParams(params)
				if err != nil {
					return err
				}
				for _, filter := range filterValues {
					if filter.value != "" {
						query.Set(filter.param, filter.value)
					}
				}
				return executeAPIAndWrite(cmd, opts, "GET", path, query, nil)
			})
		},
	}
	cmd.Flags().StringArrayVar(&params, "param", nil, "additional query parameter key=value; repeatable")
	for i := range filterValues {
		filter := &filterValues[i]
		cmd.Flags().StringVar(&filter.value, filter.flag, filter.defaultVal, filter.description)
	}
	return cmd
}
