package tmc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/huski-inc/tmcopilot-cli/internal/config"
)

func executeRootCommand(t *testing.T, args []string) (string, string) {
	t.Helper()
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	return stdout.String(), stderr.String()
}

func TestSearchTrademarksCommandBuildsOpenAPIRequest(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method mismatch: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/trademark/search" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization header mismatch: %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"items":[]}}`))
	}))
	defer server.Close()

	home := t.TempDir()
	outFile := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--output", outFile,
		"search", "trademarks",
		"--name", "Nike",
		"--class", "25,35",
		"--owner", "Nike Inc",
		"--limit", "5",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}

	if !reflect.DeepEqual(gotBody["name"], []any{"Nike"}) {
		t.Fatalf("name body mismatch: %#v", gotBody["name"])
	}
	if !reflect.DeepEqual(gotBody["class"], []any{"25", "35"}) {
		t.Fatalf("class body mismatch: %#v", gotBody["class"])
	}
	if !reflect.DeepEqual(gotBody["owners"], []any{"Nike Inc"}) {
		t.Fatalf("owners body mismatch: %#v", gotBody["owners"])
	}
	if gotBody["limit"] != float64(5) {
		t.Fatalf("limit body mismatch: %#v", gotBody["limit"])
	}
	raw, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatalf("decode output file: %v", err)
	}
	if output["ok"] != true {
		t.Fatalf("output ok mismatch: %#v", output)
	}
}

func TestTTABSearchCommandsBuildOpenAPIRequest(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		commandHeader string
	}{
		{
			name:          "legacy search group",
			args:          []string{"search", "ttab"},
			commandHeader: "tmc search ttab",
		},
		{
			name:          "ttab group",
			args:          []string{"ttab", "search"},
			commandHeader: "tmc ttab search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			var gotCommand string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method mismatch: %s", r.Method)
				}
				if r.URL.Path != "/api/v1/trademark/ttab/search" {
					t.Fatalf("path mismatch: %s", r.URL.Path)
				}
				gotCommand = r.Header.Get("X-TMCopilot-CLI-Command")
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"list":[],"total":0}}`))
			}))
			defer server.Close()

			args := append([]string{
				"--endpoint", server.URL,
			}, tt.args...)
			args = append(args,
				"--case-number", "91234567",
				"--case-type", "opposition",
				"--plaintiff", "Nike",
				"--defendant", "Adidas",
				"--lawyer", "Smith",
				"--law-firm", "Example LLP",
				"--citable", "y",
				"--mark", "AIR",
				"--serial", "97346091",
				"--registration", "1234567",
				"--filing-date-start", "2024-01-01",
				"--filing-date-end", "2024-12-31",
				"--issue", "opposition,cancellation",
				"--issue", "expungement",
				"--sort-filing-date", "desc",
				"--sort-event-date", "asc",
			)

			executeRootCommand(t, args)

			if gotCommand != tt.commandHeader {
				t.Fatalf("command header = %q, want %q", gotCommand, tt.commandHeader)
			}
			for key, want := range map[string]string{
				"case_number":            "91234567",
				"case_type":              "opposition",
				"case_plaintiff":         "Nike",
				"case_defendant":         "Adidas",
				"case_lawyer":            "Smith",
				"case_law_firm":          "Example LLP",
				"case_citable":           "y",
				"tm_mark":                "AIR",
				"tm_serial_number":       "97346091",
				"tm_registration_number": "1234567",
				"case_filing_date_start": "2024-01-01",
				"case_filing_date_end":   "2024-12-31",
				"sort_case_filing_date":  "desc",
				"sort_case_event_date":   "asc",
			} {
				if gotBody[key] != want {
					t.Fatalf("%s body mismatch: got %#v want %q; body=%#v", key, gotBody[key], want, gotBody)
				}
			}
			if !reflect.DeepEqual(gotBody["case_issue"], []any{"opposition", "cancellation", "expungement"}) {
				t.Fatalf("case_issue body mismatch: %#v", gotBody["case_issue"])
			}
		})
	}
}

func TestTTABCaseCommandsFetchByNumber(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		commandHeader string
	}{
		{
			name:          "legacy search group",
			args:          []string{"search", "ttab-case", "91234567"},
			commandHeader: "tmc search ttab-case",
		},
		{
			name:          "ttab group",
			args:          []string{"ttab", "case", "91234567"},
			commandHeader: "tmc ttab case",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotCommand string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method mismatch: %s", r.Method)
				}
				if r.URL.Path != "/api/v1/trademark/ttab/91234567" {
					t.Fatalf("path mismatch: %s", r.URL.Path)
				}
				gotCommand = r.Header.Get("X-TMCopilot-CLI-Command")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"number":"91234567"}}`))
			}))
			defer server.Close()

			args := append([]string{"--endpoint", server.URL}, tt.args...)
			executeRootCommand(t, args)
			if gotCommand != tt.commandHeader {
				t.Fatalf("command header = %q, want %q", gotCommand, tt.commandHeader)
			}
		})
	}
}

func TestLawsuitSearchCommandsBuildOpenAPIRequest(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		commandHeader string
	}{
		{
			name:          "legacy search group",
			args:          []string{"search", "lawsuits"},
			commandHeader: "tmc search lawsuits",
		},
		{
			name:          "lawsuits group",
			args:          []string{"lawsuits", "search"},
			commandHeader: "tmc lawsuits search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			var gotCommand string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method mismatch: %s", r.Method)
				}
				if r.URL.Path != "/api/v1/trademark/wide-table/lawsuits" {
					t.Fatalf("path mismatch: %s", r.URL.Path)
				}
				gotCommand = r.Header.Get("X-TMCopilot-CLI-Command")
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"list":[],"total":0}}`))
			}))
			defer server.Close()

			args := append([]string{
				"--endpoint", server.URL,
			}, tt.args...)
			args = append(args,
				"--case-at", "2024-01-01",
				"--case-closed-at", "2024-12-31",
				"--case-name", "Nike v Adidas,Nike v Puma",
				"--case-number-code", "1:24-cv-123",
				"--party", "Nike",
				"--plaintiff", "Nike Inc",
				"--defendant", "Adidas AG",
				"--lawyer", "Smith",
				"--law-firm", "Example LLP",
				"--trademark", "AIR,97346091",
				"--usage-idempotency-key", "usage_1",
				"--limit", "25",
				"--page", "2",
				"--sort-case-at", "desc",
				"--sort-case-name", "asc",
				"--sort-case-number-code", "desc",
				"--sort-index", "asc",
				"--sort-law-firm-count", "desc",
				"--sort-lawsuit-defendant-count", "asc",
				"--sort-lawsuit-plaintiff-count", "desc",
				"--sort-lawyer-count", "asc",
			)

			executeRootCommand(t, args)

			if gotCommand != tt.commandHeader {
				t.Fatalf("command header = %q, want %q", gotCommand, tt.commandHeader)
			}
			for key, want := range map[string][]any{
				"case_at":          {"2024-01-01"},
				"case_closed_at":   {"2024-12-31"},
				"case_name":        {"Nike v Adidas", "Nike v Puma"},
				"case_number_code": {"1:24-cv-123"},
				"party_name":       {"Nike"},
				"plaintiff_name":   {"Nike Inc"},
				"defendant_name":   {"Adidas AG"},
				"lawyer_name":      {"Smith"},
				"law_firm_name":    {"Example LLP"},
				"trademark":        {"AIR", "97346091"},
			} {
				if !reflect.DeepEqual(gotBody[key], want) {
					t.Fatalf("%s body mismatch: got %#v want %#v; body=%#v", key, gotBody[key], want, gotBody)
				}
			}
			for key, want := range map[string]string{
				"usage_idempotency_key":        "usage_1",
				"sort_case_at":                 "desc",
				"sort_case_name":               "asc",
				"sort_case_number_code":        "desc",
				"sort_index":                   "asc",
				"sort_law_firm_count":          "desc",
				"sort_lawsuit_defendant_count": "asc",
				"sort_lawsuit_plaintiff_count": "desc",
				"sort_lawyer_count":            "asc",
			} {
				if gotBody[key] != want {
					t.Fatalf("%s body mismatch: got %#v want %q; body=%#v", key, gotBody[key], want, gotBody)
				}
			}
			if gotBody["limit"] != float64(25) || gotBody["page"] != float64(2) {
				t.Fatalf("pagination body mismatch: %#v", gotBody)
			}
		})
	}
}

func TestLawsuitGetCommandsFetchByCaseNumber(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		commandHeader string
	}{
		{
			name:          "legacy search group",
			args:          []string{"search", "lawsuit", "CASE123"},
			commandHeader: "tmc search lawsuit",
		},
		{
			name:          "lawsuits group",
			args:          []string{"lawsuits", "get", "CASE123"},
			commandHeader: "tmc lawsuits get",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotCommand string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method mismatch: %s", r.Method)
				}
				if r.URL.Path != "/api/v1/trademark/wide-table/lawsuits/CASE123" {
					t.Fatalf("path mismatch: %s", r.URL.Path)
				}
				gotCommand = r.Header.Get("X-TMCopilot-CLI-Command")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"case_number":"CASE123"}}`))
			}))
			defer server.Close()

			args := append([]string{"--endpoint", server.URL}, tt.args...)
			executeRootCommand(t, args)
			if gotCommand != tt.commandHeader {
				t.Fatalf("command header = %q, want %q", gotCommand, tt.commandHeader)
			}
		})
	}
}

