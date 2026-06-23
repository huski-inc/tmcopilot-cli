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

type LawsuitSearchRequest struct {
	CaseAt                    []string `json:"case_at,omitempty"`
	CaseClosedAt              []string `json:"case_closed_at,omitempty"`
	CaseName                  []string `json:"case_name,omitempty"`
	CaseNumberCode            []string `json:"case_number_code,omitempty"`
	PartyName                 []string `json:"party_name,omitempty"`
	PlaintiffName             []string `json:"plaintiff_name,omitempty"`
	DefendantName             []string `json:"defendant_name,omitempty"`
	LawyerName                []string `json:"lawyer_name,omitempty"`
	LawFirmName               []string `json:"law_firm_name,omitempty"`
	Trademark                 []string `json:"trademark,omitempty"`
	UsageIdempotencyKey       string   `json:"usage_idempotency_key,omitempty"`
	Limit                     int      `json:"limit,omitempty"`
	Page                      int      `json:"page,omitempty"`
	SortCaseAt                string   `json:"sort_case_at,omitempty"`
	SortCaseName              string   `json:"sort_case_name,omitempty"`
	SortCaseNumberCode        string   `json:"sort_case_number_code,omitempty"`
	SortIndex                 string   `json:"sort_index,omitempty"`
	SortLawFirmCount          string   `json:"sort_law_firm_count,omitempty"`
	SortLawsuitDefendantCount string   `json:"sort_lawsuit_defendant_count,omitempty"`
	SortLawsuitPlaintiffCount string   `json:"sort_lawsuit_plaintiff_count,omitempty"`
	SortLawyerCount           string   `json:"sort_lawyer_count,omitempty"`
}

func (r LawsuitSearchRequest) Empty() bool {
	return len(r.CaseAt) == 0 &&
		len(r.CaseClosedAt) == 0 &&
		len(r.CaseName) == 0 &&
		len(r.CaseNumberCode) == 0 &&
		len(r.PartyName) == 0 &&
		len(r.PlaintiffName) == 0 &&
		len(r.DefendantName) == 0 &&
		len(r.LawyerName) == 0 &&
		len(r.LawFirmName) == 0 &&
		len(r.Trademark) == 0 &&
		r.UsageIdempotencyKey == "" &&
		r.Limit <= 0 &&
		r.Page <= 0 &&
		r.SortCaseAt == "" &&
		r.SortCaseName == "" &&
		r.SortCaseNumberCode == "" &&
		r.SortIndex == "" &&
		r.SortLawFirmCount == "" &&
		r.SortLawsuitDefendantCount == "" &&
		r.SortLawsuitPlaintiffCount == "" &&
		r.SortLawyerCount == ""
}

type LawsuitListRequest struct {
	Limit                     int    `json:"limit,omitempty"`
	Page                      int    `json:"page,omitempty"`
	SortCaseAt                string `json:"sort_case_at,omitempty"`
	SortCaseName              string `json:"sort_case_name,omitempty"`
	SortCaseNumberCode        string `json:"sort_case_number_code,omitempty"`
	SortIndex                 string `json:"sort_index,omitempty"`
	SortLawFirmCount          string `json:"sort_law_firm_count,omitempty"`
	SortLawsuitDefendantCount string `json:"sort_lawsuit_defendant_count,omitempty"`
	SortLawsuitPlaintiffCount string `json:"sort_lawsuit_plaintiff_count,omitempty"`
	SortLawyerCount           string `json:"sort_lawyer_count,omitempty"`
}

type WideTableTrademarkListRequest struct {
	Limit            int    `json:"limit,omitempty"`
	Page             int    `json:"page,omitempty"`
	Status           *int   `json:"status,omitempty"`
	SortFilingAt     string `json:"sort_filing_at,omitempty"`
	SortIndex        string `json:"sort_index,omitempty"`
	SortLawsuitCount string `json:"sort_lawsuit_count,omitempty"`
	SortMark         string `json:"sort_mark,omitempty"`
	SortSerialNumber string `json:"sort_serial_number,omitempty"`
	SortStatus       string `json:"sort_status,omitempty"`
}

