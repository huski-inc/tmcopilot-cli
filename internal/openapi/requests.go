package openapi

type TrademarkSearchRequest struct {
	Name                 []string `json:"name,omitempty"`
	SerialNumbers        []string `json:"sn,omitempty"`
	RegistrationNumbers  []string `json:"rn,omitempty"`
	Classes              []string `json:"class,omitempty"`
	Statuses             []string `json:"status,omitempty"`
	Owners               []string `json:"owners,omitempty"`
	Regions              []string `json:"regions,omitempty"`
	GoodsServices        []string `json:"goods_services,omitempty"`
	Lawyers              []string `json:"lawyers,omitempty"`
	LawFirms             []string `json:"law_firm,omitempty"`
	DesignSearchCodes    []string `json:"design_search_code,omitempty"`
	DesignSearchPrefixes []string `json:"design_search_prefix,omitempty"`
	SortFilingDate       string   `json:"sort_filing_date,omitempty"`
	SortMark             string   `json:"sort_mark,omitempty"`
	SortSerialNumber     string   `json:"sort_serial_number,omitempty"`
	SortStatus           string   `json:"sort_status,omitempty"`
	SortSimilarity       string   `json:"sort_similarity,omitempty"`
	Limit                int      `json:"limit,omitempty"`
	Page                 int      `json:"page,omitempty"`
}

func (r TrademarkSearchRequest) Empty() bool {
	return len(r.Name) == 0 &&
		len(r.SerialNumbers) == 0 &&
		len(r.RegistrationNumbers) == 0 &&
		len(r.Classes) == 0 &&
		len(r.Statuses) == 0 &&
		len(r.Owners) == 0 &&
		len(r.Regions) == 0 &&
		len(r.GoodsServices) == 0 &&
		len(r.Lawyers) == 0 &&
		len(r.LawFirms) == 0 &&
		len(r.DesignSearchCodes) == 0 &&
		len(r.DesignSearchPrefixes) == 0 &&
		r.SortFilingDate == "" &&
		r.SortMark == "" &&
		r.SortSerialNumber == "" &&
		r.SortStatus == "" &&
		r.SortSimilarity == "" &&
		r.Limit <= 0 &&
		r.Page <= 0
}

type TrademarkDetailRequest struct {
	SerialNumbers                  []string `json:"serial_numbers,omitempty"`
	Country                        string   `json:"country,omitempty"`
	DisableStatements              bool     `json:"disable_statements,omitempty"`
	DisableDgraph                  bool     `json:"disable_dgraph,omitempty"`
	DisableTTAB                    bool     `json:"disable_ttab,omitempty"`
	DisableCaseFileHeader          bool     `json:"disable_case_file_header,omitempty"`
	DisableCaseFileEventStatements bool     `json:"disable_case_file_event_statements,omitempty"`
	DisableCaseFileCorrespondent   bool     `json:"disable_case_file_correspondent,omitempty"`
}

type OfficeActionSearchRequest struct {
	Mark             []string `json:"mark,omitempty"`
	Owners           []string `json:"owners,omitempty"`
	Lawyers          []string `json:"lawyers,omitempty"`
	Classes          []string `json:"class,omitempty"`
	GoodsServices    []string `json:"goods_services,omitempty"`
	IssueType        []string `json:"issue_type,omitempty"`
	DocumentKeywords []string `json:"document_keywords,omitempty"`
	TrademarkStatus  []int    `json:"trademark_status,omitempty"`
	ResponseResult   []int    `json:"response_result,omitempty"`
	FilingDate       []int    `json:"filing_date,omitempty"`
	SortFilingDate   string   `json:"sort_filing_date,omitempty"`
	SortMark         string   `json:"sort_mark,omitempty"`
	SortSerialNumber string   `json:"sort_serial_number,omitempty"`
	SortStatus       string   `json:"sort_status,omitempty"`
}

func (r OfficeActionSearchRequest) Empty() bool {
	return len(r.Mark) == 0 &&
		len(r.Owners) == 0 &&
		len(r.Lawyers) == 0 &&
		len(r.Classes) == 0 &&
		len(r.GoodsServices) == 0 &&
		len(r.IssueType) == 0 &&
		len(r.DocumentKeywords) == 0 &&
		len(r.TrademarkStatus) == 0 &&
		len(r.ResponseResult) == 0 &&
		len(r.FilingDate) == 0 &&
		r.SortFilingDate == "" &&
		r.SortMark == "" &&
		r.SortSerialNumber == "" &&
		r.SortStatus == ""
}