func TestLawsuitRelatedCommandsBuildOpenAPIRequest(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		path          string
		commandHeader string
	}{
		{
			name:          "brand owner lawsuits",
			args:          []string{"lawsuits", "brand-owner", "owner_graph_1"},
			path:          "/api/v1/trademark/wide-table/brand-owners/owner_graph_1/lawsuits",
			commandHeader: "tmc lawsuits brand-owner",
		},
		{
			name:          "lawyer lawsuits",
			args:          []string{"lawsuits", "lawyer", "lawyer_graph_1"},
			path:          "/api/v1/trademark/wide-table/lawyers/lawyer_graph_1/lawsuits",
			commandHeader: "tmc lawsuits lawyer",
		},
		{
			name:          "lawyers group lawsuits",
			args:          []string{"lawyers", "lawsuits", "lawyer_graph_1"},
			path:          "/api/v1/trademark/wide-table/lawyers/lawyer_graph_1/lawsuits",
			commandHeader: "tmc lawyers lawsuits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			var gotCommand string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method mismatch: %s", r.Method)
				}
				if r.URL.Path != tt.path {
					t.Fatalf("path mismatch: %s", r.URL.Path)
				}
				gotCommand = r.Header.Get("X-TMCopilot-CLI-Command")
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"list":[],"total":0}}`))
			}))
			defer server.Close()

			args := append([]string{"--endpoint", server.URL}, tt.args...)
			args = append(args,
				"--limit", "10",
				"--page", "3",
				"--sort-case-at", "asc",
				"--sort-lawyer-count", "desc",
			)
			executeRootCommand(t, args)
			if gotCommand != tt.commandHeader {
				t.Fatalf("command header = %q, want %q", gotCommand, tt.commandHeader)
			}
			if gotBody["limit"] != float64(10) || gotBody["page"] != float64(3) {
				t.Fatalf("pagination body mismatch: %#v", gotBody)
			}
			if gotBody["sort_case_at"] != "asc" || gotBody["sort_lawyer_count"] != "desc" {
				t.Fatalf("sort body mismatch: %#v", gotBody)
			}
		})
	}
}

func TestLawyerSearchCommandsBuildQuery(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		commandHeader string
	}{
		{
			name:          "legacy search group",
			args:          []string{"search", "lawyers"},
			commandHeader: "tmc search lawyers",
		},
		{
			name:          "lawyers group",
			args:          []string{"lawyers", "search"},
			commandHeader: "tmc lawyers search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotCommand string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method mismatch: %s", r.Method)
				}
				if r.URL.Path != "/api/v1/trademark/lawyer/search" {
					t.Fatalf("path mismatch: %s", r.URL.Path)
				}
				gotCommand = r.Header.Get("X-TMCopilot-CLI-Command")
				query := r.URL.Query()
				for key, want := range map[string]string{
					"name":          "Smith",
					"city":          "San Francisco",
					"state":         "CA",
					"zip_code":      "94105",
					"email_name":    "sarah",
					"email_domain":  "example.com",
					"email_address": "sarah@example.com",
					"page":          "2",
					"limit":         "25",
				} {
					if got := query.Get(key); got != want {
						t.Fatalf("query %s = %q, want %q", key, got, want)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"list":[],"total":0}}`))
			}))
			defer server.Close()

			args := append([]string{"--endpoint", server.URL}, tt.args...)
			args = append(args,
				"--name", "Smith",
				"--city", "San Francisco",
				"--state", "CA",
				"--zip-code", "94105",
				"--email-name", "sarah",
				"--email-domain", "example.com",
				"--email-address", "sarah@example.com",
				"--page", "2",
				"--limit", "25",
			)
			executeRootCommand(t, args)
			if gotCommand != tt.commandHeader {
				t.Fatalf("command header = %q, want %q", gotCommand, tt.commandHeader)
			}
		})
	}
}

func TestLawyerRankingAndContactCommands(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		path          string
		query         map[string]string
		commandHeader string
	}{
		{
			name:          "legacy ranking",
			args:          []string{"search", "lawyer-ranking", "--type", "combined_metrics", "--limit", "10"},
			path:          "/api/v1/trademark/lawyer/ranking",
			query:         map[string]string{"type": "combined_metrics", "limit": "10"},
			commandHeader: "tmc search lawyer-ranking",
		},
		{
			name:          "top-level ranking",
			args:          []string{"lawyers", "ranking", "--type", "combined_metrics", "--limit", "10"},
			path:          "/api/v1/trademark/lawyer/ranking",
			query:         map[string]string{"type": "combined_metrics", "limit": "10"},
			commandHeader: "tmc lawyers ranking",
		},
		{
			name:          "legacy contact",
			args:          []string{"search", "lawyer-contact", "--name", "Sarah Smith"},
			path:          "/api/v1/trademark/lawyer/contact",
			query:         map[string]string{"name": "Sarah Smith"},
			commandHeader: "tmc search lawyer-contact",
		},
		{
			name:          "top-level contact",
			args:          []string{"lawyers", "contact", "--name", "Sarah Smith"},
			path:          "/api/v1/trademark/lawyer/contact",
			query:         map[string]string{"name": "Sarah Smith"},
			commandHeader: "tmc lawyers contact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotCommand string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method mismatch: %s", r.Method)
				}
				if r.URL.Path != tt.path {
					t.Fatalf("path mismatch: %s", r.URL.Path)
				}
				gotCommand = r.Header.Get("X-TMCopilot-CLI-Command")
				query := r.URL.Query()
				for key, want := range tt.query {
					if got := query.Get(key); got != want {
						t.Fatalf("query %s = %q, want %q", key, got, want)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{}}`))
			}))
			defer server.Close()

			args := append([]string{"--endpoint", server.URL}, tt.args...)
			executeRootCommand(t, args)
			if gotCommand != tt.commandHeader {
				t.Fatalf("command header = %q, want %q", gotCommand, tt.commandHeader)
			}
		})
	}
}

func TestLawyerWideTableCommandsBuildOpenAPIRequest(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		method        string
		path          string
		commandHeader string
		wantBody      map[string]any
	}{
		{
			name:          "get lawyer info",
			args:          []string{"lawyers", "get", "lawyer_graph_1"},
			method:        http.MethodGet,
			path:          "/api/v1/trademark/wide-table/lawyers/lawyer_graph_1",
			commandHeader: "tmc lawyers get",
		},
		{
			name: "list lawyer trademarks",
			args: []string{
				"lawyers", "trademarks", "lawyer_graph_1",
				"--limit", "10",
				"--page", "3",
				"--status", "1",
				"--sort-filing-at", "desc",
				"--sort-index", "asc",
				"--sort-lawsuit-count", "desc",
				"--sort-mark", "asc",
				"--sort-serial-number", "desc",
				"--sort-status", "asc",
			},
			method:        http.MethodPost,
			path:          "/api/v1/trademark/wide-table/lawyers/lawyer_graph_1/trademarks",
			commandHeader: "tmc lawyers trademarks",
			wantBody: map[string]any{
				"limit":              float64(10),
				"page":               float64(3),
				"status":             float64(1),
				"sort_filing_at":     "desc",
				"sort_index":         "asc",
				"sort_lawsuit_count": "desc",
				"sort_mark":          "asc",
				"sort_serial_number": "desc",
				"sort_status":        "asc",
			},
		},
		{
			name: "list lawyer trademarks with status zero",
			args: []string{
				"lawyers", "trademarks", "lawyer_graph_1",
				"--status", "0",
			},
			method:        http.MethodPost,
			path:          "/api/v1/trademark/wide-table/lawyers/lawyer_graph_1/trademarks",
			commandHeader: "tmc lawyers trademarks",
			wantBody: map[string]any{
				"status": float64(0),
			},
		},
		{
			name: "list lawyer law firms",
			args: []string{
				"lawyers", "law-firms", "lawyer_graph_1",
				"--limit", "20",
				"--page", "2",
				"--sort-name", "asc",
				"--sort-rank", "desc",
				"--sort-trademark-count", "asc",
				"--sort-lawsuit-count", "desc",
				"--sort-lawyer-count", "asc",
			},
			method:        http.MethodPost,
			path:          "/api/v1/trademark/wide-table/lawyers/lawyer_graph_1/law-firms",
			commandHeader: "tmc lawyers law-firms",
			wantBody: map[string]any{
				"limit":                float64(20),
				"page":                 float64(2),
				"sort_name":            "asc",
				"sort_rank":            "desc",
				"sort_trademark_count": "asc",
				"sort_lawsuit_count":   "desc",
				"sort_lawyer_count":    "asc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			var gotCommand string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Fatalf("method mismatch: %s", r.Method)
				}
				if r.URL.Path != tt.path {
					t.Fatalf("path mismatch: %s", r.URL.Path)
				}
				gotCommand = r.Header.Get("X-TMCopilot-CLI-Command")
				if tt.wantBody != nil {
					if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
						t.Fatalf("decode request body: %v", err)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{}}`))
			}))
			defer server.Close()

			args := append([]string{"--endpoint", server.URL}, tt.args...)
			executeRootCommand(t, args)
			if gotCommand != tt.commandHeader {
				t.Fatalf("command header = %q, want %q", gotCommand, tt.commandHeader)
			}
			for key, want := range tt.wantBody {
				if gotBody[key] != want {
					t.Fatalf("%s body mismatch: got %#v want %#v; body=%#v", key, gotBody[key], want, gotBody)
				}
			}
		})
	}
}