type WideTableLawFirmListRequest struct {
	Limit              int    `json:"limit,omitempty"`
	Page               int    `json:"page,omitempty"`
	SortName           string `json:"sort_name,omitempty"`
	SortRank           string `json:"sort_rank,omitempty"`
	SortTrademarkCount string `json:"sort_trademark_count,omitempty"`
	SortLawsuitCount   string `json:"sort_lawsuit_count,omitempty"`
	SortLawyerCount    string `json:"sort_lawyer_count,omitempty"`
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

type PortfolioImportTrademarksRequest struct {
	Country           string   `json:"country,omitempty"`
	LawyerNames       []string `json:"lawyer_names,omitempty"`
	OrganizationNames []string `json:"organization_names,omitempty"`
	OwnerNames        []string `json:"owner_names,omitempty"`
}

func (r PortfolioImportTrademarksRequest) Empty() bool {
	return r.Country == "" &&
		len(r.LawyerNames) == 0 &&
		len(r.OrganizationNames) == 0 &&
		len(r.OwnerNames) == 0
}

func (r PortfolioImportTrademarksRequest) HasImportCriteria() bool {
	return len(r.LawyerNames) > 0 || len(r.OrganizationNames) > 0 || len(r.OwnerNames) > 0
}

type PortfolioUpdateTrademarkRequest struct {
	AttorneyDocketNumber string `json:"attorney_docket_number,omitempty"`
	Country              string `json:"country,omitempty"`
	Format               *int   `json:"format,omitempty"`
	Status               *int   `json:"status,omitempty"`
	Text                 string `json:"text,omitempty"`
}

func (r PortfolioUpdateTrademarkRequest) Empty() bool {
	return r.AttorneyDocketNumber == "" &&
		r.Country == "" &&
		r.Format == nil &&
		r.Status == nil &&
		r.Text == ""
}

type PortfolioActionStatusUpdateRequest struct {
	Status int    `json:"status"`
	Note   string `json:"note,omitempty"`
}

type PortfolioCBPServiceRequestCreateRequest struct {
	RequestType        string   `json:"request_type,omitempty"`
	TrademarkID        string   `json:"trademark_id,omitempty"`
	SerialNumber       string   `json:"serial_number,omitempty"`
	RegistrationNumber string   `json:"registration_number,omitempty"`
	RecordationNumber  string   `json:"recordation_number,omitempty"`
	RecordationType    string   `json:"recordation_type,omitempty"`
	MarkName           string   `json:"mark_name,omitempty"`
	ContactName        string   `json:"contact_name,omitempty"`
	ContactEmail       string   `json:"contact_email,omitempty"`
	PortsOfEntry       []string `json:"ports_of_entry,omitempty"`
	Notes              string   `json:"notes,omitempty"`
}

type PortfolioTrademarkMetadataRequest struct {
	AttorneyName       string   `json:"attorney_name,omitempty"`
	CBPStatus          string   `json:"cbp_status,omitempty"`
	CustomReminderDate string   `json:"custom_reminder_date,omitempty"`
	ExpiryDate         string   `json:"expiry_date,omitempty"`
	FilingDate         string   `json:"filing_date,omitempty"`
	GoodsServices      string   `json:"goods_services,omitempty"`
	MadridIRN          string   `json:"madrid_irn,omitempty"`
	MarkImageURL       string   `json:"mark_image_url,omitempty"`
	NextEventDate      string   `json:"next_event_date,omitempty"`
	NextEventType      string   `json:"next_event_type,omitempty"`
	NiceClasses        []int    `json:"nice_classes,omitempty"`
	OwnerName          string   `json:"owner_name,omitempty"`
	RegistrationDate   string   `json:"registration_date,omitempty"`
	RegistrationNumber string   `json:"registration_number,omitempty"`
	ReminderIntervals  []string `json:"reminder_intervals,omitempty"`
}

func (r PortfolioTrademarkMetadataRequest) Empty() bool {
	return r.AttorneyName == "" &&
		r.CBPStatus == "" &&
		r.CustomReminderDate == "" &&
		r.ExpiryDate == "" &&
		r.FilingDate == "" &&
		r.GoodsServices == "" &&
		r.MadridIRN == "" &&
		r.MarkImageURL == "" &&
		r.NextEventDate == "" &&
		r.NextEventType == "" &&
		len(r.NiceClasses) == 0 &&
		r.OwnerName == "" &&
		r.RegistrationDate == "" &&
		r.RegistrationNumber == "" &&
		len(r.ReminderIntervals) == 0
}

type PortfolioMonitorConfig struct {
	CBPActionEnable            *bool `json:"cbp_action_enable,omitempty"`
	ConflictActionEnable       *bool `json:"conflict_action_enable,omitempty"`
	ConflictImageEnable        *bool `json:"conflict_image_enable,omitempty"`
	ConflictNotifyAssigned     *bool `json:"conflict_notify_assigned,omitempty"`
	ConflictNotifyClient       *bool `json:"conflict_notify_client,omitempty"`
	ConflictNotifyMe           *bool `json:"conflict_notify_me,omitempty"`
	ConflictTextEnable         *bool `json:"conflict_text_enable,omitempty"`
	OfficeActionEnable         *bool `json:"office_action_enable,omitempty"`
	OfficeActionNotifyAssigned *bool `json:"office_action_notify_assigned,omitempty"`
	OfficeActionNotifyClient   *bool `json:"office_action_notify_client,omitempty"`
	OfficeActionNotifyMe       *bool `json:"office_action_notify_me,omitempty"`
}

func (r PortfolioMonitorConfig) Empty() bool {
	return r.CBPActionEnable == nil &&
		r.ConflictActionEnable == nil &&
		r.ConflictImageEnable == nil &&
		r.ConflictNotifyAssigned == nil &&
		r.ConflictNotifyClient == nil &&
		r.ConflictNotifyMe == nil &&
		r.ConflictTextEnable == nil &&
		r.OfficeActionEnable == nil &&
		r.OfficeActionNotifyAssigned == nil &&
		r.OfficeActionNotifyClient == nil &&
		r.OfficeActionNotifyMe == nil
}

type PortfolioUpdateMonitorConfigRequest struct {
	Config PortfolioMonitorConfig `json:"config"`
}

type PortfolioBatchUpdateMonitorConfigRequest struct {
	Config       PortfolioMonitorConfig `json:"config"`
	TrademarkIDs []string               `json:"trademark_ids,omitempty"`
}

type PortfolioBatchToggleMonitorConfigRequest struct {
	TrademarkIDs []string `json:"trademark_ids,omitempty"`
	MonitorType  string   `json:"monitor_type,omitempty"`
	Enable       *bool    `json:"enable,omitempty"`
	ConflictMode string   `json:"conflict_mode,omitempty"`
}

type PortfolioGroupToggleMonitorConfigRequest struct {
	MonitorType  string `json:"monitor_type,omitempty"`
	Enable       *bool  `json:"enable,omitempty"`
	ConflictMode string `json:"conflict_mode,omitempty"`
}
