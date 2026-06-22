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
	Aliases       []string                `json:"aliases,omitempty"`
	Short         string                  `json:"short,omitempty"`
	Use           string                  `json:"use,omitempty"`
	Flags         []flagSchema            `json:"flags,omitempty"`
	GlobalFlags   []flagSchema            `json:"global_flags,omitempty"`
	Endpoint      *openapi.Endpoint       `json:"endpoint,omitempty"`
	OpenAPI       *openapi.EndpointSchema `json:"openapi,omitempty"`
	Children      []commandListItem       `json:"children,omitempty"`
	SchemaCommand string                  `json:"schema_command,omitempty"`
}

type commandListItem struct {
	Command       string            `json:"command"`
	Aliases       []string          `json:"aliases,omitempty"`
	Short         string            `json:"short,omitempty"`
	Endpoint      *openapi.Endpoint `json:"endpoint,omitempty"`
	SchemaCommand string            `json:"schema_command,omitempty"`
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
					items := listMappedCommandSchemas(cmd.Root(), "")
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
		Aliases:       commandAliases(cmd),
		Short:         cmd.Short,
		Use:           cmd.UseLine(),
		Flags:         collectFlags(cmd.Flags()),
		Children:      listChildCommandSchemas(cmd),
		SchemaCommand: "tmc schema " + key,
	}
	if spec, ok := commandEndpointSpecs[key]; ok {
		if schema, err := endpointSchemaFor(spec.Method, spec.Path); err == nil {
			endpoint := schema.Endpoint
			result.Endpoint = &endpoint
			if includeOpenAPI {
				result.OpenAPI = &schema
			}
		}
	}
	return result
}

func listMappedCommandSchemas(root *cobra.Command, prefix string) []commandListItem {
	items := make([]commandListItem, 0, len(commandEndpointSpecs))
	for key, spec := range commandEndpointSpecs {
		if prefix != "" && !strings.HasPrefix(key, prefix+" ") {
			continue
		}
		cmd, err := resolveSchemaCommandTarget(root, strings.Fields(key))
		if err != nil {
			continue
		}
		item := commandListItem{
			Command:       "tmc " + key,
			Aliases:       commandAliases(cmd),
			Short:         cmd.Short,
			SchemaCommand: "tmc schema " + key,
		}
		if endpoint, ok := openapi.FindEndpoint(spec.Method, spec.Path); ok {
			item.Endpoint = &endpoint
		}
		items = append(items, item)
	}
	sortCommandList(items)
	return items
}

func listChildCommandSchemas(cmd *cobra.Command) []commandListItem {
	prefix := commandKey(cmd)
	if prefix == "" {
		return nil
	}
	items := listMappedCommandSchemas(cmd.Root(), prefix)
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
			Aliases:       commandAliases(child),
			Short:         child.Short,
			SchemaCommand: "tmc schema " + key,
		})
	}
	sortCommandList(items)
	return items
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