func TestConfigInitUsesPersistentEndpointOverride(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_ENDPOINT", "")
	t.Setenv("TMC_ENDPOINT", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", "https://api.example.test/", "config", "init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Profiles[config.DefaultProfile].Endpoint; got != "https://api.example.test" {
		t.Fatalf("endpoint = %q", got)
	}
}

func TestAuthImportKeyReadsStdinAndUsesPersistentEndpoint(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(" tmc_stdin_key \n"))
	cmd.SetArgs([]string{
		"--endpoint", "https://api.example.test/",
		"auth", "import-key",
		"--name", "local",
		"--api-key-stdin",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Profiles["local"].Endpoint; got != "https://api.example.test" {
		t.Fatalf("endpoint = %q", got)
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles["local"].APIKey; got != "tmc_stdin_key" {
		t.Fatalf("api key = %q", got)
	}
}

func TestSetupStoresAPIKeyAndChecks(t *testing.T) {
	var sawAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tmc_setup_key" {
			t.Fatalf("authorization header mismatch: %q", got)
		}
		sawAuth = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"email":"setup@example.com"}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(" tmc_setup_key \n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"setup",
		"--api-key-stdin",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup failed: %v stderr=%s", err, stderr.String())
	}
	if !sawAuth {
		t.Fatal("auth check was not called")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Profiles[config.DefaultProfile].Endpoint; got != server.URL {
		t.Fatalf("endpoint = %q", got)
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles[config.DefaultProfile].APIKey; got != "tmc_setup_key" {
		t.Fatalf("api key = %q", got)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"auth_method":"api_key"`)) {
		t.Fatalf("stdout missing setup result: %s", stdout.String())
	}
}

func TestAuthLoginAuthorizesInBrowserAndStoresAPIKey(t *testing.T) {
	var sawCreateAuthorization, sawPoll, sawCheck bool
	var gotDeviceUUID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/api-key-authorizations":
			sawCreateAuthorization = true
			if got := r.Header.Get("Authorization"); got != "" {
				t.Fatalf("authorization create Authorization header = %q", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode authorization body: %v", err)
			}
			gotDeviceUUID, _ = body["device_uuid"].(string)
			if gotDeviceUUID == "" {
				t.Fatalf("device_uuid was not sent: %#v", body)
			}
			if body["device_name"] != "Codex CLI" {
				t.Fatalf("device_name = %#v", body["device_name"])
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"authorization_id":"akreq_1","authorization_url":"https://app.example.test/api-key-authorize?request_id=akreq_1","poll_token":"poll_token_1","authorization_expires_in":60,"interval":0}}`))
		case "/api/v1/auth/api-key-authorizations/akreq_1/result":
			sawPoll = true
			if got := r.Header.Get("Authorization"); got != "Bearer poll_token_1" {
				t.Fatalf("poll Authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"status":"approved","api_key":"tmc_authorized_key","key":{"id":"key_1","name":"Codex CLI","bound_device_uuid":"` + gotDeviceUUID + `","bound_device_name":"Codex CLI","key_prefix":"tmc_aut","created_at":1710000000}}}`))
		case "/api/v1/auth/me":
			sawCheck = true
			if got := r.Header.Get("Authorization"); got != "Bearer tmc_authorized_key" {
				t.Fatalf("check Authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"email":"user@example.com"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--no-browser",
		"--device-name", "Codex CLI",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login failed: %v stderr=%s", err, stderr.String())
	}
	if !sawCreateAuthorization || !sawPoll || !sawCheck {
		t.Fatalf("expected create/poll/check calls, got create=%v poll=%v check=%v", sawCreateAuthorization, sawPoll, sawCheck)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Profiles[config.DefaultProfile].DeviceUUID; got == "" || got != gotDeviceUUID {
		t.Fatalf("stored device uuid = %q, sent %q", got, gotDeviceUUID)
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles[config.DefaultProfile].APIKey; got != "tmc_authorized_key" {
		t.Fatalf("stored api key = %q", got)
	}
	for _, leaked := range [][]byte{
		[]byte("tmc_authorized_key"),
		[]byte("poll_token_1"),
		[]byte("key_1"),
		[]byte("tmc_aut"),
		[]byte(gotDeviceUUID),
		[]byte("user@example.com"),
		[]byte(`"key_prefix"`),
		[]byte(`"bound_device_uuid"`),
	} {
		if bytes.Contains(stdout.Bytes(), leaked) {
			t.Fatalf("stdout leaked secret %q: %s", leaked, stdout.String())
		}
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"auth_method":"api_key_authorization"`)) {
		t.Fatalf("stdout missing authorization result: %s", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`https://app.example.test/api-key-authorize?request_id=akreq_1`)) {
		t.Fatalf("stderr missing authorization URL: %s", stderr.String())
	}
}

func TestAuthLoginNoWaitStoresPendingAuthorizationOnly(t *testing.T) {
	var sawCreateAuthorization bool
	var gotDeviceUUID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/api-key-authorizations":
			sawCreateAuthorization = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode authorization body: %v", err)
			}
			gotDeviceUUID, _ = body["device_uuid"].(string)
			if gotDeviceUUID == "" {
				t.Fatalf("device_uuid was not sent: %#v", body)
			}
			if body["device_name"] != "Codex CLI" {
				t.Fatalf("device_name = %#v", body["device_name"])
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"authorization_id":"akreq_1","authorization_url":"https://app.example.test/api-key-authorize?request_id=akreq_1","poll_token":"poll_token_1","authorization_expires_in":60,"interval":0}}`))
		case "/api/v1/auth/api-key-authorizations/akreq_1/result", "/api/v1/auth/me":
			t.Fatalf("no-wait should not poll or check credentials, got path: %s", r.URL.Path)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--no-wait",
		"--device-name", "Codex CLI",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login --no-wait failed: %v stderr=%s", err, stderr.String())
	}
	if !sawCreateAuthorization {
		t.Fatal("authorization request was not created")
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("credentials should not be written during no-wait, stat err=%v", err)
	}
	store, err := loadPendingAuthorizationStore()
	if err != nil {
		t.Fatalf("load pending store: %v", err)
	}
	pending, ok := store.Authorizations["akreq_1"]
	if !ok {
		t.Fatalf("pending authorization not stored: %#v", store.Authorizations)
	}
	if pending.PollToken != "poll_token_1" || pending.DeviceUUID != gotDeviceUUID || pending.DeviceName != "Codex CLI" {
		t.Fatalf("pending authorization mismatch: %#v", pending)
	}

	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			Stored        bool   `json:"stored"`
			ResumeCommand string `json:"resume_command"`
			SetupResume   string `json:"setup_resume"`
			Authorization struct {
				ID               string `json:"id"`
				Status           string `json:"status"`
				AuthorizationURL string `json:"authorization_url"`
			} `json:"authorization"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode no-wait result: %v output=%s", err, stdout.String())
	}
	if result.Data.Stored || result.Data.Authorization.ID != "akreq_1" || result.Data.Authorization.Status != "pending" {
		t.Fatalf("no-wait result mismatch: %#v", result.Data)
	}
	if result.Data.ResumeCommand != "tmc auth login --request-id akreq_1" || result.Data.SetupResume != "tmc setup --request-id akreq_1" {
		t.Fatalf("resume commands mismatch: %#v", result.Data)
	}
	if result.Data.Authorization.AuthorizationURL != "https://app.example.test/api-key-authorize?request_id=akreq_1" {
		t.Fatalf("authorization url = %q", result.Data.Authorization.AuthorizationURL)
	}
	for _, leaked := range [][]byte{
		[]byte("poll_token_1"),
		[]byte(gotDeviceUUID),
	} {
		if bytes.Contains(stdout.Bytes(), leaked) || bytes.Contains(stderr.Bytes(), leaked) {
			t.Fatalf("no-wait output leaked secret %q: stdout=%s stderr=%s", leaked, stdout.String(), stderr.String())
		}
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`tmc auth login --request-id akreq_1`)) {
		t.Fatalf("stderr missing resume command: %s", stderr.String())
	}
}

func TestAuthLoginRequestIDResumesPendingAuthorization(t *testing.T) {
	var createCalls, pollCalls, checkCalls int
	var gotDeviceUUID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/api-key-authorizations":
			createCalls++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode authorization body: %v", err)
			}
			gotDeviceUUID, _ = body["device_uuid"].(string)
			if gotDeviceUUID == "" {
				t.Fatalf("device_uuid was not sent: %#v", body)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"authorization_id":"akreq_1","authorization_url":"https://app.example.test/api-key-authorize?request_id=akreq_1","poll_token":"poll_token_1","authorization_expires_in":60,"interval":0}}`))
		case "/api/v1/auth/api-key-authorizations/akreq_1/result":
			pollCalls++
			if got := r.Header.Get("Authorization"); got != "Bearer poll_token_1" {
				t.Fatalf("poll Authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"status":"approved","api_key":"tmc_authorized_key","key":{"id":"key_1","name":"Codex CLI","bound_device_uuid":"` + gotDeviceUUID + `","bound_device_name":"Codex CLI","key_prefix":"tmc_aut","created_at":1710000000}}}`))
		case "/api/v1/auth/me":
			checkCalls++
			if got := r.Header.Get("Authorization"); got != "Bearer tmc_authorized_key" {
				t.Fatalf("check Authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"email":"user@example.com"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--no-wait",
		"--device-name", "Codex CLI",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login --no-wait failed: %v stderr=%s", err, stderr.String())
	}
	if createCalls != 1 || pollCalls != 0 || checkCalls != 0 {
		t.Fatalf("unexpected calls after no-wait: create=%d poll=%d check=%d", createCalls, pollCalls, checkCalls)
	}
	for _, leaked := range [][]byte{[]byte("poll_token_1"), []byte("tmc_authorized_key")} {
		if bytes.Contains(stdout.Bytes(), leaked) || bytes.Contains(stderr.Bytes(), leaked) {
			t.Fatalf("no-wait output leaked secret %q: stdout=%s stderr=%s", leaked, stdout.String(), stderr.String())
		}
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--request-id", "akreq_1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login --request-id failed: %v stderr=%s", err, stderr.String())
	}
	if createCalls != 1 || pollCalls != 1 || checkCalls != 1 {
		t.Fatalf("unexpected calls after resume: create=%d poll=%d check=%d", createCalls, pollCalls, checkCalls)
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles[config.DefaultProfile].APIKey; got != "tmc_authorized_key" {
		t.Fatalf("stored api key = %q", got)
	}
	store, err := loadPendingAuthorizationStore()
	if err != nil {
		t.Fatalf("load pending store: %v", err)
	}
	if _, ok := store.Authorizations["akreq_1"]; ok {
		t.Fatalf("pending authorization was not removed: %#v", store.Authorizations)
	}
	for _, leaked := range [][]byte{
		[]byte("tmc_authorized_key"),
		[]byte("poll_token_1"),
		[]byte("key_1"),
		[]byte("tmc_aut"),
		[]byte(gotDeviceUUID),
		[]byte("user@example.com"),
		[]byte(`"key_prefix"`),
		[]byte(`"bound_device_uuid"`),
	} {
		if bytes.Contains(stdout.Bytes(), leaked) || bytes.Contains(stderr.Bytes(), leaked) {
			t.Fatalf("resume output leaked secret %q: stdout=%s stderr=%s", leaked, stdout.String(), stderr.String())
		}
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"stored":true`)) ||
		!bytes.Contains(stdout.Bytes(), []byte(`"id":"akreq_1"`)) ||
		!bytes.Contains(stdout.Bytes(), []byte(`"verified":true`)) {
		t.Fatalf("resume result missing status: %s", stdout.String())
	}
}

func TestSetupNoWaitPrintsSetupResumeCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/api-key-authorizations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"authorization_id":"akreq_setup","authorization_url":"https://app.example.test/api-key-authorize?request_id=akreq_setup","poll_token":"poll_token_setup","authorization_expires_in":60,"interval":0}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"setup",
		"--no-wait",
		"--device-name", "Codex CLI",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup --no-wait failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`tmc setup --request-id akreq_setup`)) {
		t.Fatalf("stderr missing setup resume command: %s", stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"setup_resume":"tmc setup --request-id akreq_setup"`)) {
		t.Fatalf("stdout missing setup resume command: %s", stdout.String())
	}
	for _, leaked := range [][]byte{[]byte("poll_token_setup")} {
		if bytes.Contains(stdout.Bytes(), leaked) || bytes.Contains(stderr.Bytes(), leaked) {
			t.Fatalf("setup no-wait output leaked secret %q: stdout=%s stderr=%s", leaked, stdout.String(), stderr.String())
		}
	}
}

