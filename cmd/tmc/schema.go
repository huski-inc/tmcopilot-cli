package tmc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

type commandEndpointSpec struct {
	Method string
	Path   string
}

type commandSchema struct {
	Command       string                  `json:"command"`
	Kind          string                  `json:"kind,omitempty"`
	Aliases       []string                `json:"aliases,omitempty"`
	Short         string                  `json:"short,omitempty"`
	Use           string                  `json:"use,omitempty"`
	Flags         []flagSchema            `json:"flags,omitempty"`
	GlobalFlags   []flagSchema            `json:"global_flags,omitempty"`
	Endpoint      *openapi.Endpoint       `json:"endpoint,omitempty"`
	OpenAPI       *openapi.EndpointSchema `json:"openapi,omitempty"`
	Safety        *commandSafety          `json:"safety,omitempty"`
	Pagination    *commandPagination      `json:"pagination,omitempty"`
	Examples      []string                `json:"examples,omitempty"`
	Children      []commandListItem       `json:"children,omitempty"`
	SchemaCommand string                  `json:"schema_command,omitempty"`
}

type commandListItem struct {
	Command       string             `json:"command"`
	Kind          string             `json:"kind,omitempty"`
	Aliases       []string           `json:"aliases,omitempty"`
	Short         string             `json:"short,omitempty"`
	Endpoint      *openapi.Endpoint  `json:"endpoint,omitempty"`
	Safety        *commandSafety     `json:"safety,omitempty"`
	Pagination    *commandPagination `json:"pagination,omitempty"`
	SchemaCommand string             `json:"schema_command,omitempty"`
}

type commandSafety struct {
	AuthRequired   bool   `json:"auth_required"`
	ReadOnly       bool   `json:"read_only"`
	SideEffect     bool   `json:"side_effect"`
	Destructive    bool   `json:"destructive"`
	SupportsDryRun bool   `json:"supports_dry_run"`
	RequiresYes    bool   `json:"requires_yes"`
	Hint           string `json:"hint,omitempty"`
}

type commandPagination struct {
	SupportsPageAll   bool     `json:"supports_page_all"`
	SupportsFields    bool     `json:"supports_fields"`
	SupportsManifest  bool     `json:"supports_manifest"`
	RecommendedFormat string   `json:"recommended_format,omitempty"`
	Flags             []string `json:"flags,omitempty"`
}

type flagSchema struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type,omitempty"`
	Default   string `json:"default,omitempty"`
	Usage     string `json:"usage,omitempty"`
}

func newSchemaCommand(opts *globalOptions) *cobra.Command {
	var includeOpenAPI bool
	cmd := &cobra.Command{
		Use:   "schema [command...]",
		Short: "View CLI command schema and related Swagger metadata",
		Long: `View schema for a TMCopilot CLI command.

This command is command-first: use CLI paths such as "search trademarks" or
"portfolio trademarks list". Use "tmc api schema METHOD /path" only for raw
Open API fallback and debugging.`,
		Example: `  tmc schema
  tmc schema search trademarks
  tmc schema search companies
  tmc schema portfolio trademarks list
  tmc schema gap create`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				if len(args) == 0 {
					items := listCommandSchemas(cmd.Root(), "", false)
					return writeResult(rt, map[string]any{
						"count":    len(items),
						"commands": items,
					}, nil)
				}
				if isHTTPMethodArg(args[0]) {
					return fmt.Errorf("tmc schema expects a CLI command path such as `tmc schema search trademarks`; use `tmc api schema %s %s` for raw endpoint schema", strings.ToUpper(args[0]), strings.Join(args[1:], " "))
				}
				target, err := resolveSchemaCommandTarget(cmd.Root(), args)
				if err != nil {
					return err
				}
				result := buildCommandSchema(target, includeOpenAPI)
				return writeResult(rt, result, nil)
			})
		},
	}
	cmd.Flags().BoolVar(&includeOpenAPI, "openapi", false, "include raw Swagger parameters, responses, and referenced definitions")
	return cmd
}

func newAPIEndpointSchemaCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "schema <method> <path>",
		Short: "Show raw Swagger schema for one endpoint",
		Long:  "Show raw Swagger parameters, responses, and referenced definitions for one Open API endpoint. Prefer `tmc schema <command...>` for normal agent workflows.",
		Example: `  tmc api schema POST /trademark/search
  tmc api schema GET /portfolio/trademarks/search`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				schema, err := endpointSchemaFor(args[0], args[1])
				if err != nil {
					return err
				}
				return writeResult(rt, schema, map[string]any{
					"source_hash": openapi.SourceHash,
					"source_path": openapi.SourcePath,
				})
			})
		},
	}
}

func resolveSchemaCommandTarget(root *cobra.Command, args []string) (*cobra.Command, error) {
	current := root
	for i, arg := range args {
		child := findChildCommand(current, arg)
		if child == nil {
			return nil, fmt.Errorf("unknown CLI command path %q at %q; run `tmc schema` to list command schemas", strings.Join(args, " "), strings.Join(args[:i+1], " "))
		}
		current = child
	}
	return current, nil
}

func buildCommandSchema(cmd *cobra.Command, includeOpenAPI bool) commandSchema {
	key := commandKey(cmd)
	result := commandSchema{
		Command:       "tmc " + key,
		Kind:          commandKindFor(key),
		Aliases:       commandAliases(cmd),
		Short:         cmd.Short,
		Use:           cmd.UseLine(),
		Flags:         collectFlags(cmd.Flags()),
		GlobalFlags:   collectFlags(cmd.Root().PersistentFlags()),
		Children:      listChildCommandSchemas(cmd),
		SchemaCommand: "tmc schema " + key,
	}
	result.Safety = localCommandSafetyFor(key, cmd)
	if spec, ok := commandEndpointSpecs[key]; ok {
		if schema, err := endpointSchemaFor(spec.Method, spec.Path); err == nil {
			endpoint := schema.Endpoint
			result.Kind = "endpoint"
			result.Endpoint = &endpoint
			result.Safety = commandSafetyFor(key, spec)
			if includeOpenAPI {
				result.OpenAPI = &schema
			}
		}
	}
	result.Pagination = commandPaginationFor(cmd)
	result.Examples = commandExamplesFor(cmd, key, result.Safety, result.Pagination)
	return result
}

func listCommandSchemas(root *cobra.Command, prefix string, includeExact bool) []commandListItem {
	items := []commandListItem{}
	walkRunnableCommands(root, func(cmd *cobra.Command) {
		key := commandKey(cmd)
		if key == "" {
			return
		}
		if prefix != "" {
			if key == prefix && !includeExact {
				return
			}
			if key != prefix && !strings.HasPrefix(key, prefix+" ") {
				return
			}
		}
		items = append(items, buildCommandListItem(cmd, key))
	})
	sortCommandList(items)
	return items
}

func listChildCommandSchemas(cmd *cobra.Command) []commandListItem {
	prefix := commandKey(cmd)
	if prefix == "" {
		return nil
	}
	items := listCommandSchemas(cmd.Root(), prefix, false)
	if len(items) > 0 {
		return items
	}
	for _, child := range cmd.Commands() {
		if child.Hidden || child.Name() == "help" || child.Name() == "completion" {
			continue
		}
		key := commandKey(child)
		items = append(items, commandListItem{
			Command:       "tmc " + key,
			Kind:          commandKindFor(key),
			Aliases:       commandAliases(child),
			Short:         child.Short,
			Safety:        localCommandSafetyFor(key, child),
			Pagination:    commandPaginationFor(child),
			SchemaCommand: "tmc schema " + key,
		})
	}
	sortCommandList(items)
	return items
}

func walkRunnableCommands(cmd *cobra.Command, visit func(*cobra.Command)) {
	if cmd == nil {
		return
	}
	for _, child := range cmd.Commands() {
		if shouldSkipSchemaCommand(child) {
			continue
		}
		if child.Runnable() {
			visit(child)
		}
		walkRunnableCommands(child, visit)
	}
}

func shouldSkipSchemaCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return true
	}
	if cmd.Hidden {
		return true
	}
	switch cmd.Name() {
	case "completion", "help":
		return true
	default:
		return false
	}
}

