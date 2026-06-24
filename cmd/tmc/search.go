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

func newSearchCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search trademark, owner, lawyer, office action, and TTAB data",
	}
	cmd.AddCommand(newTrademarkSearchCommand(opts))
	cmd.AddCommand(newTrademarkDetailCommand(opts))
	cmd.AddCommand(newOfficeActionSearchCommand(opts))
	cmd.AddCommand(newTTABSearchCommand(opts))
	cmd.AddCommand(newTTABCaseCommand(opts))
	cmd.AddCommand(newLawsuitSearchLegacyCommand(opts))
	cmd.AddCommand(newLawsuitGetLegacyCommand(opts))
	cmd.AddCommand(newOwnerSearchCommand(opts))
	cmd.AddCommand(newOwnerRankingCommand(opts))
	cmd.AddCommand(newLawyerSearchCommand(opts))
	cmd.AddCommand(newLawyerRankingCommand(opts))
	cmd.AddCommand(newLawyerContactCommand(opts))
	cmd.AddCommand(newSearchTipsCommand(opts))
	cmd.AddCommand(newSearchSummaryCommand(opts))
	cmd.AddCommand(newTrademarkImageCommand(opts))
	cmd.AddCommand(newUSPTOOfficeActionDocumentCommand(opts))
	return cmd
}

func defaultTrademarkSearchSimilarities() []string {
	return []string{"Exact", "Fuzzy", "Phonetic"}
}

func newTTABCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ttab",
		Short: "Search and fetch TTAB cases",
	}
	cmd.AddCommand(newTTABSearchSubcommand(opts))
	cmd.AddCommand(newTTABCaseSubcommand(opts))
	return cmd
}