func TestAuthLoginAuthorizationRequestOutAndIdempotencyHeader(t *testing.T) {
	var sawCreateAuthorization bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/api-key-authorizations":
			sawCreateAuthorization = true
			if got := r.Header.Get("Idempotency-Key"); got != "idem-1" {
				t.Fatalf("Idempotency-Key = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"authorization_id":"akreq_1","authorization_url":"https://app.example.test/api-key-authorize?request_id=akreq_1","poll_token":"poll_token_1","authorization_expires_in":60,"interval":0}}`))
		case "/api/v1/auth/api-key-authorizations/akreq_1/result":
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"status":"approved","api_key":"tmc_authorized_key","key":{"id":"key_1","name":"Codex CLI","key_prefix":"tmc_aut"}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")
	requestOut := filepath.Join(t.TempDir(), "auth-plan.json")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--request-out", requestOut,
		"--idempotency-key", "idem-1",
		"auth", "login",
		"--no-browser",
		"--device-name", "Codex CLI",
		"--check=false",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login failed: %v stderr=%s", err, stderr.String())
	}
	if !sawCreateAuthorization {
		t.Fatal("authorization request was not created")
	}
	raw, err := os.ReadFile(requestOut)
	if err != nil {
		t.Fatalf("read request out: %v", err)
	}
	if !bytes.Contains(raw, []byte(`/auth/api-key-authorizations`)) ||
		!bytes.Contains(raw, []byte(`"auth_method": "api_key_authorization"`)) {
		t.Fatalf("request out missing authorization plan: %s", string(raw))
	}
	for _, leaked := range [][]byte{[]byte("tmc_authorized_key"), []byte("poll_token_1")} {
		if bytes.Contains(raw, leaked) || bytes.Contains(stdout.Bytes(), leaked) {
			t.Fatalf("secret %q leaked: request_out=%s stdout=%s", leaked, string(raw), stdout.String())
		}
	}
}

func TestAuthLoginLegacyPasswordCreatesAndStoresCLIAPIKey(t *testing.T) {
	var sawLogin, sawCreateKey, sawCheck bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			sawLogin = true
			if got := r.Header.Get("Authorization"); got != "" {
				t.Fatalf("login authorization header = %q", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode login body: %v", err)
			}
			if body["email"] != "user@example.com" || body["password"] != "secret" {
				t.Fatalf("login body mismatch: %#v", body)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"tokens":{"access_token":"jwt_login_token","expires_in":3600},"user":{"email":"user@example.com"}}}`))
		case "/api/v1/auth/api-keys":
			sawCreateKey = true
			if got := r.Header.Get("Authorization"); got != "Bearer jwt_login_token" {
				t.Fatalf("api key authorization header = %q", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode api key body: %v", err)
			}
			if body["name"] != "codex cli" {
				t.Fatalf("api key name = %#v", body["name"])
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"raw_key":"tmc_created_key","key":{"id":"key_1","name":"codex cli","key_prefix":"tmc_cre"}}}`))
		case "/api/v1/auth/me":
			sawCheck = true
			if got := r.Header.Get("Authorization"); got != "Bearer tmc_created_key" {
				t.Fatalf("check authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"email":"user@example.com"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("secret\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--email", "user@example.com",
		"--password-stdin",
		"--key-name", "codex cli",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login failed: %v stderr=%s", err, stderr.String())
	}
	if !sawLogin || !sawCreateKey || !sawCheck {
		t.Fatalf("expected login/create/check calls, got login=%v create=%v check=%v", sawLogin, sawCreateKey, sawCheck)
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles[config.DefaultProfile].APIKey; got != "tmc_created_key" {
		t.Fatalf("stored api key = %q", got)
	}
	if bytes.Contains(stdout.Bytes(), []byte("tmc_created_key")) {
		t.Fatalf("stdout leaked raw api key: %s", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("secret")) {
		t.Fatalf("stdout leaked password: %s", stdout.String())
	}
	for _, leaked := range [][]byte{
		[]byte("key_1"),
		[]byte("tmc_cre"),
		[]byte("user@example.com"),
		[]byte(`"key_prefix"`),
	} {
		if bytes.Contains(stdout.Bytes(), leaked) {
			t.Fatalf("stdout leaked credential metadata %q: %s", leaked, stdout.String())
		}
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"auth_method":"login_api_key"`)) {
		t.Fatalf("stdout missing login setup result: %s", stdout.String())
	}
}

func TestSetupDryRunDoesNotCallAPIOrWriteCredentials(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("tmc_dryrun_key\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--dry-run",
		"setup",
		"--api-key-stdin",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup dry-run failed: %v stderr=%s", err, stderr.String())
	}
	if called {
		t.Fatal("server was called during setup dry-run")
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("credentials should not be written during dry-run, stat err=%v", err)
	}
	if bytes.Contains(stdout.Bytes(), []byte("tmc_dryrun_key")) {
		t.Fatalf("dry-run leaked api key: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"dry_run":true`)) {
		t.Fatalf("dry-run output missing marker: %s", stdout.String())
	}
}

func TestSetupDryRunDoesNotImportEnvAPIKeyByDefault(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "tmc_env_key")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dry-run", "setup"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup dry-run failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"auth_method":"api_key_authorization"`)) ||
		!bytes.Contains(stdout.Bytes(), []byte(`"/auth/api-key-authorizations"`)) {
		t.Fatalf("setup should plan API key authorization flow instead of env API key import: %s", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte(`"api_key_source":"env"`)) {
		t.Fatalf("setup should not import env API key by default: %s", stdout.String())
	}
}

func TestAuthLoginLegacyPasswordDryRunDoesNotCallAPIOrReadPassword(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("secret\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--dry-run",
		"auth", "login",
		"--email", "user@example.com",
		"--password-stdin",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login dry-run failed: %v stderr=%s", err, stderr.String())
	}
	if called {
		t.Fatal("server was called during auth login dry-run")
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("credentials should not be written during dry-run, stat err=%v", err)
	}
	if bytes.Contains(stdout.Bytes(), []byte("secret")) {
		t.Fatalf("dry-run leaked password: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"/auth/login"`)) || !bytes.Contains(stdout.Bytes(), []byte(`redacted`)) {
		t.Fatalf("dry-run output missing redacted login plan: %s", stdout.String())
	}
}

func TestAuthLogoutRemovesLocalCredential(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cfg := config.DefaultConfig()
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := config.SaveCredentials(&config.Credentials{Profiles: map[string]config.Credential{
		config.DefaultProfile: {APIKey: "tmc_logout_key"},
	}}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"auth", "logout"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth logout failed: %v stderr=%s", err, stderr.String())
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if _, ok := creds.Profiles[config.DefaultProfile]; ok {
		t.Fatalf("credential was not removed: %#v", creds.Profiles)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"removed":1`)) {
		t.Fatalf("logout output missing removal count: %s", stdout.String())
	}
}

func TestAuthLoginLegacyPasswordWritesRedactedRequestOutAndIdempotencyHeader(t *testing.T) {
	var sawCreateKey bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"tokens":{"access_token":"jwt_login_token"}}}`))
		case "/api/v1/auth/api-keys":
			sawCreateKey = true
			if got := r.Header.Get("Idempotency-Key"); got != "idem-1" {
				t.Fatalf("Idempotency-Key = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"raw_key":"tmc_created_key","key":{"id":"key_1","name":"codex cli","key_prefix":"tmc_cre"}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")
	requestOut := filepath.Join(t.TempDir(), "auth-plan.json")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("secret\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--request-out", requestOut,
		"--idempotency-key", "idem-1",
		"auth", "login",
		"--email", "user@example.com",
		"--password-stdin",
		"--key-name", "codex cli",
		"--check=false",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login failed: %v stderr=%s", err, stderr.String())
	}
	if !sawCreateKey {
		t.Fatal("api key creation was not called")
	}
	raw, err := os.ReadFile(requestOut)
	if err != nil {
		t.Fatalf("read request out: %v", err)
	}
	if !bytes.Contains(raw, []byte(`/auth/login`)) || !bytes.Contains(raw, []byte(`redacted`)) {
		t.Fatalf("request out missing redacted login plan: %s", string(raw))
	}
	for _, leaked := range [][]byte{[]byte("secret"), []byte("tmc_created_key")} {
		if bytes.Contains(raw, leaked) {
			t.Fatalf("request out leaked secret %q: %s", leaked, string(raw))
		}
	}
}

func TestAuthLoginLegacyPasswordCheckFailureStoresCredentialAsUnverified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"tokens":{"access_token":"jwt_login_token"}}}`))
		case "/api/v1/auth/api-keys":
			_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"raw_key":"tmc_unverified_key","key":{"id":"key_1","name":"codex cli","key_prefix":"tmc_unv"}}}`))
		case "/api/v1/auth/me":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"code":50300,"message":{"title":"Unavailable","text":"temporary failure"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("secret\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "login",
		"--email", "user@example.com",
		"--password-stdin",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login should keep generated key as unverified instead of failing: %v stderr=%s", err, stderr.String())
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got := creds.Profiles[config.DefaultProfile].APIKey; got != "tmc_unverified_key" {
		t.Fatalf("stored api key = %q", got)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"verified":false`)) {
		t.Fatalf("stdout missing unverified marker: %s", stdout.String())
	}
}

func TestAuthImportKeyCheckFailureDoesNotStoreCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":40100,"message":{"title":"Unauthorized","text":"invalid key"}}`))
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("TMCOPILOT_HOME", home)
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("bad-key\n"))
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"auth", "import-key",
		"--api-key-stdin",
		"--check",
	})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected auth import-key check failure")
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("credentials should not be written after failed check, stat err=%v", err)
	}
}

