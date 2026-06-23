package tmc

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

func newPortfolioOfficeActionsCommand(opts *globalOptions) *cobra.Command {
	cmd := newPagedListCommand(opts, portfolioOfficeActionsListSpec("office"))
	cmd.AddCommand(newPagedListCommand(opts, portfolioOfficeActionsListSpec("list")))
	cmd.AddCommand(newPortfolioOfficeActionDeadlinesCommand(opts))
	cmd.AddCommand(newPortfolioTrademarkActionsListCommand(opts, "for-trademark <trademark-id>", "List office actions by trademark", "office-actions"))
	cmd.AddCommand(newPortfolioTrademarkActionGetCommand(opts, "get <trademark-id> <action-id>", "Get an office action", "office-actions"))
	cmd.AddCommand(newPortfolioTrademarkActionStatusCommand(opts, "status <trademark-id> <action-id>", "Update an office action status", "office-actions"))
	return cmd
}

func portfolioOfficeActionsListSpec(use string) listCommandSpec {
	return listCommandSpec{
		Use:   use,
		Short: "List portfolio office actions",
		Path:  "/portfolio/actions/office",
		Filters: []queryFlagSpec{
			{Flag: "keyword", Param: "keyword", Description: "keyword filter"},
			{Flag: "serial", Param: "serial", Description: "legacy serial filter"},
			{Flag: "status", Param: "status", Description: "status filter"},
		},
	}
}

func newPortfolioConflictActionsCommand(opts *globalOptions) *cobra.Command {
	cmd := newPagedListCommand(opts, portfolioConflictActionsListSpec("conflict"))
	cmd.AddCommand(newPagedListCommand(opts, portfolioConflictActionsListSpec("list")))
	cmd.AddCommand(newPortfolioConflictActionGroupsCommand(opts))
	cmd.AddCommand(newPortfolioTrademarkActionsListCommand(opts, "for-trademark <trademark-id>", "List conflict actions by trademark", "conflict-actions"))
	cmd.AddCommand(newPortfolioTrademarkActionGetCommand(opts, "get <trademark-id> <action-id>", "Get a conflict action", "conflict-actions"))
	cmd.AddCommand(newPortfolioTrademarkActionStatusCommand(opts, "status <trademark-id> <action-id>", "Update a conflict action status", "conflict-actions"))
	return cmd
}

func portfolioConflictActionsListSpec(use string) listCommandSpec {
	return listCommandSpec{
		Use:          use,
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
	}
}

func newPortfolioCBPActionsCommand(opts *globalOptions) *cobra.Command {
	cmd := newPagedListCommand(opts, portfolioCBPActionsListSpec("cbp"))
	cmd.AddCommand(newPagedListCommand(opts, portfolioCBPActionsListSpec("list")))
	cmd.AddCommand(newPortfolioCBPServiceRequestsCommand(opts))
	cmd.AddCommand(newPortfolioCBPSubmitServiceRequestCommand(opts))
	return cmd
}

func portfolioCBPActionsListSpec(use string) listCommandSpec {
	return listCommandSpec{
		Use:          use,
		Short:        "List portfolio CBP recordations",
		Path:         "/portfolio/actions/cbp",
		SortParam:    "sort_field",
		SortDirParam: "sort_dir",
		Filters: []queryFlagSpec{
			{Flag: "keyword", Param: "keyword", Description: "keyword filter"},
			{Flag: "status", Param: "status", Description: "status filter"},
		},
	}
}

func newPortfolioOfficeActionDeadlinesCommand(opts *globalOptions) *cobra.Command {
	var limit int
	var params []string
	cmd := &cobra.Command{
		Use:   "deadlines",
		Short: "List upcoming office action deadlines",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, err := parseParams(params)
			if err != nil {
				return err
			}
			if limit > 0 {
				query.Set("limit", strconv.Itoa(limit))
			}
			return callAPIAndWrite(cmd, opts, "GET", "/portfolio/actions/office/deadlines", query, nil)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum deadlines to return")
	cmd.Flags().StringArrayVar(&params, "param", nil, "additional query parameter key=value; repeatable")
	return cmd
}

func newPortfolioConflictActionGroupsCommand(opts *globalOptions) *cobra.Command {
	var page int
	var pageSize int
	var params []string
	var keyword, status, risk, jurisdiction, reviewed, groupBy string
	var sortField, sortDir, dateFrom, dateTo string
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "List grouped conflict actions",
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
			setQuery(query, "keyword", keyword)
			setQuery(query, "status", status)
			setQuery(query, "risk", risk)
			setQuery(query, "jurisdiction", jurisdiction)
			setQuery(query, "reviewed", reviewed)
			setQuery(query, "group_by", groupBy)
			setQuery(query, "sort_field", sortField)
			setQuery(query, "sort_dir", sortDir)
			setQuery(query, "date_from", dateFrom)
			setQuery(query, "date_to", dateTo)
			return callAPIAndWrite(cmd, opts, "GET", "/portfolio/actions/conflict/groups", query, nil)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "page size")
	cmd.Flags().StringArrayVar(&params, "param", nil, "additional query parameter key=value; repeatable")
	cmd.Flags().StringVar(&keyword, "keyword", "", "keyword filter")
	cmd.Flags().StringVar(&status, "status", "", "status filter")
	cmd.Flags().StringVar(&risk, "risk", "", "risk filter")
	cmd.Flags().StringVar(&jurisdiction, "jurisdiction", "", "jurisdiction filter")
	cmd.Flags().StringVar(&reviewed, "reviewed", "", "review filter")
	cmd.Flags().StringVar(&groupBy, "group-by", "", "grouping mode")
	cmd.Flags().StringVar(&sortField, "sort", "", "sort field")
	cmd.Flags().StringVar(&sortDir, "sort-dir", "", "sort direction")
	cmd.Flags().StringVar(&dateFrom, "date-from", "", "calendar range start timestamp")
	cmd.Flags().StringVar(&dateTo, "date-to", "", "calendar range end timestamp")
	return cmd
}

