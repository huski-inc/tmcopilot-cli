package tmc

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

func newPortfolioTrademarksImportCommand(opts *globalOptions) *cobra.Command {
	return newPortfolioTrademarksImportLikeCommand(opts, "import", "Import trademarks by owner, organization, or lawyer names", "/portfolio/trademarks/import")
}

func newPortfolioTrademarksImportPreviewCommand(opts *globalOptions) *cobra.Command {
	return newPortfolioTrademarksImportLikeCommand(opts, "import-preview", "Preview a portfolio trademark import", "/portfolio/trademarks/import/preview")
}

func newPortfolioTrademarksImportLikeCommand(opts *globalOptions, use string, short string, path string) *cobra.Command {
	var data string
	var country string
	var ownerNames, organizationNames, lawyerNames []string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.PortfolioImportTrademarksRequest{
					Country:           strings.TrimSpace(country),
					OwnerNames:        splitStringValues(ownerNames),
					OrganizationNames: splitStringValues(organizationNames),
					LawyerNames:       splitStringValues(lawyerNames),
				}
				if !req.HasImportCriteria() {
					return nil, fmt.Errorf("at least one of --owner-name, --organization-name, or --lawyer-name is required")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, "POST", path, nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&country, "country", "", "country code")
	cmd.Flags().StringArrayVar(&ownerNames, "owner-name", nil, "owner name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&organizationNames, "organization-name", nil, "organization name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&lawyerNames, "lawyer-name", nil, "lawyer name; repeatable or comma-separated")
	return cmd
}

func newPortfolioTrademarkUpdateCommand(opts *globalOptions) *cobra.Command {
	var data string
	var text, country, attorneyDocketNumber string
	var format, status int
	cmd := &cobra.Command{
		Use:   "update <trademark-id>",
		Short: "Update a portfolio trademark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.PortfolioUpdateTrademarkRequest{
					Text:                 strings.TrimSpace(text),
					Country:              strings.TrimSpace(country),
					AttorneyDocketNumber: strings.TrimSpace(attorneyDocketNumber),
					Format:               changedIntPtr(cmd, "trademark-format", format),
					Status:               changedIntPtr(cmd, "status", status),
				}
				if req.Empty() {
					return nil, fmt.Errorf("update requires --data or at least one update flag")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, "PUT", "/portfolio/trademarks/"+url.PathEscape(args[0]), nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&text, "text", "", "mark text")
	cmd.Flags().StringVar(&country, "country", "", "country code")
	cmd.Flags().StringVar(&attorneyDocketNumber, "attorney-docket-number", "", "attorney docket number")
	cmd.Flags().IntVar(&format, "trademark-format", 0, "trademark format integer")
	cmd.Flags().IntVar(&status, "status", 0, "trademark status integer")
	return cmd
}

func newPortfolioTrademarkMetadataCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "Work with manual portfolio trademark metadata",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "get <trademark-id>",
		Short: "Get manual portfolio trademark metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "GET", "/portfolio/trademarks/"+url.PathEscape(args[0])+"/metadata", nil, nil)
		},
	})
	cmd.AddCommand(newPortfolioTrademarkMetadataUpdateCommand(opts))
	return cmd
}