type TTABSearchRequest struct {
	CaseNumber          string   `json:"case_number,omitempty"`
	CaseType            string   `json:"case_type,omitempty"`
	CasePlaintiff       string   `json:"case_plaintiff,omitempty"`
	CaseDefendant       string   `json:"case_defendant,omitempty"`
	CaseLawyer          string   `json:"case_lawyer,omitempty"`
	CaseLawFirm         string   `json:"case_law_firm,omitempty"`
	CaseCitable         string   `json:"case_citable,omitempty"`
	TrademarkMark       string   `json:"tm_mark,omitempty"`
	TrademarkSerial     string   `json:"tm_serial_number,omitempty"`
	TrademarkRegister   string   `json:"tm_registration_number,omitempty"`
	CaseFilingDateStart string   `json:"case_filing_date_start,omitempty"`
	CaseFilingDateEnd   string   `json:"case_filing_date_end,omitempty"`
	SortCaseFilingDate  string   `json:"sort_case_filing_date,omitempty"`
	SortCaseEventDate   string   `json:"sort_case_event_date,omitempty"`
	CaseIssue           []string `json:"case_issue,omitempty"`
}

func (r TTABSearchRequest) Empty() bool {
	return r.CaseNumber == "" &&
		r.CaseType == "" &&
		r.CasePlaintiff == "" &&
		r.CaseDefendant == "" &&
		r.CaseLawyer == "" &&
		r.CaseLawFirm == "" &&
		r.CaseCitable == "" &&
		r.TrademarkMark == "" &&
		r.TrademarkSerial == "" &&
		r.TrademarkRegister == "" &&
		r.CaseFilingDateStart == "" &&
		r.CaseFilingDateEnd == "" &&
		r.SortCaseFilingDate == "" &&
		r.SortCaseEventDate == "" &&
		len(r.CaseIssue) == 0
}

type GapCreateRequest struct {
	Title                 string   `json:"title,omitempty"`
	BaseCompanyName       string   `json:"base_company_name,omitempty"`
	BaseSourceType        string   `json:"base_source_type,omitempty"`
	BenchmarkCompanyName  string   `json:"benchmark_company_name,omitempty"`
	BenchmarkSourceType   string   `json:"benchmark_source_type,omitempty"`
	CompetitorID          string   `json:"competitor_id,omitempty"`
	BusinessContext       string   `json:"business_context,omitempty"`
	ProductFocus          string   `json:"product_focus,omitempty"`
	ReportAudience        string   `json:"report_audience,omitempty"`
	BaseOwnerAliases      []string `json:"base_owner_aliases,omitempty"`
	BenchmarkOwnerAliases []string `json:"benchmark_owner_aliases,omitempty"`
	NiceClasses           []string `json:"nice_classes,omitempty"`
	StatusFilter          []string `json:"status_filter,omitempty"`
	TargetMarkets         []string `json:"target_markets,omitempty"`
	IncludeLive           *bool    `json:"include_live,omitempty"`
	IncludePending        *bool    `json:"include_pending,omitempty"`
	IncludeAbandoned      *bool    `json:"include_abandoned,omitempty"`
	RunImmediately        *bool    `json:"run_immediately,omitempty"`
}

func (r GapCreateRequest) Empty() bool {
	return r.Title == "" &&
		r.BaseCompanyName == "" &&
		r.BaseSourceType == "" &&
		r.BenchmarkCompanyName == "" &&
		r.BenchmarkSourceType == "" &&
		r.CompetitorID == "" &&
		r.BusinessContext == "" &&
		r.ProductFocus == "" &&
		r.ReportAudience == "" &&
		len(r.BaseOwnerAliases) == 0 &&
		len(r.BenchmarkOwnerAliases) == 0 &&
		len(r.NiceClasses) == 0 &&
		len(r.StatusFilter) == 0 &&
		len(r.TargetMarkets) == 0 &&
		r.IncludeLive == nil &&
		r.IncludePending == nil &&
		r.IncludeAbandoned == nil &&
		r.RunImmediately == nil
}

type GapGenerateReportRequest struct {
	SelectedClasses []string `json:"selected_classes,omitempty"`
}

type CommonLawSearchRequest struct {
	Name                  []string `json:"name,omitempty"`
	Platform              string   `json:"platform,omitempty"`
	CollaborationID       string   `json:"collaboration_id,omitempty"`
	CollaborationSharedID string   `json:"collaboration_shared_id,omitempty"`
}

func (r CommonLawSearchRequest) Empty() bool {
	return len(r.Name) == 0 &&
		r.Platform == "" &&
		r.CollaborationID == "" &&
		r.CollaborationSharedID == ""
}

type CommonLawMaxSimilarityRequest struct {
	Keyword string `json:"keyword,omitempty"`
}

type DomainSearchRequest struct {
	Keyword string `json:"keyword,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type DomainMaxSimilarityRequest struct {
	Keyword string `json:"keyword,omitempty"`
}

type ImageSearchTaskRequest struct {
	Bucket        string   `json:"bucket,omitempty"`
	Key           string   `json:"key,omitempty"`
	CloudfrontURL string   `json:"cloudfront_url,omitempty"`
	Countries     []string `json:"countries,omitempty"`
}

func (r ImageSearchTaskRequest) Empty() bool {
	return r.Bucket == "" &&
		r.Key == "" &&
		r.CloudfrontURL == "" &&
		len(r.Countries) == 0
}

type ImageSearchTaskResultRequest struct {
	ID string `json:"id,omitempty"`
}