func newTrademarkSearchCommand(opts *globalOptions) *cobra.Command {
	var data string
	var names, serials, registrations, classes, statuses, owners, regions []string
	var goodsServices, lawyers, lawFirms, designCodes, designPrefixes, similarities []string
	var limit int
	var page int
	var sortFilingDate, sortMark, sortSerialNumber, sortStatus, sortSimilarity string
	cmd := &cobra.Command{
		Use:     "trademarks",
		Aliases: []string{"trademark"},
		Short:   "Search US trademarks",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.TrademarkSearchRequest{
					Name:                 splitStringValues(names),
					SerialNumbers:        splitStringValues(serials),
					RegistrationNumbers:  splitStringValues(registrations),
					Classes:              splitStringValues(classes),
					Statuses:             splitStringValues(statuses),
					Owners:               splitStringValues(owners),
					Regions:              splitStringValues(regions),
					GoodsServices:        splitStringValues(goodsServices),
					Lawyers:              splitStringValues(lawyers),
					LawFirms:             splitStringValues(lawFirms),
					DesignSearchCodes:    splitStringValues(designCodes),
					DesignSearchPrefixes: splitStringValues(designPrefixes),
					Similarity:           splitStringValues(similarities),
					SortFilingDate:       strings.TrimSpace(sortFilingDate),
					SortMark:             strings.TrimSpace(sortMark),
					SortSerialNumber:     strings.TrimSpace(sortSerialNumber),
					SortStatus:           strings.TrimSpace(sortStatus),
					SortSimilarity:       strings.TrimSpace(sortSimilarity),
				}
				if limit > 0 {
					req.Limit = limit
				}
				if page > 0 {
					req.Page = page
				}
				if req.Empty() {
					return nil, fmt.Errorf("search trademarks requires --data or at least one search/filter flag")
				}
				if len(req.Similarity) == 0 {
					req.Similarity = defaultTrademarkSearchSimilarities()
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/trademark/search", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringArrayVar(&names, "name", nil, "mark text; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&serials, "serial", nil, "serial number; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&registrations, "registration", nil, "registration number; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&classes, "class", nil, "Nice class; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&statuses, "status", nil, "trademark status; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&owners, "owner", nil, "owner name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&regions, "region", nil, "region code; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&goodsServices, "goods-services", nil, "goods/services keyword; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&lawyers, "lawyer", nil, "lawyer name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&lawFirms, "law-firm", nil, "law firm; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&designCodes, "design-code", nil, "design search code; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&designPrefixes, "design-prefix", nil, "design search prefix; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&similarities, "similarity", nil, "similarity analysis type; repeatable or comma-separated; defaults to Exact,Fuzzy,Phonetic")
	cmd.Flags().IntVar(&limit, "limit", 0, "result limit")
	cmd.Flags().IntVar(&page, "page", 0, "result page")
	cmd.Flags().StringVar(&sortFilingDate, "sort-filing-date", "", "sort filing date: asc or desc")
	cmd.Flags().StringVar(&sortMark, "sort-mark", "", "sort mark: asc or desc")
	cmd.Flags().StringVar(&sortSerialNumber, "sort-serial-number", "", "sort serial number: asc or desc")
	cmd.Flags().StringVar(&sortStatus, "sort-status", "", "sort status: asc or desc")
	cmd.Flags().StringVar(&sortSimilarity, "sort-similarity", "", "sort similarity: asc or desc")
	return cmd
}

func newTrademarkDetailCommand(opts *globalOptions) *cobra.Command {
	var serials []string
	var country string
	var disableStatements, disableDgraph, disableTTAB bool
	var disableHeader, disableEvents, disableCorrespondent bool
	cmd := &cobra.Command{
		Use:   "detail [serial...]",
		Short: "Get trademark details",
		RunE: func(cmd *cobra.Command, args []string) error {
			allSerials := append([]string{}, serials...)
			allSerials = append(allSerials, args...)
			body := openapi.TrademarkDetailRequest{
				SerialNumbers:                  splitStringValues(allSerials),
				Country:                        strings.TrimSpace(country),
				DisableStatements:              disableStatements,
				DisableDgraph:                  disableDgraph,
				DisableTTAB:                    disableTTAB,
				DisableCaseFileHeader:          disableHeader,
				DisableCaseFileEventStatements: disableEvents,
				DisableCaseFileCorrespondent:   disableCorrespondent,
			}
			if len(body.SerialNumbers) == 0 {
				return fmt.Errorf("at least one serial number is required")
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/trademark/detail", nil, body)
		},
	}
	cmd.Flags().StringArrayVar(&serials, "serial", nil, "serial number; repeatable or comma-separated")
	cmd.Flags().StringVar(&country, "country", "", "country code; defaults to backend behavior")
	cmd.Flags().BoolVar(&disableStatements, "disable-statements", false, "disable statements in detail response")
	cmd.Flags().BoolVar(&disableDgraph, "disable-dgraph", false, "disable Dgraph enrichment")
	cmd.Flags().BoolVar(&disableTTAB, "disable-ttab", false, "disable TTAB enrichment")
	cmd.Flags().BoolVar(&disableHeader, "disable-case-file-header", false, "disable case file header")
	cmd.Flags().BoolVar(&disableEvents, "disable-case-file-event-statements", false, "disable case file event statements")
	cmd.Flags().BoolVar(&disableCorrespondent, "disable-case-file-correspondent", false, "disable case file correspondent")
	return cmd
}

func newOfficeActionSearchCommand(opts *globalOptions) *cobra.Command {
	var data string
	var marks, owners, lawyers, classes, goodsServices, issueTypes, documentKeywords []string
	var trademarkStatuses, responseResults, filingDates []string
	var sortFilingDate, sortMark, sortSerialNumber, sortStatus string
	cmd := &cobra.Command{
		Use:     "office-actions",
		Aliases: []string{"office-action"},
		Short:   "Search trademark office actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				trademarkStatus, err := splitIntValues(trademarkStatuses, "trademark_status")
				if err != nil {
					return nil, err
				}
				responseResult, err := splitIntValues(responseResults, "response_result")
				if err != nil {
					return nil, err
				}
				filingDate, err := splitIntValues(filingDates, "filing_date")
				if err != nil {
					return nil, err
				}
				req := openapi.OfficeActionSearchRequest{
					Mark:             splitStringValues(marks),
					Owners:           splitStringValues(owners),
					Lawyers:          splitStringValues(lawyers),
					Classes:          splitStringValues(classes),
					GoodsServices:    splitStringValues(goodsServices),
					IssueType:        splitStringValues(issueTypes),
					DocumentKeywords: splitStringValues(documentKeywords),
					TrademarkStatus:  trademarkStatus,
					ResponseResult:   responseResult,
					FilingDate:       filingDate,
					SortFilingDate:   strings.TrimSpace(sortFilingDate),
					SortMark:         strings.TrimSpace(sortMark),
					SortSerialNumber: strings.TrimSpace(sortSerialNumber),
					SortStatus:       strings.TrimSpace(sortStatus),
				}
				if req.Empty() {
					return nil, fmt.Errorf("search office-actions requires --data or at least one search/filter flag")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/trademark/office-action/search", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringArrayVar(&marks, "mark", nil, "mark text; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&owners, "owner", nil, "owner name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&lawyers, "lawyer", nil, "lawyer name; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&classes, "class", nil, "Nice class; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&goodsServices, "goods-services", nil, "goods/services keyword; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&issueTypes, "issue-type", nil, "issue type; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&documentKeywords, "document-keyword", nil, "document keyword; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&trademarkStatuses, "trademark-status", nil, "trademark status integer; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&responseResults, "response-result", nil, "response result integer; repeatable or comma-separated")
	cmd.Flags().StringArrayVar(&filingDates, "filing-date", nil, "filing date timestamp integer; repeatable or comma-separated")
	cmd.Flags().StringVar(&sortFilingDate, "sort-filing-date", "", "sort filing date: asc or desc")
	cmd.Flags().StringVar(&sortMark, "sort-mark", "", "sort mark: asc or desc")
	cmd.Flags().StringVar(&sortSerialNumber, "sort-serial-number", "", "sort serial number: asc or desc")
	cmd.Flags().StringVar(&sortStatus, "sort-status", "", "sort status: asc or desc")
	return cmd
}

func newTTABSearchCommand(opts *globalOptions) *cobra.Command {
	return newTTABSearchCommandFor(opts, "ttab", []string{"ttab-cases"})
}

func newTTABSearchSubcommand(opts *globalOptions) *cobra.Command {
	return newTTABSearchCommandFor(opts, "search", []string{"cases"})
}

func newTTABSearchCommandFor(opts *globalOptions, use string, aliases []string) *cobra.Command {
	var data string
	var caseNumber, caseType, plaintiff, defendant, lawyer, lawFirm, citable string
	var mark, serial, registration, filingStart, filingEnd string
	var issues []string
	var sortFilingDate, sortEventDate string
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Search TTAB cases",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.TTABSearchRequest{
					CaseNumber:          strings.TrimSpace(caseNumber),
					CaseType:            strings.TrimSpace(caseType),
					CasePlaintiff:       strings.TrimSpace(plaintiff),
					CaseDefendant:       strings.TrimSpace(defendant),
					CaseLawyer:          strings.TrimSpace(lawyer),
					CaseLawFirm:         strings.TrimSpace(lawFirm),
					CaseCitable:         strings.TrimSpace(citable),
					TrademarkMark:       strings.TrimSpace(mark),
					TrademarkSerial:     strings.TrimSpace(serial),
					TrademarkRegister:   strings.TrimSpace(registration),
					CaseFilingDateStart: strings.TrimSpace(filingStart),
					CaseFilingDateEnd:   strings.TrimSpace(filingEnd),
					SortCaseFilingDate:  strings.TrimSpace(sortFilingDate),
					SortCaseEventDate:   strings.TrimSpace(sortEventDate),
					CaseIssue:           splitStringValues(issues),
				}
				if req.Empty() {
					return nil, fmt.Errorf("search ttab requires --data or at least one search/filter flag")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/trademark/ttab/search", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&caseNumber, "case-number", "", "TTAB case number")
	cmd.Flags().StringVar(&caseType, "case-type", "", "case type")
	cmd.Flags().StringVar(&plaintiff, "plaintiff", "", "case plaintiff")
	cmd.Flags().StringVar(&defendant, "defendant", "", "case defendant")
	cmd.Flags().StringVar(&lawyer, "lawyer", "", "case lawyer")
	cmd.Flags().StringVar(&lawFirm, "law-firm", "", "case law firm")
	cmd.Flags().StringVar(&citable, "citable", "", "case citable value: y or n")
	cmd.Flags().StringVar(&mark, "mark", "", "trademark mark")
	cmd.Flags().StringVar(&serial, "serial", "", "trademark serial number")
	cmd.Flags().StringVar(&registration, "registration", "", "trademark registration number")
	cmd.Flags().StringVar(&filingStart, "filing-date-start", "", "case filing date start")
	cmd.Flags().StringVar(&filingEnd, "filing-date-end", "", "case filing date end")
	cmd.Flags().StringArrayVar(&issues, "issue", nil, "case issue; repeatable or comma-separated")
	cmd.Flags().StringVar(&sortFilingDate, "sort-filing-date", "", "sort case filing date: asc or desc")
	cmd.Flags().StringVar(&sortEventDate, "sort-event-date", "", "sort case event date: asc or desc")
	return cmd
}

func newTTABCaseCommand(opts *globalOptions) *cobra.Command {
	return newTTABCaseCommandFor(opts, "ttab-case <case-number>", nil)
}

func newTTABCaseSubcommand(opts *globalOptions) *cobra.Command {
	return newTTABCaseCommandFor(opts, "case <case-number>", []string{"get", "detail"})
}

func newTTABCaseCommandFor(opts *globalOptions, use string, aliases []string) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Get a TTAB case by case number",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/trademark/ttab/"+url.PathEscape(args[0]), nil, nil)
		},
	}
}

func newOwnerSearchCommand(opts *globalOptions) *cobra.Command {
	var name string
	var page int
	var limit int
	cmd := &cobra.Command{
		Use:     "owners",
		Aliases: []string{"companies", "company", "owner"},
		Short:   "Search trademark owners and companies",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required")
			}
			query := url.Values{}
			query.Set("name", name)
			if page > 0 {
				query.Set("page", strconv.Itoa(page))
			}
			if limit > 0 {
				query.Set("limit", strconv.Itoa(limit))
			}
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/trademark/owner/search", query, nil)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "owner name to search")
	cmd.Flags().IntVar(&page, "page", 0, "page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "items per page")
	return cmd
}

func newOwnerRankingCommand(opts *globalOptions) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:     "owner-ranking",
		Aliases: []string{"company-ranking"},
		Short:   "Get trademark owner ranking",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			if limit > 0 {
				query.Set("limit", strconv.Itoa(limit))
			}
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/trademark/owner/ranking", query, nil)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "max number of results")
	return cmd
}

func newSearchTipsCommand(opts *globalOptions) *cobra.Command {
	var ownerName, lawyerName, trademarkLawyerName, region string
	cmd := &cobra.Command{
		Use:   "tips",
		Short: "Get trademark search tips",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			setQuery(query, "owner_name", ownerName)
			setQuery(query, "lawyer_name", lawyerName)
			setQuery(query, "trademark_lawyer_name", trademarkLawyerName)
			setQuery(query, "region", region)
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/trademark/search/tips", query, nil)
		},
	}
	cmd.Flags().StringVar(&ownerName, "owner-name", "", "owner name")
	cmd.Flags().StringVar(&lawyerName, "lawyer-name", "", "lawyer name")
	cmd.Flags().StringVar(&trademarkLawyerName, "trademark-lawyer-name", "", "trademark lawyer name")
	cmd.Flags().StringVar(&region, "region", "", "region: us or international")
	return cmd
}