func TestPathArgumentsAreEscaped(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
	}{
		{
			name:     "portfolio trademark get",
			args:     []string{"portfolio", "trademarks", "get", "abc/def"},
			wantPath: "/api/v1/portfolio/trademarks/abc%2Fdef",
		},
		{
			name:     "auth api key revoke",
			args:     []string{"--yes", "auth", "api-keys", "revoke", "key/1"},
			wantPath: "/api/v1/auth/api-keys/key%2F1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.EscapedPath()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"ok":true}}`))
			}))
			defer server.Close()

			t.Setenv("TMCOPILOT_HOME", t.TempDir())
			t.Setenv("TMCOPILOT_API_KEY", "test-key")

			cmd := NewRootCommand()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			args := append([]string{"--endpoint", server.URL}, tt.args...)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

func TestPortfolioImportBuildsOpenAPIRequest(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method mismatch: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/portfolio/trademarks/import" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"trademarks_imported":2}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"portfolio", "trademarks", "import",
		"--owner-name", "Nike,Adidas",
		"--organization-name", "Nike Inc.",
		"--lawyer-name", "Smith",
		"--country", "US",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	if !reflect.DeepEqual(gotBody["owner_names"], []any{"Nike", "Adidas"}) {
		t.Fatalf("owner names mismatch: %#v", gotBody["owner_names"])
	}
	if !reflect.DeepEqual(gotBody["organization_names"], []any{"Nike Inc."}) {
		t.Fatalf("organization names mismatch: %#v", gotBody["organization_names"])
	}
	if !reflect.DeepEqual(gotBody["lawyer_names"], []any{"Smith"}) || gotBody["country"] != "US" {
		t.Fatalf("body mismatch: %#v", gotBody)
	}
}

func TestPortfolioTrademarkUpdateBuildsOpenAPIRequest(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method mismatch: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/portfolio/trademarks/123" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"ok":true}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"portfolio", "trademarks", "update", "123",
		"--text", "NIKE",
		"--country", "US",
		"--trademark-format", "1",
		"--status", "10",
		"--attorney-docket-number", "ADN-001",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	if gotBody["text"] != "NIKE" || gotBody["country"] != "US" || gotBody["attorney_docket_number"] != "ADN-001" {
		t.Fatalf("body mismatch: %#v", gotBody)
	}
	if gotBody["format"] != float64(1) || gotBody["status"] != float64(10) {
		t.Fatalf("integer body mismatch: %#v", gotBody)
	}
}

func TestPortfolioTrademarkMetadataUpdateBuildsOpenAPIRequest(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method mismatch: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/portfolio/trademarks/123/metadata" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"trademark_id":123}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"portfolio", "trademarks", "metadata", "update", "123",
		"--owner-name", "Nike Inc.",
		"--attorney-name", "Sarah Chen",
		"--nice-class", "25,35",
		"--reminder-interval", "2_months,1_month",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	if gotBody["owner_name"] != "Nike Inc." || gotBody["attorney_name"] != "Sarah Chen" {
		t.Fatalf("body mismatch: %#v", gotBody)
	}
	if !reflect.DeepEqual(gotBody["nice_classes"], []any{float64(25), float64(35)}) {
		t.Fatalf("nice classes mismatch: %#v", gotBody["nice_classes"])
	}
	if !reflect.DeepEqual(gotBody["reminder_intervals"], []any{"2_months", "1_month"}) {
		t.Fatalf("reminder intervals mismatch: %#v", gotBody["reminder_intervals"])
	}
}