func newPortfolioTrademarkActionsListCommand(opts *globalOptions, use string, short string, actionPath string) *cobra.Command {
	var page int
	var pageSize int
	var params []string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
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
			path := portfolioTrademarkActionPath(args[0], actionPath)
			return callAPIAndWrite(cmd, opts, "GET", path, query, nil)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "page size")
	cmd.Flags().StringArrayVar(&params, "param", nil, "additional query parameter key=value; repeatable")
	return cmd
}

func newPortfolioTrademarkActionGetCommand(opts *globalOptions, use string, short string, actionPath string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := portfolioTrademarkActionPath(args[0], actionPath) + "/" + url.PathEscape(args[1])
			return callAPIAndWrite(cmd, opts, "GET", path, nil, nil)
		},
	}
}

func newPortfolioTrademarkActionStatusCommand(opts *globalOptions, use string, short string, actionPath string) *cobra.Command {
	var data string
	var status int
	var note string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				if !cmd.Flags().Changed("status") {
					return nil, fmt.Errorf("--status is required")
				}
				return openapi.PortfolioActionStatusUpdateRequest{
					Status: status,
					Note:   strings.TrimSpace(note),
				}, nil
			})
			if err != nil {
				return err
			}
			path := portfolioTrademarkActionPath(args[0], actionPath) + "/" + url.PathEscape(args[1]) + "/status"
			return callAPIAndWrite(cmd, opts, "PUT", path, nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().IntVar(&status, "status", 0, "action status integer")
	cmd.Flags().StringVar(&note, "note", "", "status note")
	return cmd
}

func newPortfolioCBPServiceRequestsCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "service-requests",
		Short: "List CBP recordation service requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "GET", "/portfolio/actions/cbp/service-requests", nil, nil)
		},
	}
}

func newPortfolioCBPSubmitServiceRequestCommand(opts *globalOptions) *cobra.Command {
	var data string
	var requestType, trademarkID, serialNumber, registrationNumber, recordationNumber string
	var recordationType, markName, contactName, contactEmail, notes string
	var portsOfEntry []string
	cmd := &cobra.Command{
		Use:     "submit",
		Aliases: []string{"submit-service-request"},
		Short:   "Submit a CBP recordation service request",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.PortfolioCBPServiceRequestCreateRequest{
					RequestType:        strings.TrimSpace(requestType),
					TrademarkID:        strings.TrimSpace(trademarkID),
					SerialNumber:       strings.TrimSpace(serialNumber),
					RegistrationNumber: strings.TrimSpace(registrationNumber),
					RecordationNumber:  strings.TrimSpace(recordationNumber),
					RecordationType:    strings.TrimSpace(recordationType),
					MarkName:           strings.TrimSpace(markName),
					ContactName:        strings.TrimSpace(contactName),
					ContactEmail:       strings.TrimSpace(contactEmail),
					PortsOfEntry:       splitStringValues(portsOfEntry),
					Notes:              strings.TrimSpace(notes),
				}
				if req.RequestType == "" {
					return nil, fmt.Errorf("--request-type is required")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, "POST", "/portfolio/actions/cbp/service-requests", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&requestType, "request-type", "", "service request type")
	cmd.Flags().StringVar(&trademarkID, "trademark-id", "", "portfolio trademark ID")
	cmd.Flags().StringVar(&serialNumber, "serial-number", "", "trademark serial number")
	cmd.Flags().StringVar(&registrationNumber, "registration-number", "", "trademark registration number")
	cmd.Flags().StringVar(&recordationNumber, "recordation-number", "", "CBP recordation number")
	cmd.Flags().StringVar(&recordationType, "recordation-type", "", "CBP recordation type")
	cmd.Flags().StringVar(&markName, "mark-name", "", "mark name")
	cmd.Flags().StringVar(&contactName, "contact-name", "", "contact name")
	cmd.Flags().StringVar(&contactEmail, "contact-email", "", "contact email")
	cmd.Flags().StringArrayVar(&portsOfEntry, "port-of-entry", nil, "port of entry; repeatable or comma-separated")
	cmd.Flags().StringVar(&notes, "notes", "", "request notes")
	return cmd
}

func portfolioTrademarkActionPath(trademarkID string, actionPath string) string {
	return "/portfolio/trademarks/" + url.PathEscape(trademarkID) + "/" + actionPath
}
