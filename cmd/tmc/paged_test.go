package tmc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/client"
)

func TestBuildListQueryUsesPagingSortAndFilters(t *testing.T) {
	opts := &listOptions{
		page:         -1,
		pageSize:     0,
		params:       []string{"owner=nike"},
		sortField:    "mark",
		sortDir:      "desc",
		sortParam:    "sort_field",
		sortDirParam: "sort_dir",
		filters: []queryFlag{
			{param: "include_archived", value: "true"},
			{param: "empty", value: " "},
		},
	}

	query, err := buildListQuery(opts)
	if err != nil {
		t.Fatalf("buildListQuery returned error: %v", err)
	}

	want := url.Values{
		"owner":            []string{"nike"},
		"page":             []string{"1"},
		"page_size":        []string{"20"},
		"sort_field":       []string{"mark"},
		"sort_dir":         []string{"desc"},
		"include_archived": []string{"true"},
	}
	if !reflect.DeepEqual(query, want) {
		t.Fatalf("query mismatch\nwant: %#v\n got: %#v", want, query)
	}
}

func TestStreamPagesRequestsEachPageAndProjectsFieldsThroughPublicPath(t *testing.T) {
	var requestedPages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		requestedPages = append(requestedPages, page)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"message": {"title": "OK", "text": "ok"},
			"data": {
				"items": [{"id": "id-` + page + `", "serial_number": "US-TM-8841869` + page + `", "name": "mark-` + page + `", "hidden": "drop"}],
				"total": 3,
				"page": ` + page + `,
				"page_size": 1,
				"total_pages": 3
			}
		}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	rt := &runtimeContext{
		Client: client.New(server.URL, "test-key", "", "test", time.Second),
		Format: "ndjson",
	}
	opts := &listOptions{page: 1, pageSize: 1, maxPages: 2, fields: "id,serial_number"}
	query := url.Values{"page_size": []string{"1"}}

	if err := streamPages(cmd, rt, "/portfolio/trademarks/search", query, opts); err != nil {
		t.Fatalf("streamPages returned error: %v", err)
	}
	if !reflect.DeepEqual(requestedPages, []string{"1", "2"}) {
		t.Fatalf("requested pages mismatch: %#v", requestedPages)
	}

	scanner := bufio.NewScanner(&out)
	var rows []map[string]any
	for scanner.Scan() {
		var row map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatalf("invalid ndjson row %q: %v", scanner.Text(), err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan output: %v", err)
	}
	wantRows := []map[string]any{
		{"id": "id-1", "serial_number": "88418691"},
		{"id": "id-2", "serial_number": "88418692"},
	}
	if !reflect.DeepEqual(rows, wantRows) {
		t.Fatalf("rows mismatch\nwant: %#v\n got: %#v", wantRows, rows)
	}
}

func TestStreamPagesCreatesNestedOutputFileAndWritesSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"message": {"title": "OK", "text": "ok"},
			"data": {
				"items": [{"id": "id-1", "name": "mark-1"}],
				"total": 1,
				"page": 1,
				"page_size": 1,
				"total_pages": 1
			}
		}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "exports", "rows.ndjson")
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	rt := &runtimeContext{
		Client:     client.New(server.URL, "test-key", "", "test", time.Second),
		Format:     "ndjson",
		OutputPath: outputPath,
	}
	opts := &listOptions{page: 1, pageSize: 1, fields: "id"}

	if err := streamPages(cmd, rt, "/portfolio/trademarks/search", url.Values{}, opts); err != nil {
		t.Fatalf("streamPages returned error: %v", err)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}
	if string(raw) != "{\"id\":\"id-1\"}\n" {
		t.Fatalf("export content = %q", raw)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("stat export file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("export mode = %v, want 0600", got)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"path":"`+outputPath+`"`)) {
		t.Fatalf("summary missing output path: %s", out.String())
	}
}

func TestStreamPagesHonorsMaxRowsAndWritesManifest(t *testing.T) {
	var requestedPages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		requestedPages = append(requestedPages, page)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"message": {"title": "OK", "text": "ok"},
			"data": {
				"items": [{"id": "id-` + page + `a"}, {"id": "id-` + page + `b"}],
				"total": 4,
				"page": ` + page + `,
				"page_size": 2,
				"total_pages": 2
			}
		}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "rows.ndjson")
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	rt := &runtimeContext{
		Client:     client.New(server.URL, "test-key", "", "test", time.Second),
		Format:     "ndjson",
		OutputPath: outputPath,
	}
	opts := &listOptions{page: 1, pageSize: 2, maxRows: 1, manifestPath: manifestPath, fields: "id"}

	if err := streamPages(cmd, rt, "/portfolio/trademarks/search", url.Values{}, opts); err != nil {
		t.Fatalf("streamPages returned error: %v", err)
	}
	if !reflect.DeepEqual(requestedPages, []string{"1"}) {
		t.Fatalf("requested pages mismatch: %#v", requestedPages)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}
	if string(raw) != "{\"id\":\"id-1a\"}\n" {
		t.Fatalf("export content = %q", raw)
	}
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !bytes.Contains(manifest, []byte(`"rows": 1`)) {
		t.Fatalf("manifest missing row count: %s", manifest)
	}
}

func TestSplitStringValuesAndIntSlice(t *testing.T) {
	got := splitStringValues([]string{"a,b", " c ", "", "d"})
	if !reflect.DeepEqual(got, []string{"a", "b", "c", "d"}) {
		t.Fatalf("split values mismatch: %#v", got)
	}

	ints, err := splitIntValues([]string{"1,2", "3"}, "ids")
	if err != nil {
		t.Fatalf("splitIntValues returned error: %v", err)
	}
	if !reflect.DeepEqual(ints, []int{1, 2, 3}) {
		t.Fatalf("int slice mismatch: %#v", ints)
	}
}