func TestPortfolioMonitorCommandsBuildOpenAPIRequests(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		wantBody func(t *testing.T, body map[string]any)
	}{
		{
			name:     "single update",
			args:     []string{"portfolio", "trademarks", "monitor", "update", "123", "--office-action-enable=true", "--conflict-action-enable=false"},
			wantPath: "/api/v1/portfolio/trademarks/123/monitor",
			wantBody: func(t *testing.T, body map[string]any) {
				config, ok := body["config"].(map[string]any)
				if !ok {
					t.Fatalf("config missing: %#v", body)
				}
				if config["office_action_enable"] != true || config["conflict_action_enable"] != false {
					t.Fatalf("config mismatch: %#v", config)
				}
			},
		},
		{
			name:     "batch toggle",
			args:     []string{"portfolio", "trademarks", "monitor", "batch-toggle", "--trademark-id", "123,456", "--monitor-type", "conflict", "--enable=false", "--conflict-mode", "text"},
			wantPath: "/api/v1/portfolio/trademark-monitor/toggle",
			wantBody: func(t *testing.T, body map[string]any) {
				if !reflect.DeepEqual(body["trademark_ids"], []any{"123", "456"}) {
					t.Fatalf("ids mismatch: %#v", body["trademark_ids"])
				}
				if body["monitor_type"] != "conflict" || body["enable"] != false || body["conflict_mode"] != "text" {
					t.Fatalf("body mismatch: %#v", body)
				}
			},
		},
		{
			name:     "group toggle",
			args:     []string{"portfolio", "groups", "monitor-toggle", "group/1", "--monitor-type", "office_action", "--enable=true"},
			wantPath: "/api/v1/portfolio/trademark-groups/group%2F1/monitor/toggle",
			wantBody: func(t *testing.T, body map[string]any) {
				if body["monitor_type"] != "office_action" || body["enable"] != true {
					t.Fatalf("body mismatch: %#v", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Fatalf("method mismatch: %s", r.Method)
				}
				gotPath = r.URL.EscapedPath()
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"ok":true}}`))
			}))
			defer server.Close()

			t.Setenv("TMCOPILOT_HOME", t.TempDir())
			t.Setenv("TMCOPILOT_API_KEY", "test-key")

			cmd := NewRootCommand()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			args := append([]string{"--endpoint", server.URL}, tt.args...)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tt.wantPath)
			}
			tt.wantBody(t, gotBody)
		})
	}
}

func TestPortfolioActionCommandsBuildRequests(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantMethod string
		wantPath   string
		wantQuery  map[string]string
		wantBody   func(t *testing.T, body map[string]any)
	}{
		{
			name:       "office deadlines",
			args:       []string{"portfolio", "actions", "office", "deadlines", "--limit", "5"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/portfolio/actions/office/deadlines",
			wantQuery:  map[string]string{"limit": "5"},
		},
		{
			name:       "office list",
			args:       []string{"portfolio", "actions", "office", "list", "--page", "2", "--page-size", "5", "--keyword", "nike"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/portfolio/actions/office",
			wantQuery:  map[string]string{"page": "2", "page_size": "5", "keyword": "nike"},
		},
		{
			name:       "conflict list",
			args:       []string{"portfolio", "actions", "conflict", "list", "--page", "2", "--page-size", "5", "--risk", "high"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/portfolio/actions/conflict",
			wantQuery:  map[string]string{"page": "2", "page_size": "5", "risk": "high"},
		},
		{
			name:       "cbp list",
			args:       []string{"portfolio", "actions", "cbp", "list", "--page", "2", "--page-size", "5", "--status", "active"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/portfolio/actions/cbp",
			wantQuery:  map[string]string{"page": "2", "page_size": "5", "status": "active"},
		},
		{
			name:       "conflict groups",
			args:       []string{"portfolio", "actions", "conflict", "groups", "--page", "2", "--page-size", "3", "--risk", "high", "--group-by", "mark", "--sort", "due_date", "--sort-dir", "asc"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/portfolio/actions/conflict/groups",
			wantQuery: map[string]string{
				"page":       "2",
				"page_size":  "3",
				"risk":       "high",
				"group_by":   "mark",
				"sort_field": "due_date",
				"sort_dir":   "asc",
			},
		},
		{
			name:       "office actions by trademark",
			args:       []string{"portfolio", "actions", "office", "for-trademark", "123", "--page", "4", "--page-size", "7"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/portfolio/trademarks/123/office-actions",
			wantQuery:  map[string]string{"page": "4", "page_size": "7"},
		},
		{
			name:       "conflict action get escapes IDs",
			args:       []string{"portfolio", "actions", "conflict", "get", "tm/123", "action/456"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/portfolio/trademarks/tm%2F123/conflict-actions/action%2F456",
		},
		{
			name:       "office status includes zero",
			args:       []string{"portfolio", "actions", "office", "status", "123", "456", "--status", "0", "--note", "reviewed"},
			wantMethod: http.MethodPut,
			wantPath:   "/api/v1/portfolio/trademarks/123/office-actions/456/status",
			wantBody: func(t *testing.T, body map[string]any) {
				if body["status"] != float64(0) || body["note"] != "reviewed" {
					t.Fatalf("body mismatch: %#v", body)
				}
			},
		},
		{
			name:       "cbp service requests",
			args:       []string{"portfolio", "actions", "cbp", "service-requests"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/portfolio/actions/cbp/service-requests",
		},
		{
			name:       "cbp submit service request",
			args:       []string{"portfolio", "actions", "cbp", "submit", "--request-type", "renew", "--trademark-id", "tm1", "--serial-number", "90000001", "--port-of-entry", "LAX,SFO", "--contact-email", "ops@example.com"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/portfolio/actions/cbp/service-requests",
			wantBody: func(t *testing.T, body map[string]any) {
				if body["request_type"] != "renew" || body["trademark_id"] != "tm1" || body["serial_number"] != "90000001" || body["contact_email"] != "ops@example.com" {
					t.Fatalf("body mismatch: %#v", body)
				}
				if !reflect.DeepEqual(body["ports_of_entry"], []any{"LAX", "SFO"}) {
					t.Fatalf("ports mismatch: %#v", body["ports_of_entry"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPath string
			var gotQuery url.Values
			var gotBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.EscapedPath()
				gotQuery = r.URL.Query()
				if r.Body != nil && r.ContentLength != 0 {
					if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
						t.Fatalf("decode request body: %v", err)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"ok":true}}`))
			}))
			defer server.Close()

			t.Setenv("TMCOPILOT_HOME", t.TempDir())
			t.Setenv("TMCOPILOT_API_KEY", "test-key")

			cmd := NewRootCommand()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			args := append([]string{"--endpoint", server.URL}, tt.args...)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
			}
			if gotMethod != tt.wantMethod {
				t.Fatalf("method = %q, want %q", gotMethod, tt.wantMethod)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tt.wantPath)
			}
			for key, want := range tt.wantQuery {
				if got := gotQuery.Get(key); got != want {
					t.Fatalf("query %s = %q, want %q (all query=%s)", key, got, want, gotQuery.Encode())
				}
			}
			if tt.wantBody != nil {
				tt.wantBody(t, gotBody)
			} else if gotBody != nil {
				t.Fatalf("unexpected body: %#v", gotBody)
			}
		})
	}
}

func TestPortfolioActionCommandsValidateRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "status requires status flag",
			args: []string{"portfolio", "actions", "conflict", "status", "123", "456"},
		},
		{
			name: "cbp submit requires request type",
			args: []string{"portfolio", "actions", "cbp", "submit", "--trademark-id", "tm1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			t.Setenv("TMCOPILOT_HOME", t.TempDir())
			t.Setenv("TMCOPILOT_API_KEY", "test-key")

			cmd := NewRootCommand()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			args := append([]string{"--endpoint", server.URL}, tt.args...)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil {
				t.Fatalf("command should fail")
			}
			if called {
				t.Fatalf("API should not be called")
			}
		})
	}
}

func TestPortfolioActionSchemaCommandsAreMapped(t *testing.T) {
	commands := []string{
		"portfolio actions office deadlines",
		"portfolio actions office for-trademark",
		"portfolio actions office get",
		"portfolio actions office list",
		"portfolio actions office status",
		"portfolio actions conflict groups",
		"portfolio actions conflict for-trademark",
		"portfolio actions conflict get",
		"portfolio actions conflict list",
		"portfolio actions conflict status",
		"portfolio actions cbp list",
		"portfolio actions cbp service-requests",
		"portfolio actions cbp submit",
	}
	for _, commandPath := range commands {
		t.Run(commandPath, func(t *testing.T) {
			args := append([]string{"--format", "json", "schema"}, strings.Fields(commandPath)...)
			stdout, _ := executeRootCommand(t, args)
			if !strings.Contains(stdout, `"coverage":"typed"`) {
				t.Fatalf("schema output should include typed endpoint coverage: %s", stdout)
			}
		})
	}
}

func TestPortfolioTasksCommandsAreNotExposed(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"portfolio", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("portfolio help failed: %v stderr=%s", err, stderr.String())
	}
	if strings.Contains(stdout.String(), "tasks") {
		t.Fatalf("portfolio help should not expose tasks command: %s", stdout.String())
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "portfolio", "tasks", "list"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("schema portfolio tasks list should fail after tasks commands are removed")
	}
}