func newSearchSummaryCommand(opts *globalOptions) *cobra.Command {
	var data string
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Generate trademark search summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readDataArg(data)
			if err != nil {
				return err
			}
			if body == nil {
				return fmt.Errorf("--data is required")
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/trademark/search/summary", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	return cmd
}

func newTrademarkImageCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "image",
		Aliases: []string{"image-search"},
		Short:   "Create and inspect trademark image search tasks",
	}
	cmd.AddCommand(newTrademarkImageCreateCommand(opts))
	cmd.AddCommand(newTrademarkImageResultCommand(opts))
	cmd.AddCommand(newTrademarkImageResultPostCommand(opts))
	return cmd
}

func newTrademarkImageCreateCommand(opts *globalOptions) *cobra.Command {
	var data string
	var bucket, key, cloudfrontURL string
	var countries []string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a trademark image search task",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				req := openapi.ImageSearchTaskRequest{
					Bucket:        strings.TrimSpace(bucket),
					Key:           strings.TrimSpace(key),
					CloudfrontURL: strings.TrimSpace(cloudfrontURL),
					Countries:     splitStringValues(countries),
				}
				if req.Bucket == "" {
					return nil, fmt.Errorf("--bucket is required")
				}
				if req.Key == "" {
					return nil, fmt.Errorf("--key is required")
				}
				return req, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/trademark/image/task", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.Flags().StringVar(&bucket, "bucket", "", "uploaded image S3 bucket")
	cmd.Flags().StringVar(&key, "key", "", "uploaded image S3 key")
	cmd.Flags().StringVar(&cloudfrontURL, "cloudfront-url", "", "uploaded image CloudFront URL")
	cmd.Flags().StringArrayVar(&countries, "country", nil, "country code; repeatable or comma-separated")
	return cmd
}

func newTrademarkImageResultCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "result <id>",
		Short: "Get a trademark image search task result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, http.MethodGet, "/trademark/image/task/"+url.PathEscape(args[0])+"/result", nil, nil)
		},
	}
}

func newTrademarkImageResultPostCommand(opts *globalOptions) *cobra.Command {
	var data string
	cmd := &cobra.Command{
		Use:   "result-post <id>",
		Short: "Get a trademark image search task result through the POST endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyFromDataOrBuilder(data, func() (any, error) {
				return openapi.ImageSearchTaskResultRequest{ID: strings.TrimSpace(args[0])}, nil
			})
			if err != nil {
				return err
			}
			return callAPIAndWrite(cmd, opts, http.MethodPost, "/trademark/image/task/result", nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	return cmd
}

func newUSPTOOfficeActionDocumentCommand(opts *globalOptions) *cobra.Command {
	var serialNumber, documentPageID, documentType, documentDate string
	cmd := &cobra.Command{
		Use:   "uspto-document",
		Short: "Download a USPTO Office Action document",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			setQuery(query, "serial_number", serialNumber)
			setQuery(query, "document_page_id", documentPageID)
			setQuery(query, "document_type", documentType)
			setQuery(query, "document_date", documentDate)
			for _, required := range []struct {
				flag  string
				value string
			}{
				{flag: "--serial-number", value: serialNumber},
				{flag: "--document-page-id", value: documentPageID},
				{flag: "--document-type", value: documentType},
				{flag: "--document-date", value: documentDate},
			} {
				if strings.TrimSpace(required.value) == "" {
					return fmt.Errorf("%s is required", required.flag)
				}
			}
			return handleCommand(cmd, func() error {
				return executeAPIDownloadAndWrite(cmd, opts, http.MethodGet, "/trademark/office-action/uspto/document", query, nil, nil)
			})
		},
	}
	cmd.Flags().StringVar(&serialNumber, "serial-number", "", "trademark serial number")
	cmd.Flags().StringVar(&documentPageID, "document-page-id", "", "USPTO document page id")
	cmd.Flags().StringVar(&documentType, "document-type", "", "USPTO document type")
	cmd.Flags().StringVar(&documentDate, "document-date", "", "USPTO document date")
	return cmd
}