var commandEndpointSpecs = map[string]commandEndpointSpec{
	"auth api-keys create":                 {Method: "POST", Path: "/auth/api-keys"},
	"auth api-keys list":                   {Method: "GET", Path: "/auth/api-keys"},
	"auth api-keys revoke":                 {Method: "DELETE", Path: "/auth/api-keys/{id}"},
	"auth collaborators accept":            {Method: "POST", Path: "/auth/collaborators/invitations/{token}/accept"},
	"auth collaborators delete-invitation": {Method: "DELETE", Path: "/auth/collaborators/invitations/{id}"},
	"auth collaborators invite":            {Method: "POST", Path: "/auth/collaborators/invitations"},
	"auth collaborators list":              {Method: "GET", Path: "/auth/collaborators"},
	"auth collaborators remove":            {Method: "DELETE", Path: "/auth/collaborators/{id}"},
	"auth collaborators role":              {Method: "PUT", Path: "/auth/collaborators/{id}/role"},
	"auth notification-preferences get":    {Method: "GET", Path: "/auth/notification-preferences"},
	"auth notification-preferences update": {Method: "PUT", Path: "/auth/notification-preferences"},
	"auth ui-settings":                     {Method: "GET", Path: "/auth/ui-settings"},
	"auth whoami":                          {Method: "GET", Path: "/auth/me"},
	"auth workspaces":                      {Method: "GET", Path: "/auth/workspaces"},
	"competitors activities list":          {Method: "GET", Path: "/competitors/activities"},
	"competitors list":                     {Method: "GET", Path: "/competitors"},
	"competitors reports list":             {Method: "GET", Path: "/competitors/reports"},
	"files list":                           {Method: "GET", Path: "/files"},
	"files presign":                        {Method: "POST", Path: "/files/presign"},
	"files upload-presign":                 {Method: "POST", Path: "/upload/presign"},
	"gap create":                           {Method: "POST", Path: "/gap-analyses"},
	"gap delete":                           {Method: "DELETE", Path: "/gap-analyses/{id}"},
	"gap generate-report":                  {Method: "POST", Path: "/gap-analyses/{id}/reports/generate"},
	"gap get":                              {Method: "GET", Path: "/gap-analyses/{id}"},
	"gap list":                             {Method: "GET", Path: "/gap-analyses"},
	"gap reports":                          {Method: "GET", Path: "/gap-analyses/{id}/reports"},
	"gap results":                          {Method: "GET", Path: "/gap-analyses/{id}/results"},
	"gap run":                              {Method: "POST", Path: "/gap-analyses/{id}/run"},
	"gap shares create":                    {Method: "POST", Path: "/gap-analyses/{id}/share"},
	"gap shares get":                       {Method: "GET", Path: "/gap-analyses/shares/{token}"},
	"gap shares list":                      {Method: "GET", Path: "/gap-analyses/{id}/shares"},
	"gap shares revoke":                    {Method: "DELETE", Path: "/gap-analyses/{id}/shares/{token}"},
	"portfolio actions cbp":                {Method: "GET", Path: "/portfolio/actions/cbp"},
	"portfolio actions cbp-summary":        {Method: "GET", Path: "/portfolio/actions/cbp/summary"},
	"portfolio actions conflict":           {Method: "GET", Path: "/portfolio/actions/conflict"},
	"portfolio actions conflict-summary":   {Method: "GET", Path: "/portfolio/actions/conflict/summary"},
	"portfolio actions office":             {Method: "GET", Path: "/portfolio/actions/office"},
	"portfolio actions office-summary":     {Method: "GET", Path: "/portfolio/actions/office/summary"},
	"portfolio activity list":              {Method: "GET", Path: "/portfolio/activity"},
	"portfolio counts":                     {Method: "GET", Path: "/portfolio/trademarks/counts"},
	"portfolio tasks get":                  {Method: "GET", Path: "/portfolio/tasks/{taskId}"},
	"portfolio tasks latest-sync":          {Method: "GET", Path: "/portfolio/tasks/latest-sync"},
	"portfolio tasks list":                 {Method: "GET", Path: "/portfolio/tasks"},
	"portfolio tasks stats":                {Method: "GET", Path: "/portfolio/tasks/stats"},
	"portfolio trademarks get":             {Method: "GET", Path: "/portfolio/trademarks/{trademarkId}"},
	"portfolio trademarks list":            {Method: "GET", Path: "/portfolio/trademarks/search"},
	"portfolio trademarks monitored":       {Method: "GET", Path: "/portfolio/trademarks/monitored"},
	"search detail":                        {Method: "POST", Path: "/trademark/detail"},
	"search lawyer-contact":                {Method: "GET", Path: "/trademark/lawyer/contact"},
	"search lawyer-ranking":                {Method: "GET", Path: "/trademark/lawyer/ranking"},
	"search lawyers":                       {Method: "GET", Path: "/trademark/lawyer/search"},
	"search office-actions":                {Method: "POST", Path: "/trademark/office-action/search"},
	"search owner-ranking":                 {Method: "GET", Path: "/trademark/owner/ranking"},
	"search owners":                        {Method: "GET", Path: "/trademark/owner/search"},
	"search summary":                       {Method: "POST", Path: "/trademark/search/summary"},
	"search tips":                          {Method: "GET", Path: "/trademark/search/tips"},
	"search trademarks":                    {Method: "POST", Path: "/trademark/search"},
	"search ttab":                          {Method: "POST", Path: "/trademark/ttab/search"},
	"search ttab-case":                     {Method: "GET", Path: "/trademark/ttab/{case_number}"},
}