func TestDryRunDoesNotCallAPIAndWritesRequestOut(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	requestOut := filepath.Join(t.TempDir(), "request.json")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--dry-run",
		"--request-out", requestOut,
		"api", "POST", "/auth/api-keys",
		"--data", `{"name":"dry"}`,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	if called {
		t.Fatal("server was called during dry-run")
	}
	raw, err := os.ReadFile(requestOut)
	if err != nil {
		t.Fatalf("read request out: %v", err)
	}
	if !bytes.Contains(raw, []byte(`"method": "POST"`)) || !bytes.Contains(raw, []byte(`"path": "/auth/api-keys"`)) {
		t.Fatalf("request out mismatch: %s", raw)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"dry"`)) {
		t.Fatalf("stdout missing dry-run body: %s", stdout.String())
	}
}

func TestDestructiveRequestRequiresYes(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", "https://api.example.test", "api", "DELETE", "/gap-analyses/id-1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("DELETE without --yes error = nil")
	}
	if !strings.Contains(stderr.String(), `"type":"cli_error"`) {
		t.Fatalf("stderr missing typed cli error: %s", stderr.String())
	}
}

func TestAPICatalogFiltersEndpoints(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"api", "catalog", "--tag", "auth", "--coverage", "typed"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`/auth/me`)) {
		t.Fatalf("catalog output missing auth typed endpoint: %s", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte(`/competitors`)) {
		t.Fatalf("catalog output includes non-auth endpoint: %s", stdout.String())
	}
}

func TestSchemaShowsEndpointMetadata(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "search", "trademarks"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema failed: %v stderr=%s", err, stderr.String())
	}
	for _, want := range []string{
		`"command":"tmc search trademarks"`,
		`"path":"/trademark/search"`,
		`"--class"`,
	} {
		if !bytes.Contains(stdout.Bytes(), []byte(want)) {
			t.Fatalf("schema output missing %q: %s", want, stdout.String())
		}
	}
	if bytes.Contains(stdout.Bytes(), []byte(`"definitions"`)) {
		t.Fatalf("schema output should not include raw OpenAPI definitions by default: %s", stdout.String())
	}
}

func TestSchemaIncludesAgentSafetyMetadata(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "search", "trademarks"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema failed: %v stderr=%s", err, stderr.String())
	}
	var searchSchema struct {
		OK   bool `json:"ok"`
		Data struct {
			Safety struct {
				ReadOnly    bool `json:"read_only"`
				SideEffect  bool `json:"side_effect"`
				Destructive bool `json:"destructive"`
			} `json:"safety"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &searchSchema); err != nil {
		t.Fatalf("decode schema: %v output=%s", err, stdout.String())
	}
	if !searchSchema.Data.Safety.ReadOnly || searchSchema.Data.Safety.SideEffect || searchSchema.Data.Safety.Destructive {
		t.Fatalf("search safety metadata mismatch: %#v", searchSchema.Data.Safety)
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "gap", "delete"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete schema failed: %v stderr=%s", err, stderr.String())
	}
	var deleteSchema struct {
		OK   bool `json:"ok"`
		Data struct {
			Safety struct {
				SideEffect  bool `json:"side_effect"`
				Destructive bool `json:"destructive"`
				RequiresYes bool `json:"requires_yes"`
			} `json:"safety"`
			Examples []string `json:"examples"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &deleteSchema); err != nil {
		t.Fatalf("decode delete schema: %v output=%s", err, stdout.String())
	}
	if !deleteSchema.Data.Safety.SideEffect || !deleteSchema.Data.Safety.Destructive || !deleteSchema.Data.Safety.RequiresYes {
		t.Fatalf("delete safety metadata mismatch: %#v", deleteSchema.Data.Safety)
	}
	if len(deleteSchema.Data.Examples) == 0 {
		t.Fatalf("delete schema missing examples: %s", stdout.String())
	}
}

func TestSchemaIncludesPaginationMetadata(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "portfolio", "trademarks", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema failed: %v stderr=%s", err, stderr.String())
	}
	var schema struct {
		OK   bool `json:"ok"`
		Data struct {
			Pagination struct {
				SupportsPageAll   bool     `json:"supports_page_all"`
				SupportsFields    bool     `json:"supports_fields"`
				SupportsManifest  bool     `json:"supports_manifest"`
				RecommendedFormat string   `json:"recommended_format"`
				Flags             []string `json:"flags"`
			} `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("decode schema: %v output=%s", err, stdout.String())
	}
	if !schema.Data.Pagination.SupportsPageAll || !schema.Data.Pagination.SupportsFields || !schema.Data.Pagination.SupportsManifest {
		t.Fatalf("pagination metadata mismatch: %#v", schema.Data.Pagination)
	}
	if schema.Data.Pagination.RecommendedFormat != "ndjson" {
		t.Fatalf("recommended format = %q", schema.Data.Pagination.RecommendedFormat)
	}
}

func TestPortfolioMutationSchemasExposeTypedSafety(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "portfolio", "trademarks", "import"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema failed: %v stderr=%s", err, stderr.String())
	}
	var schema struct {
		Data struct {
			Endpoint struct {
				Coverage string `json:"coverage"`
				Path     string `json:"path"`
			} `json:"endpoint"`
			Safety struct {
				ReadOnly   bool `json:"read_only"`
				SideEffect bool `json:"side_effect"`
			} `json:"safety"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("decode schema: %v output=%s", err, stdout.String())
	}
	if schema.Data.Endpoint.Path != "/portfolio/trademarks/import" || schema.Data.Endpoint.Coverage != "typed" {
		t.Fatalf("endpoint metadata mismatch: %#v", schema.Data.Endpoint)
	}
	if schema.Data.Safety.ReadOnly || !schema.Data.Safety.SideEffect {
		t.Fatalf("import safety mismatch: %#v", schema.Data.Safety)
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "portfolio", "trademarks", "import-preview"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("preview schema failed: %v stderr=%s", err, stderr.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("decode preview schema: %v output=%s", err, stdout.String())
	}
	if !schema.Data.Safety.ReadOnly || schema.Data.Safety.SideEffect {
		t.Fatalf("preview safety mismatch: %#v", schema.Data.Safety)
	}
}

func TestAgentBootstrapReturnsMachineReadableGuidance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization header mismatch: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"email":"agent@example.com"}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", server.URL, "agent", "bootstrap", "--check"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent bootstrap failed: %v stderr=%s", err, stderr.String())
	}
	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			CLI struct {
				Commands []string `json:"commands"`
			} `json:"cli"`
			Auth struct {
				Configured bool `json:"configured"`
				Verified   bool `json:"verified"`
			} `json:"auth"`
			Skills []struct {
				Name string `json:"name"`
			} `json:"skills"`
			Discovery struct {
				Schema string `json:"schema"`
			} `json:"discovery"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode bootstrap: %v output=%s", err, stdout.String())
	}
	if !result.Data.Auth.Configured || !result.Data.Auth.Verified {
		t.Fatalf("auth metadata mismatch: %#v", result.Data.Auth)
	}
	if !reflect.DeepEqual(result.Data.CLI.Commands, []string{"tmc", "tmcopilot"}) {
		t.Fatalf("commands = %#v", result.Data.CLI.Commands)
	}
	if len(result.Data.Skills) == 0 {
		t.Fatalf("bootstrap missing skills: %s", stdout.String())
	}
	if result.Data.Discovery.Schema != "tmc schema <command...>" {
		t.Fatalf("schema discovery = %q", result.Data.Discovery.Schema)
	}
	if bytes.Contains(stdout.Bytes(), []byte("agent@example.com")) {
		t.Fatalf("bootstrap leaked user profile data: %s", stdout.String())
	}
}

func TestSchemaOpenAPIFlagIncludesRawDefinitions(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "--openapi", "search", "trademarks"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema --openapi failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"definitions"`)) {
		t.Fatalf("schema --openapi output missing definitions: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"internal_protocol_rest_handler.searchByTextRequest"`)) {
		t.Fatalf("schema --openapi output missing request definition: %s", stdout.String())
	}
}

func TestAPIEndpointSchemaShowsRawEndpointMetadata(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"api", "schema", "POST", "/trademark/search"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("api schema failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"path":"/trademark/search"`)) {
		t.Fatalf("api schema output missing endpoint: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"internal_protocol_rest_handler.searchByTextRequest"`)) {
		t.Fatalf("api schema output missing definition: %s", stdout.String())
	}
}

func TestSchemaRejectsRawEndpointFormWithHint(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "POST", "/trademark/search"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected raw endpoint schema form to fail")
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`tmc schema expects a CLI command path`)) {
		t.Fatalf("stderr missing command-path hint: %s", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`tmc api schema POST /trademark/search`)) {
		t.Fatalf("stderr missing api schema hint: %s", stderr.String())
	}
}

func TestSchemaAuthLogoutIsLocalOnly(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "auth", "logout"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema auth logout failed: %v stderr=%s", err, stderr.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte(`/auth/logout`)) {
		t.Fatalf("local logout schema should not expose backend logout endpoint: %s", stdout.String())
	}
}

func TestSearchAliasesBuildRequests(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dry-run", "search", "trademark", "--name", "Nike", "--class", "25"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("alias dry-run failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"/trademark/search"`)) {
		t.Fatalf("alias dry-run output missing endpoint: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"class":["25"]`)) {
		t.Fatalf("alias dry-run output missing class body: %s", stdout.String())
	}
}

func TestExecuteUnknownCommandWritesStructuredSuggestion(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{"search", "trademarkz", "--name", "Nike"}, nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`"type":"validation_error"`)) {
		t.Fatalf("stderr missing validation error: %s", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`trademarks`)) {
		t.Fatalf("stderr missing suggestion: %s", stderr.String())
	}
}

func TestExecuteUnknownFlagWritesStructuredSuggestion(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{"search", "trademarks", "--clas", "25"}, nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`"type":"validation_error"`)) {
		t.Fatalf("stderr missing validation error: %s", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`--class`)) {
		t.Fatalf("stderr missing flag suggestion: %s", stderr.String())
	}
}

func TestAPIDownloadWritesRawResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test"); got != "yes" {
			t.Fatalf("X-Test header = %q", got)
		}
		_, _ = w.Write([]byte("raw-body"))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")
	outFile := filepath.Join(t.TempDir(), "downloads", "body.txt")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--output", outFile,
		"api", "download", "GET", "/files/raw",
		"--header", "X-Test=yes",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	raw, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(raw) != "raw-body" {
		t.Fatalf("download content = %q", raw)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"bytes":8`)) {
		t.Fatalf("summary missing bytes: %s", stdout.String())
	}
}

func TestCommonLawSearchBuildsOpenAPIRequest(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method mismatch: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/common-law/search/social/handle" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":[]}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"common-law", "search", "social-handle",
		"--name", "Nike,Adidas",
		"--platform", "instagram",
		"--collaboration-id", "collab_1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	if !reflect.DeepEqual(gotBody["name"], []any{"Nike", "Adidas"}) {
		t.Fatalf("name body mismatch: %#v", gotBody["name"])
	}
	if gotBody["platform"] != "instagram" || gotBody["collaboration_id"] != "collab_1" {
		t.Fatalf("body mismatch: %#v", gotBody)
	}
}

func TestDomainSearchBuildsOpenAPIRequest(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method mismatch: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/domain/search" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"total":0}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", server.URL, "domain", "search", "--keyword", "nike", "--limit", "25"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	if gotBody["keyword"] != "nike" || gotBody["limit"] != float64(25) {
		t.Fatalf("body mismatch: %#v", gotBody)
	}
}

func TestTrademarkImageCreateBuildsOpenAPIRequest(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method mismatch: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/trademark/image/task" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":{"title":"OK","text":"ok"},"data":{"id":"task_1"}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"search", "image", "create",
		"--bucket", "tmc-images",
		"--key", "uploads/mark.png",
		"--cloudfront-url", "https://cdn.example.test/mark.png",
		"--country", "US,CA",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	if gotBody["bucket"] != "tmc-images" || gotBody["key"] != "uploads/mark.png" {
		t.Fatalf("body mismatch: %#v", gotBody)
	}
	if !reflect.DeepEqual(gotBody["countries"], []any{"US", "CA"}) {
		t.Fatalf("countries body mismatch: %#v", gotBody["countries"])
	}
}

func TestUSPTOOfficeActionDocumentDownloadsRawResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method mismatch: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/trademark/office-action/uspto/document" {
			t.Fatalf("path mismatch: %s", r.URL.Path)
		}
		query := r.URL.Query()
		for key, want := range map[string]string{
			"serial_number":    "97346091",
			"document_page_id": "DOC123",
			"document_type":    "OOA",
			"document_date":    "2024-01-02",
		} {
			if got := query.Get(key); got != want {
				t.Fatalf("query %s = %q, want %q", key, got, want)
			}
		}
		_, _ = w.Write([]byte("pdf-bytes"))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "test-key")
	outFile := filepath.Join(t.TempDir(), "doc.pdf")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--endpoint", server.URL,
		"--output", outFile,
		"search", "uspto-document",
		"--serial-number", "97346091",
		"--document-page-id", "DOC123",
		"--document-type", "OOA",
		"--document-date", "2024-01-02",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v stderr=%s", err, stderr.String())
	}
	raw, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(raw) != "pdf-bytes" {
		t.Fatalf("download content = %q", raw)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"bytes":9`)) {
		t.Fatalf("summary missing bytes: %s", stdout.String())
	}
}