func newPortfolioTrademarkMetadataUpdateCommand(opts *globalOptions) *cobra.Command {
	var data string
	var attorneyName, cbpStatus, customReminderDate, expiryDate, filingDate string
	var goodsServices, madridIRN, markImageURL, nextEventDate, nextEventType string
	var ownerName, registrationDate, registrationNumber string
	var niceClasses, reminderIntervals []string
	cmd := &cobra.Command{
		Use:   "update <trademark-id>",
		Short: "Update manual portfolio trademark metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				classes, err := splitIntValues(niceClasses, "nice_classes")
				if err != nil {
					return nil, err
				}
				req := openapi.PortfolioTrademarkMetadataRequest{
					AttorneyName:       strings.TrimSpace(attorneyName),
					CBPStatus:          strings.TrimSpace(cbpStatus),
					CustomReminderDate: strings.TrimSpace(customReminderDate),
					ExpiryDate:         strings.TrimSpace(expiryDate),
					FilingDate:         strings.TrimSpace(filingDate),
					GoodsServices:      strings.TrimSpace(goodsServices),
					MadridIRN:          strings.TrimSpace(madridIRN),
					MarkImageURL:       strings.TrimSpace(markImageURL),
					NextEventDate:      strings.TrimSpace(nextEventDate),
					NextEventType:      strings.TrimSpace(nextEventType),
					NiceClasses:        classes,
					OwnerName:          strings.TrimSpace(ownerName),
					RegistrationDate:   strings.TrimSpace(registrationDate),
					RegistrationNumber: strings.TrimSpace(registrationNumber),
					ReminderIntervals:  splitStringValues(reminderIntervals),
				}
				if req.Empty() {
					return nil, fmt.Errorf("metadata update requires --data or at least one metadata flag")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, "PUT", "/portfolio/trademarks/"+url.PathEscape(args[0])+"/metadata", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&attorneyName, "attorney-name", "", "attorney name")
	cmd.Flags().StringVar(&cbpStatus, "cbp-status", "", "CBP status")
	cmd.Flags().StringVar(&customReminderDate, "custom-reminder-date", "", "custom reminder date")
	cmd.Flags().StringVar(&expiryDate, "expiry-date", "", "expiry date")
	cmd.Flags().StringVar(&filingDate, "filing-date", "", "filing date")
	cmd.Flags().StringVar(&goodsServices, "goods-services", "", "goods/services text")
	cmd.Flags().StringVar(&madridIRN, "madrid-irn", "", "Madrid IRN")
	cmd.Flags().StringVar(&markImageURL, "mark-image-url", "", "mark image URL")
	cmd.Flags().StringVar(&nextEventDate, "next-event-date", "", "next event date")
	cmd.Flags().StringVar(&nextEventType, "next-event-type", "", "next event type")
	cmd.Flags().StringArrayVar(&niceClasses, "nice-class", nil, "Nice class integer; repeatable or comma-separated")
	cmd.Flags().StringVar(&ownerName, "owner-name", "", "owner name")
	cmd.Flags().StringVar(&registrationDate, "registration-date", "", "registration date")
	cmd.Flags().StringVar(&registrationNumber, "registration-number", "", "registration number")
	cmd.Flags().StringArrayVar(&reminderIntervals, "reminder-interval", nil, "reminder interval; repeatable or comma-separated")
	return cmd
}

func newPortfolioTrademarkMonitorCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Update portfolio trademark monitor settings",
	}
	cmd.AddCommand(newPortfolioTrademarkMonitorUpdateCommand(opts))
	cmd.AddCommand(newPortfolioTrademarkMonitorBatchUpdateCommand(opts))
	cmd.AddCommand(newPortfolioTrademarkMonitorBatchToggleCommand(opts))
	return cmd
}