func buildCommandListItem(cmd *cobra.Command, key string) commandListItem {
	item := commandListItem{
		Command:       "tmc " + key,
		Kind:          commandKindFor(key),
		Aliases:       commandAliases(cmd),
		Short:         cmd.Short,
		Safety:        localCommandSafetyFor(key, cmd),
		Pagination:    commandPaginationFor(cmd),
		SchemaCommand: "tmc schema " + key,
	}
	if spec, ok := commandEndpointSpecs[key]; ok {
		item.Kind = "endpoint"
		if endpoint, ok := openapi.FindEndpoint(spec.Method, spec.Path); ok {
			item.Endpoint = &endpoint
		}
		item.Safety = commandSafetyFor(key, spec)
	}
	return item
}

func commandKindFor(key string) string {
	if _, ok := commandEndpointSpecs[key]; ok {
		return "endpoint"
	}
	return "local"
}

func sortCommandList(items []commandListItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Command < items[j].Command
	})
}

func endpointSchemaFor(method string, path string) (openapi.EndpointSchema, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizeCatalogPath(path)
	schema, ok := openapi.FindEndpointSchema(method, path)
	if !ok {
		return openapi.EndpointSchema{}, fmt.Errorf("schema not found in catalog: %s %s", method, path)
	}
	if openapi.IsInternalEndpoint(schema.Endpoint) {
		return openapi.EndpointSchema{}, fmt.Errorf("schema not found in catalog")
	}
	return schema, nil
}

func commandKey(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	path := strings.TrimSpace(cmd.CommandPath())
	path = strings.TrimPrefix(path, "tmc")
	return strings.TrimSpace(path)
}

func commandAliases(cmd *cobra.Command) []string {
	if cmd == nil || len(cmd.Aliases) == 0 {
		return nil
	}
	parent := strings.TrimSpace(strings.TrimPrefix(cmd.Parent().CommandPath(), "tmc"))
	aliases := make([]string, 0, len(cmd.Aliases))
	for _, alias := range cmd.Aliases {
		if parent == "" {
			aliases = append(aliases, "tmc "+alias)
		} else {
			aliases = append(aliases, "tmc "+parent+" "+alias)
		}
	}
	sort.Strings(aliases)
	return aliases
}