func TestNewSearchSchemasExposeSafetyAndTypedCoverage(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "search", "image", "create"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema failed: %v stderr=%s", err, stderr.String())
	}
	var imageSchema struct {
		Data struct {
			Endpoint struct {
				Coverage string `json:"coverage"`
				Path     string `json:"path"`
			} `json:"endpoint"`
			Safety struct {
				ReadOnly   bool `json:"read_only"`
				SideEffect bool `json:"side_effect"`
			} `json:"safety"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &imageSchema); err != nil {
		t.Fatalf("decode schema: %v output=%s", err, stdout.String())
	}
	if imageSchema.Data.Endpoint.Path != "/trademark/image/task" || imageSchema.Data.Endpoint.Coverage != "typed" {
		t.Fatalf("endpoint metadata mismatch: %#v", imageSchema.Data.Endpoint)
	}
	if imageSchema.Data.Safety.ReadOnly || !imageSchema.Data.Safety.SideEffect {
		t.Fatalf("image create safety mismatch: %#v", imageSchema.Data.Safety)
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "ttab", "search"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ttab schema failed: %v stderr=%s", err, stderr.String())
	}
	var ttabSchema struct {
		Data struct {
			Endpoint struct {
				Coverage string `json:"coverage"`
				Path     string `json:"path"`
			} `json:"endpoint"`
			Safety struct {
				ReadOnly   bool `json:"read_only"`
				SideEffect bool `json:"side_effect"`
			} `json:"safety"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &ttabSchema); err != nil {
		t.Fatalf("decode ttab schema: %v output=%s", err, stdout.String())
	}
	if ttabSchema.Data.Endpoint.Path != "/trademark/ttab/search" || ttabSchema.Data.Endpoint.Coverage != "typed" {
		t.Fatalf("ttab endpoint metadata mismatch: %#v", ttabSchema.Data.Endpoint)
	}
	if !ttabSchema.Data.Safety.ReadOnly || ttabSchema.Data.Safety.SideEffect {
		t.Fatalf("ttab safety mismatch: %#v", ttabSchema.Data.Safety)
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "lawsuits", "search"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("lawsuit schema failed: %v stderr=%s", err, stderr.String())
	}
	var lawsuitSchema struct {
		Data struct {
			Endpoint struct {
				Coverage string `json:"coverage"`
				Path     string `json:"path"`
			} `json:"endpoint"`
			Safety struct {
				ReadOnly   bool `json:"read_only"`
				SideEffect bool `json:"side_effect"`
			} `json:"safety"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &lawsuitSchema); err != nil {
		t.Fatalf("decode lawsuit schema: %v output=%s", err, stdout.String())
	}
	if lawsuitSchema.Data.Endpoint.Path != "/trademark/wide-table/lawsuits" || lawsuitSchema.Data.Endpoint.Coverage != "typed" {
		t.Fatalf("lawsuit endpoint metadata mismatch: %#v", lawsuitSchema.Data.Endpoint)
	}
	if !lawsuitSchema.Data.Safety.ReadOnly || lawsuitSchema.Data.Safety.SideEffect {
		t.Fatalf("lawsuit safety mismatch: %#v", lawsuitSchema.Data.Safety)
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"schema", "lawyers", "trademarks"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("lawyer schema failed: %v stderr=%s", err, stderr.String())
	}
	var lawyerSchema struct {
		Data struct {
			Endpoint struct {
				Coverage string `json:"coverage"`
				Path     string `json:"path"`
			} `json:"endpoint"`
			Safety struct {
				ReadOnly   bool `json:"read_only"`
				SideEffect bool `json:"side_effect"`
			} `json:"safety"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &lawyerSchema); err != nil {
		t.Fatalf("decode lawyer schema: %v output=%s", err, stdout.String())
	}
	if lawyerSchema.Data.Endpoint.Path != "/trademark/wide-table/lawyers/{graphId}/trademarks" || lawyerSchema.Data.Endpoint.Coverage != "typed" {
		t.Fatalf("lawyer endpoint metadata mismatch: %#v", lawyerSchema.Data.Endpoint)
	}
	if !lawyerSchema.Data.Safety.ReadOnly || lawyerSchema.Data.Safety.SideEffect {
		t.Fatalf("lawyer safety mismatch: %#v", lawyerSchema.Data.Safety)
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"api", "catalog", "--coverage", "typed", "--search", "common-law"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("catalog failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`/common-law/search/app-store`)) {
		t.Fatalf("catalog missing common-law typed endpoint: %s", stdout.String())
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"api", "catalog", "--coverage", "typed", "--search", "lawsuit"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("lawsuit catalog failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`/trademark/wide-table/lawsuits`)) {
		t.Fatalf("catalog missing lawsuit typed endpoint: %s", stdout.String())
	}

	cmd = NewRootCommand()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"api", "catalog", "--coverage", "typed", "--search", "lawyer"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("lawyer catalog failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`/trademark/wide-table/lawyers/{graphId}/trademarks`)) {
		t.Fatalf("catalog missing lawyer typed endpoint: %s", stdout.String())
	}
}

func TestSkillsListAndRead(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	listCmd := NewRootCommand()
	var listOut, listErr bytes.Buffer
	listCmd.SetOut(&listOut)
	listCmd.SetErr(&listErr)
	listCmd.SetArgs([]string{"skills", "list"})
	if err := listCmd.Execute(); err != nil {
		t.Fatalf("skills list failed: %v stderr=%s", err, listErr.String())
	}
	if !bytes.Contains(listOut.Bytes(), []byte(`tmc-trademark-search`)) {
		t.Fatalf("skills list missing trademark skill: %s", listOut.String())
	}

	readCmd := NewRootCommand()
	var readOut, readErr bytes.Buffer
	readCmd.SetOut(&readOut)
	readCmd.SetErr(&readErr)
	readCmd.SetArgs([]string{"skills", "read", "tmc-trademark-search"})
	if err := readCmd.Execute(); err != nil {
		t.Fatalf("skills read failed: %v stderr=%s", err, readErr.String())
	}
	if !bytes.Contains(readOut.Bytes(), []byte("tmc search trademarks")) {
		t.Fatalf("skills read missing command guidance: %s", readOut.String())
	}
}

func TestSkillsReadReferenceAsJSON(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"skills", "read", "tmc-trademark-search/references/search-fields.md", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("skills read reference failed: %v stderr=%s", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"path":"references/search-fields.md"`)) {
		t.Fatalf("json output missing reference path: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`--class 25,35`)) {
		t.Fatalf("json output missing reference content: %s", stdout.String())
	}
}

func TestSkillsReadSupportsOutputFile(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	outFile := filepath.Join(t.TempDir(), "skill.md")
	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--output", outFile, "skills", "read", "tmc-trademark-search"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("skills read output failed: %v stderr=%s", err, stderr.String())
	}
	raw, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !bytes.Contains(raw, []byte("tmc search trademarks")) {
		t.Fatalf("output file missing skill content: %s", string(raw))
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"bytes":`)) {
		t.Fatalf("stdout missing output summary: %s", stdout.String())
	}
}

func TestSkillsRejectsInvalidTarget(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"skills", "read", "../tmc-shared"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected invalid skill target to fail")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("invalid skill name")) {
		t.Fatalf("stderr missing invalid name error: %s", stderr.String())
	}
}

func TestDoctorAuthFailsWithoutAPIKey(t *testing.T) {
	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")
	t.Setenv("TMC_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "auth"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("doctor auth error = nil")
	}
	if !strings.Contains(err.Error(), "doctor auth failed") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"ok":false`) {
		t.Fatalf("stdout missing failed auth result: %s", stdout.String())
	}
}

func TestDoctorNetworkFailsOnBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":50000,"message":{"title":"Error","text":"broken"}}`))
	}))
	defer server.Close()

	t.Setenv("TMCOPILOT_HOME", t.TempDir())
	t.Setenv("TMCOPILOT_API_KEY", "")

	cmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--endpoint", server.URL, "doctor", "network"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("doctor network error = nil")
	}
	if !strings.Contains(err.Error(), "doctor network failed") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"network":{"message":"http 500`) {
		t.Fatalf("stdout missing failed network result: %s", stdout.String())
	}
}