func newPortfolioTrademarkMonitorUpdateCommand(opts *globalOptions) *cobra.Command {
	var data string
	flags := &portfolioMonitorConfigFlags{}
	cmd := &cobra.Command{
		Use:   "update <trademark-id>",
		Short: "Update one trademark monitor config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				config := buildPortfolioMonitorConfig(cmd, flags)
				if config.Empty() {
					return nil, fmt.Errorf("monitor update requires --data or at least one monitor config flag")
				}
				return openapi.PortfolioUpdateMonitorConfigRequest{Config: config}, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, "PUT", "/portfolio/trademarks/"+url.PathEscape(args[0])+"/monitor", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	addPortfolioMonitorConfigFlags(cmd, flags)
	return cmd
}

func newPortfolioTrademarkMonitorBatchUpdateCommand(opts *globalOptions) *cobra.Command {
	var data string
	var trademarkIDs []string
	flags := &portfolioMonitorConfigFlags{}
	cmd := &cobra.Command{
		Use:   "batch-update",
		Short: "Batch update trademark monitor config",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				ids := splitStringValues(trademarkIDs)
				if len(ids) == 0 {
					return nil, fmt.Errorf("--trademark-id is required")
				}
				config := buildPortfolioMonitorConfig(cmd, flags)
				if config.Empty() {
					return nil, fmt.Errorf("monitor batch-update requires --data or at least one monitor config flag")
				}
				return openapi.PortfolioBatchUpdateMonitorConfigRequest{TrademarkIDs: ids, Config: config}, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, "PUT", "/portfolio/trademark-monitor", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringArrayVar(&trademarkIDs, "trademark-id", nil, "trademark ID; repeatable or comma-separated")
	addPortfolioMonitorConfigFlags(cmd, flags)
	return cmd
}

func newPortfolioTrademarkMonitorBatchToggleCommand(opts *globalOptions) *cobra.Command {
	var data string
	var trademarkIDs []string
	var monitorType, conflictMode string
	var enable bool
	cmd := &cobra.Command{
		Use:   "batch-toggle",
		Short: "Batch toggle a trademark monitor type",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				ids := splitStringValues(trademarkIDs)
				if len(ids) == 0 {
					return nil, fmt.Errorf("--trademark-id is required")
				}
				req := openapi.PortfolioBatchToggleMonitorConfigRequest{
					TrademarkIDs: ids,
					MonitorType:  strings.TrimSpace(monitorType),
					Enable:       changedBoolPtr(cmd, "enable", enable),
					ConflictMode: strings.TrimSpace(conflictMode),
				}
				if req.MonitorType == "" {
					return nil, fmt.Errorf("--monitor-type is required")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, "PUT", "/portfolio/trademark-monitor/toggle", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringArrayVar(&trademarkIDs, "trademark-id", nil, "trademark ID; repeatable or comma-separated")
	cmd.Flags().StringVar(&monitorType, "monitor-type", "", "monitor type")
	cmd.Flags().BoolVar(&enable, "enable", false, "enable or disable the monitor type")
	cmd.Flags().StringVar(&conflictMode, "conflict-mode", "", "conflict monitor mode")
	return cmd
}

func newPortfolioTrademarkGroupsListCommand(opts *globalOptions) *cobra.Command {
	var page int
	var pageSize int
	var params []string
	var keyword, country, status, statuses, markComposition, markCompositions, groupID string
	var sortField, sortDir string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List portfolio trademark groups",
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
			setQuery(query, "country", country)
			setQuery(query, "status", status)
			setQuery(query, "statuses", statuses)
			setQuery(query, "mark_composition", markComposition)
			setQuery(query, "mark_compositions", markCompositions)
			setQuery(query, "group_id", groupID)
			setQuery(query, "sort_field", sortField)
			setQuery(query, "sort_dir", sortDir)
			return callAPIAndWrite(cmd, opts, "GET", "/portfolio/trademark-groups", query, nil)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "page size")
	cmd.Flags().StringArrayVar(&params, "param", nil, "additional query parameter key=value; repeatable")
	cmd.Flags().StringVar(&keyword, "keyword", "", "keyword filter")
	cmd.Flags().StringVar(&country, "country", "", "country filter")
	cmd.Flags().StringVar(&status, "status", "", "single trademark status filter")
	cmd.Flags().StringVar(&statuses, "statuses", "", "comma-separated trademark status filters")
	cmd.Flags().StringVar(&markComposition, "mark-composition", "", "mark composition filter")
	cmd.Flags().StringVar(&markCompositions, "mark-compositions", "", "comma-separated mark composition filters")
	cmd.Flags().StringVar(&groupID, "group-id", "", "specific trademark group ID")
	cmd.Flags().StringVar(&sortField, "sort", "", "sort field")
	cmd.Flags().StringVar(&sortDir, "sort-dir", "", "sort direction")
	return cmd
}

func newPortfolioTrademarkGroupMonitorToggleCommand(opts *globalOptions) *cobra.Command {
	var data string
	var monitorType, conflictMode string
	var enable bool
	cmd := &cobra.Command{
		Use:   "monitor-toggle <group-id>",
		Short: "Toggle a trademark group monitor type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.PortfolioGroupToggleMonitorConfigRequest{
					MonitorType:  strings.TrimSpace(monitorType),
					Enable:       changedBoolPtr(cmd, "enable", enable),
					ConflictMode: strings.TrimSpace(conflictMode),
				}
				if req.MonitorType == "" {
					return nil, fmt.Errorf("--monitor-type is required")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, "PUT", "/portfolio/trademark-groups/"+url.PathEscape(args[0])+"/monitor/toggle", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&monitorType, "monitor-type", "", "monitor type")
	cmd.Flags().BoolVar(&enable, "enable", false, "enable or disable the monitor type")
	cmd.Flags().StringVar(&conflictMode, "conflict-mode", "", "conflict monitor mode")
	return cmd
}

type portfolioMonitorConfigFlags struct {
	cbpActionEnable            bool
	conflictActionEnable       bool
	conflictImageEnable        bool
	conflictNotifyAssigned     bool
	conflictNotifyClient       bool
	conflictNotifyMe           bool
	conflictTextEnable         bool
	officeActionEnable         bool
	officeActionNotifyAssigned bool
	officeActionNotifyClient   bool
	officeActionNotifyMe       bool
}

func addPortfolioMonitorConfigFlags(cmd *cobra.Command, flags *portfolioMonitorConfigFlags) {
	cmd.Flags().BoolVar(&flags.cbpActionEnable, "cbp-action-enable", false, "enable CBP action monitoring")
	cmd.Flags().BoolVar(&flags.conflictActionEnable, "conflict-action-enable", false, "enable conflict action monitoring")
	cmd.Flags().BoolVar(&flags.conflictImageEnable, "conflict-image-enable", false, "enable conflict image monitoring")
	cmd.Flags().BoolVar(&flags.conflictNotifyAssigned, "conflict-notify-assigned", false, "notify assigned users for conflict monitoring")
	cmd.Flags().BoolVar(&flags.conflictNotifyClient, "conflict-notify-client", false, "notify client for conflict monitoring")
	cmd.Flags().BoolVar(&flags.conflictNotifyMe, "conflict-notify-me", false, "notify current user for conflict monitoring")
	cmd.Flags().BoolVar(&flags.conflictTextEnable, "conflict-text-enable", false, "enable conflict text monitoring")
	cmd.Flags().BoolVar(&flags.officeActionEnable, "office-action-enable", false, "enable Office Action monitoring")
	cmd.Flags().BoolVar(&flags.officeActionNotifyAssigned, "office-action-notify-assigned", false, "notify assigned users for Office Action monitoring")
	cmd.Flags().BoolVar(&flags.officeActionNotifyClient, "office-action-notify-client", false, "notify client for Office Action monitoring")
	cmd.Flags().BoolVar(&flags.officeActionNotifyMe, "office-action-notify-me", false, "notify current user for Office Action monitoring")
}

func buildPortfolioMonitorConfig(cmd *cobra.Command, flags *portfolioMonitorConfigFlags) openapi.PortfolioMonitorConfig {
	return openapi.PortfolioMonitorConfig{
		CBPActionEnable:            changedBoolPtr(cmd, "cbp-action-enable", flags.cbpActionEnable),
		ConflictActionEnable:       changedBoolPtr(cmd, "conflict-action-enable", flags.conflictActionEnable),
		ConflictImageEnable:        changedBoolPtr(cmd, "conflict-image-enable", flags.conflictImageEnable),
		ConflictNotifyAssigned:     changedBoolPtr(cmd, "conflict-notify-assigned", flags.conflictNotifyAssigned),
		ConflictNotifyClient:       changedBoolPtr(cmd, "conflict-notify-client", flags.conflictNotifyClient),
		ConflictNotifyMe:           changedBoolPtr(cmd, "conflict-notify-me", flags.conflictNotifyMe),
		ConflictTextEnable:         changedBoolPtr(cmd, "conflict-text-enable", flags.conflictTextEnable),
		OfficeActionEnable:         changedBoolPtr(cmd, "office-action-enable", flags.officeActionEnable),
		OfficeActionNotifyAssigned: changedBoolPtr(cmd, "office-action-notify-assigned", flags.officeActionNotifyAssigned),
		OfficeActionNotifyClient:   changedBoolPtr(cmd, "office-action-notify-client", flags.officeActionNotifyClient),
		OfficeActionNotifyMe:       changedBoolPtr(cmd, "office-action-notify-me", flags.officeActionNotifyMe),
	}
}

func changedIntPtr(cmd *cobra.Command, flag string, value int) *int {
	if cmd.Flags().Changed(flag) {
		return &value
	}
	return nil
}

func changedBoolPtr(cmd *cobra.Command, flag string, value bool) *bool {
	if cmd.Flags().Changed(flag) {
		return &value
	}
	return nil
}