func collectFlags(flags *pflag.FlagSet) []flagSchema {
	if flags == nil {
		return nil
	}
	out := []flagSchema{}
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag == nil || flag.Hidden {
			return
		}
		out = append(out, flagSchema{
			Name:      "--" + flag.Name,
			Shorthand: shorthandFlag(flag),
			Type:      flag.Value.Type(),
			Default:   flag.DefValue,
			Usage:     flag.Usage,
		})
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func shorthandFlag(flag *pflag.Flag) string {
	if flag.Shorthand == "" {
		return ""
	}
	return "-" + flag.Shorthand
}

func isHTTPMethodArg(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func commandSafetyFor(key string, spec commandEndpointSpec) *commandSafety {
	method := strings.ToUpper(strings.TrimSpace(spec.Method))
	destructive := isDestructiveRequest(method, spec.Path)
	sideEffect := destructive || isWriteMethod(method) && !isReadLikeCommand(key, spec)
	hint := "safe to run after checking required flags"
	if sideEffect {
		hint = "use --dry-run --request-out before running when the user has not confirmed inputs"
	}
	if destructive {
		hint = "requires --yes unless using --dry-run"
	}
	return &commandSafety{
		AuthRequired:   true,
		ReadOnly:       !sideEffect,
		SideEffect:     sideEffect,
		Destructive:    destructive,
		SupportsDryRun: true,
		RequiresYes:    destructive,
		Hint:           hint,
	}
}

func localCommandSafetyFor(key string, cmd *cobra.Command) *commandSafety {
	key = strings.TrimSpace(key)
	if spec, ok := localCommandSafetySpecs[key]; ok {
		return cloneCommandSafety(spec)
	}
	if key == "api" || strings.HasPrefix(key, "api ") {
		return &commandSafety{
			AuthRequired:   true,
			ReadOnly:       false,
			SideEffect:     true,
			Destructive:    false,
			SupportsDryRun: true,
			RequiresYes:    false,
			Hint:           "prefer typed commands; for raw write methods use --dry-run --request-out before running",
		}
	}
	if cmd != nil && cmd.Runnable() {
		return &commandSafety{
			AuthRequired:   false,
			ReadOnly:       true,
			SideEffect:     false,
			Destructive:    false,
			SupportsDryRun: false,
			RequiresYes:    false,
			Hint:           "local CLI command; inspect flags before running",
		}
	}
	return nil
}

func cloneCommandSafety(in commandSafety) *commandSafety {
	out := in
	return &out
}

func isWriteMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func isReadLikeCommand(key string, spec commandEndpointSpec) bool {
	method := strings.ToUpper(strings.TrimSpace(spec.Method))
	if method == "GET" {
		return true
	}
	if method != "POST" {
		return false
	}
	if key == "search image create" {
		return false
	}
	if strings.HasPrefix(key, "common-law ") || strings.HasPrefix(key, "domain ") {
		return true
	}
	if spec.Path == "/portfolio/trademarks/import/preview" {
		return true
	}
	key = strings.TrimSpace(key)
	if strings.HasPrefix(key, "search ") {
		return true
	}
	switch spec.Path {
	case "/trademark/detail",
		"/trademark/search",
		"/trademark/search/summary",
		"/trademark/office-action/search",
		"/trademark/ttab/search",
		"/trademark/wide-table/lawsuits",
		"/trademark/wide-table/brand-owners/{graphId}/lawsuits",
		"/trademark/wide-table/lawyers/{graphId}/lawsuits",
		"/trademark/wide-table/lawyers/{graphId}/law-firms",
		"/trademark/wide-table/lawyers/{graphId}/trademarks":
		return true
	default:
		return false
	}
}

func commandPaginationFor(cmd *cobra.Command) *commandPagination {
	if cmd == nil {
		return nil
	}
	flags := cmd.Flags()
	if flags == nil || flags.Lookup("page-all") == nil {
		return nil
	}
	out := &commandPagination{
		SupportsPageAll:   true,
		RecommendedFormat: "ndjson",
	}
	if flags.Lookup("fields") != nil {
		out.SupportsFields = true
		out.Flags = append(out.Flags, "--fields")
	}
	if flags.Lookup("manifest") != nil {
		out.SupportsManifest = true
		out.Flags = append(out.Flags, "--manifest")
	}
	for _, name := range []string{"--page", "--page-size", "--page-all", "--max-pages", "--max-rows", "--progress"} {
		if flags.Lookup(strings.TrimPrefix(name, "--")) != nil {
			out.Flags = append(out.Flags, name)
		}
	}
	sort.Strings(out.Flags)
	return out
}

func commandExamplesFor(cmd *cobra.Command, key string, safety *commandSafety, pagination *commandPagination) []string {
	examples := []string{}
	if strings.TrimSpace(key) != "" {
		examples = append(examples, "tmc "+key+" --help")
	}
	if safety != nil && safety.SupportsDryRun && safety.SideEffect {
		command := strings.TrimPrefix(commandUseLine(cmd, key), "tmc ")
		if _, ok := commandEndpointSpecs[key]; ok {
			examples = append(examples, "tmc --dry-run --request-out request.json "+command)
		} else {
			examples = append(examples, "tmc --dry-run "+command)
		}
	}
	if pagination != nil && pagination.SupportsPageAll {
		examples = append(examples, "tmc "+key+" --page-all --format ndjson --output export.ndjson --manifest export.manifest.json")
	}
	return examples
}

func commandUseLine(cmd *cobra.Command, key string) string {
	if cmd == nil {
		return "tmc " + key
	}
	line := strings.TrimSpace(cmd.UseLine())
	if line == "" {
		return "tmc " + key
	}
	return line
}

var localCommandSafetySpecs = map[string]commandSafety{
	"agent bootstrap": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "local discovery command; --check verifies credentials when configured",
	},
	"api catalog": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "local OpenAPI catalog inspection; prefer typed coverage for normal workflows",
	},
	"api endpoint": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "local OpenAPI endpoint metadata inspection",
	},
	"api schema": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "local OpenAPI schema inspection; use tmc schema <command...> first",
	},
	"auth import-key": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: true,
		Hint:           "stores a local API key; use --dry-run to preview without writing credentials",
	},
	"auth login": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: true,
		Hint:           "starts an auth flow and stores a local API key; use --dry-run to preview",
	},
	"auth logout": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		Destructive:    true,
		SupportsDryRun: true,
		Hint:           "removes locally stored credentials; use --dry-run to preview",
	},
	"auth status": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "reads local auth configuration; --check calls /auth/me when an API key is configured",
	},
	"config init": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: false,
		Hint:           "creates or rewrites the local CLI config file",
	},
	"config profile add": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: false,
		Hint:           "adds or replaces a local config profile",
	},
	"config profile list": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "reads local config profiles",
	},
	"config profile use": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: false,
		Hint:           "switches the active local config profile",
	},
	"config set": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: false,
		Hint:           "updates the active local config profile",
	},
	"config show": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "reads local CLI configuration",
	},
	"doctor": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "runs read-only network and auth checks",
	},
	"doctor auth": {
		AuthRequired:   true,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "calls /auth/me to verify credentials",
	},
	"doctor network": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "calls /version to verify endpoint reachability",
	},
	"gap wait": {
		AuthRequired:   true,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "polls a gap analysis with GET requests until a terminal status",
	},
	"profile add": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: false,
		Hint:           "adds or replaces a local config profile",
	},
	"portfolio monitored-summary": {
		AuthRequired:   true,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "calls the portfolio monitored summary endpoint; OpenAPI metadata is not available in this CLI build",
	},
	"profile list": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "reads local config profiles",
	},
	"profile use": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: false,
		Hint:           "switches the active local config profile",
	},
	"schema": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "local command schema inspection",
	},
	"setup": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: true,
		Hint:           "sets up local config and credentials; use --dry-run to preview",
	},
	"skills list": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "reads embedded skill inventory from this CLI build",
	},
	"skills read": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "reads embedded skill content from this CLI build",
	},
	"update": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: false,
		Hint:           "checks npm metadata and may install a newer CLI version",
	},
	"update check": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		SupportsDryRun: false,
		Hint:           "checks npm metadata and updates the local update-check cache",
	},
	"uninstall": {
		AuthRequired:   false,
		ReadOnly:       false,
		SideEffect:     true,
		Destructive:    true,
		SupportsDryRun: true,
		RequiresYes:    true,
		Hint:           "removes local tmc and tmcopilot binaries; keeps config unless --remove-config is passed",
	},
	"version": {
		AuthRequired:   false,
		ReadOnly:       true,
		SupportsDryRun: false,
		Hint:           "reads local CLI version metadata",
	},
}

var commandEndpointSpecs = map[string]commandEndpointSpec{
	"auth api-keys create":                      {Method: "POST", Path: "/auth/api-keys"},
	"auth api-keys list":                        {Method: "GET", Path: "/auth/api-keys"},
	"auth api-keys revoke":                      {Method: "DELETE", Path: "/auth/api-keys/{id}"},
	"auth collaborators accept":                 {Method: "POST", Path: "/auth/collaborators/invitations/{token}/accept"},
	"auth collaborators delete-invitation":      {Method: "DELETE", Path: "/auth/collaborators/invitations/{id}"},
	"auth collaborators invite":                 {Method: "POST", Path: "/auth/collaborators/invitations"},
	"auth collaborators list":                   {Method: "GET", Path: "/auth/collaborators"},
	"auth collaborators remove":                 {Method: "DELETE", Path: "/auth/collaborators/{id}"},
	"auth collaborators role":                   {Method: "PUT", Path: "/auth/collaborators/{id}/role"},
	"auth notification-preferences get":         {Method: "GET", Path: "/auth/notification-preferences"},
	"auth notification-preferences update":      {Method: "PUT", Path: "/auth/notification-preferences"},
	"auth ui-settings":                          {Method: "GET", Path: "/auth/ui-settings"},
	"auth whoami":                               {Method: "GET", Path: "/auth/me"},
	"auth workspaces":                           {Method: "GET", Path: "/auth/workspaces"},
	"competitors activities list":               {Method: "GET", Path: "/competitors/activities"},
	"competitors list":                          {Method: "GET", Path: "/competitors"},
	"competitors reports list":                  {Method: "GET", Path: "/competitors/reports"},
	"common-law max-similarity":                 {Method: "POST", Path: "/common-law/max-similarity"},
	"common-law search app-store":               {Method: "POST", Path: "/common-law/search/app-store"},
	"common-law search ecommerce-handle":        {Method: "POST", Path: "/common-law/search/ecommerce/handle"},
	"common-law search google-text":             {Method: "POST", Path: "/common-law/search/google/text"},
	"common-law search social-handle":           {Method: "POST", Path: "/common-law/search/social/handle"},
	"common-law search social-text":             {Method: "POST", Path: "/common-law/search/social/text"},
	"domain max-similarity":                     {Method: "POST", Path: "/domain/max-similarity"},
	"domain search":                             {Method: "POST", Path: "/domain/search"},
	"files list":                                {Method: "GET", Path: "/files"},
	"files presign":                             {Method: "POST", Path: "/files/presign"},
	"files upload-presign":                      {Method: "POST", Path: "/upload/presign"},
	"gap create":                                {Method: "POST", Path: "/gap-analyses"},
	"gap delete":                                {Method: "DELETE", Path: "/gap-analyses/{id}"},
	"gap generate-report":                       {Method: "POST", Path: "/gap-analyses/{id}/reports/generate"},
	"gap get":                                   {Method: "GET", Path: "/gap-analyses/{id}"},
	"gap list":                                  {Method: "GET", Path: "/gap-analyses"},
	"gap reports":                               {Method: "GET", Path: "/gap-analyses/{id}/reports"},
	"gap results":                               {Method: "GET", Path: "/gap-analyses/{id}/results"},
	"gap run":                                   {Method: "POST", Path: "/gap-analyses/{id}/run"},
	"gap shares create":                         {Method: "POST", Path: "/gap-analyses/{id}/share"},
	"gap shares get":                            {Method: "GET", Path: "/gap-analyses/shares/{token}"},
	"gap shares list":                           {Method: "GET", Path: "/gap-analyses/{id}/shares"},
	"gap shares revoke":                         {Method: "DELETE", Path: "/gap-analyses/{id}/shares/{token}"},
	"portfolio actions cbp":                     {Method: "GET", Path: "/portfolio/actions/cbp"},
	"portfolio actions cbp list":                {Method: "GET", Path: "/portfolio/actions/cbp"},
	"portfolio actions cbp service-requests":    {Method: "GET", Path: "/portfolio/actions/cbp/service-requests"},
	"portfolio actions cbp submit":              {Method: "POST", Path: "/portfolio/actions/cbp/service-requests"},
	"portfolio actions cbp-summary":             {Method: "GET", Path: "/portfolio/actions/cbp/summary"},
	"portfolio actions conflict":                {Method: "GET", Path: "/portfolio/actions/conflict"},
	"portfolio actions conflict for-trademark":  {Method: "GET", Path: "/portfolio/trademarks/{trademarkId}/conflict-actions"},
	"portfolio actions conflict get":            {Method: "GET", Path: "/portfolio/trademarks/{trademarkId}/conflict-actions/{id}"},
	"portfolio actions conflict groups":         {Method: "GET", Path: "/portfolio/actions/conflict/groups"},
	"portfolio actions conflict list":           {Method: "GET", Path: "/portfolio/actions/conflict"},
	"portfolio actions conflict status":         {Method: "PUT", Path: "/portfolio/trademarks/{trademarkId}/conflict-actions/{id}/status"},
	"portfolio actions conflict-summary":        {Method: "GET", Path: "/portfolio/actions/conflict/summary"},
	"portfolio actions office":                  {Method: "GET", Path: "/portfolio/actions/office"},
	"portfolio actions office deadlines":        {Method: "GET", Path: "/portfolio/actions/office/deadlines"},
	"portfolio actions office for-trademark":    {Method: "GET", Path: "/portfolio/trademarks/{trademarkId}/office-actions"},
	"portfolio actions office get":              {Method: "GET", Path: "/portfolio/trademarks/{trademarkId}/office-actions/{id}"},
	"portfolio actions office list":             {Method: "GET", Path: "/portfolio/actions/office"},
	"portfolio actions office status":           {Method: "PUT", Path: "/portfolio/trademarks/{trademarkId}/office-actions/{id}/status"},
	"portfolio actions office-summary":          {Method: "GET", Path: "/portfolio/actions/office/summary"},
	"portfolio activity list":                   {Method: "GET", Path: "/portfolio/activity"},
	"portfolio counts":                          {Method: "GET", Path: "/portfolio/trademarks/counts"},
	"portfolio groups list":                     {Method: "GET", Path: "/portfolio/trademark-groups"},
	"portfolio groups monitor-toggle":           {Method: "PUT", Path: "/portfolio/trademark-groups/{groupId}/monitor/toggle"},
	"portfolio trademarks get":                  {Method: "GET", Path: "/portfolio/trademarks/{trademarkId}"},
	"portfolio trademarks import":               {Method: "POST", Path: "/portfolio/trademarks/import"},
	"portfolio trademarks import-preview":       {Method: "POST", Path: "/portfolio/trademarks/import/preview"},
	"portfolio trademarks list":                 {Method: "GET", Path: "/portfolio/trademarks/search"},
	"portfolio trademarks metadata get":         {Method: "GET", Path: "/portfolio/trademarks/{trademarkId}/metadata"},
	"portfolio trademarks metadata update":      {Method: "PUT", Path: "/portfolio/trademarks/{trademarkId}/metadata"},
	"portfolio trademarks monitor batch-toggle": {Method: "PUT", Path: "/portfolio/trademark-monitor/toggle"},
	"portfolio trademarks monitor batch-update": {Method: "PUT", Path: "/portfolio/trademark-monitor"},
	"portfolio trademarks monitor update":       {Method: "PUT", Path: "/portfolio/trademarks/{trademarkId}/monitor"},
	"portfolio trademarks monitored":            {Method: "GET", Path: "/portfolio/trademarks/monitored"},
	"portfolio trademarks update":               {Method: "PUT", Path: "/portfolio/trademarks/{trademarkId}"},
	"search detail":                             {Method: "POST", Path: "/trademark/detail"},
	"search image create":                       {Method: "POST", Path: "/trademark/image/task"},
	"search image result":                       {Method: "GET", Path: "/trademark/image/task/{id}/result"},
	"search image result-post":                  {Method: "POST", Path: "/trademark/image/task/result"},
	"search lawyer-contact":                     {Method: "GET", Path: "/trademark/lawyer/contact"},
	"search lawyer-ranking":                     {Method: "GET", Path: "/trademark/lawyer/ranking"},
	"search lawyers":                            {Method: "GET", Path: "/trademark/lawyer/search"},
	"search office-actions":                     {Method: "POST", Path: "/trademark/office-action/search"},
	"search owner-ranking":                      {Method: "GET", Path: "/trademark/owner/ranking"},
	"search owners":                             {Method: "GET", Path: "/trademark/owner/search"},
	"search summary":                            {Method: "POST", Path: "/trademark/search/summary"},
	"search tips":                               {Method: "GET", Path: "/trademark/search/tips"},
	"search trademarks":                         {Method: "POST", Path: "/trademark/search"},
	"search ttab":                               {Method: "POST", Path: "/trademark/ttab/search"},
	"search ttab-case":                          {Method: "GET", Path: "/trademark/ttab/{case_number}"},
	"search uspto-document":                     {Method: "GET", Path: "/trademark/office-action/uspto/document"},
	"search lawsuit":                            {Method: "GET", Path: "/trademark/wide-table/lawsuits/{caseNumber}"},
	"search lawsuits":                           {Method: "POST", Path: "/trademark/wide-table/lawsuits"},
	"ttab case":                                 {Method: "GET", Path: "/trademark/ttab/{case_number}"},
	"ttab search":                               {Method: "POST", Path: "/trademark/ttab/search"},
	"lawsuits brand-owner":                      {Method: "POST", Path: "/trademark/wide-table/brand-owners/{graphId}/lawsuits"},
	"lawsuits get":                              {Method: "GET", Path: "/trademark/wide-table/lawsuits/{caseNumber}"},
	"lawsuits lawyer":                           {Method: "POST", Path: "/trademark/wide-table/lawyers/{graphId}/lawsuits"},
	"lawsuits search":                           {Method: "POST", Path: "/trademark/wide-table/lawsuits"},
	"lawyers contact":                           {Method: "GET", Path: "/trademark/lawyer/contact"},
	"lawyers get":                               {Method: "GET", Path: "/trademark/wide-table/lawyers/{graphId}"},
	"lawyers law-firms":                         {Method: "POST", Path: "/trademark/wide-table/lawyers/{graphId}/law-firms"},
	"lawyers lawsuits":                          {Method: "POST", Path: "/trademark/wide-table/lawyers/{graphId}/lawsuits"},
	"lawyers ranking":                           {Method: "GET", Path: "/trademark/lawyer/ranking"},
	"lawyers search":                            {Method: "GET", Path: "/trademark/lawyer/search"},
	"lawyers trademarks":                        {Method: "POST", Path: "/trademark/wide-table/lawyers/{graphId}/trademarks"},
}
