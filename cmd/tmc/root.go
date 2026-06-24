package tmc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/huski-inc/tmcopilot-cli/internal/client"
	"github.com/huski-inc/tmcopilot-cli/internal/config"
	"github.com/huski-inc/tmcopilot-cli/internal/output"
	"github.com/huski-inc/tmcopilot-cli/internal/version"
)

type globalOptions struct {
	profile        string
	endpoint       string
	workspaceID    string
	format         string
	output         string
	rawEnvelope    bool
	timeout        time.Duration
	dryRun         bool
	yes            bool
	requestOut     string
	idempotencyKey string
}

type runtimeContext struct {
	Config      *config.Config
	ProfileName string
	Profile     config.Profile
	APIKey      string
	APIKeySrc   string
	Client      *client.Client
	Format      string
	OutputPath  string
	RawEnvelope bool
	Out         io.Writer
}

type reportedError struct {
	err error
}

func (e reportedError) Error() string {
	return e.err.Error()
}

func (e reportedError) Unwrap() error {
	return e.err
}

func Execute(args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	cmd := NewRootCommand()
	cmd.SetArgs(args)
	if in != nil {
		cmd.SetIn(in)
	}
	if out != nil {
		cmd.SetOut(out)
	}
	if errOut != nil {
		cmd.SetErr(errOut)
	}
	lightweightUpdate := maybeRunLightweightAutomaticUpdateCheck(cmd, args)
	executedCmd, err := cmd.ExecuteC()
	if err != nil {
		var reported reportedError
		if !errors.As(err, &reported) {
			output.WriteFailure(cmd.ErrOrStderr(), classifyCommandFailure(cmd, args, err))
		}
		if lightweightUpdate != nil {
			writeUpdateAvailableNotice(cmd.ErrOrStderr(), *lightweightUpdate)
		}
		return exitCodeFor(err)
	}
	if lightweightUpdate != nil {
		writeUpdateAvailableNotice(executedCmd.ErrOrStderr(), *lightweightUpdate)
	} else {
		maybeRunAutomaticUpdateCheck(executedCmd)
	}
	return 0
}

func NewRootCommand() *cobra.Command {
	opts := &globalOptions{}
	root := &cobra.Command{
		Use:   "tmc",
		Short: "TMCopilot command line client",
		Long: `TMCopilot command line client.

COMMAND ALIAS:
  tmcopilot is equivalent to tmc.

USAGE:
  tmc <command> [subcommand] [options]
  tmc setup
  tmc search trademarks --name Nike --class 25,35
  tmc schema search trademarks
  tmc skills read tmc-trademark-search

DISCOVERY:
  Start with embedded agent guidance:
    tmc agent bootstrap
    tmc skills list
    tmc skills read tmc-shared
    tmc skills read tmc-trademark-search

  Prefer typed commands for common workflows, use schema/catalog to inspect
  Swagger metadata, then fall back to raw API calls for endpoints without a
  first-class CLI command.`,
		Example: `  tmc setup
  tmc setup --no-wait
  tmc auth login
  tmc agent bootstrap
  tmc search trademarks --name Nike --class 25,35 --limit 20
  tmc search owners --name "Nike"
  tmc ttab search --plaintiff Nike --issue opposition
  tmc lawsuits search --party Nike --trademark AIR
  tmc lawyers search --name Smith --state CA --limit 20
  tmc portfolio trademarks list --page-all --format ndjson --output trademarks.ndjson
  tmc schema search trademarks
  tmc skills read tmc-trademark-search`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&opts.profile, "profile", "", "profile name")
	root.PersistentFlags().StringVar(&opts.endpoint, "endpoint", "", "TMCopilot API endpoint")
	root.PersistentFlags().StringVar(&opts.workspaceID, "workspace", "", "workspace ID")
	root.PersistentFlags().StringVar(&opts.format, "format", "", "output format: json, pretty, raw; page-all also supports ndjson and csv")
	root.PersistentFlags().StringVar(&opts.output, "output", "", "write output to file")
	root.PersistentFlags().BoolVar(&opts.rawEnvelope, "raw-envelope", false, "return the backend response envelope")
	root.PersistentFlags().DurationVar(&opts.timeout, "timeout", 30*time.Second, "request timeout")
	root.PersistentFlags().BoolVar(&opts.dryRun, "dry-run", false, "print the request that would be sent without calling the API")
	root.PersistentFlags().BoolVar(&opts.yes, "yes", false, "confirm destructive write operations")
	root.PersistentFlags().StringVar(&opts.requestOut, "request-out", "", "write the resolved API request JSON to a file")
	root.PersistentFlags().StringVar(&opts.idempotencyKey, "idempotency-key", "", "send Idempotency-Key for supported write APIs")

	root.AddCommand(newVersionCommand(opts))
	root.AddCommand(newUpdateCommand(opts))
	root.AddCommand(newSetupCommand(opts))
	root.AddCommand(newConfigCommand(opts))
	root.AddCommand(newConfigProfileCommand(opts))
	root.AddCommand(newAuthCommand(opts))
	root.AddCommand(newDoctorCommand(opts))
	root.AddCommand(newAgentCommand(opts))
	root.AddCommand(newAPICommand(opts))
	root.AddCommand(newSchemaCommand(opts))
	root.AddCommand(newPortfolioCommand(opts))
	root.AddCommand(newCompetitorsCommand(opts))
	root.AddCommand(newCommonLawCommand(opts))
	root.AddCommand(newDomainCommand(opts))
	root.AddCommand(newTTABCommand(opts))
	root.AddCommand(newLawsuitCommand(opts))
	root.AddCommand(newLawyersCommand(opts))
	root.AddCommand(newSearchCommand(opts))
	root.AddCommand(newGapCommand(opts))
	root.AddCommand(newFilesCommand(opts))
	root.AddCommand(newSkillsCommand(opts))

	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return fmt.Errorf("%s: %w", cmd.CommandPath(), err)
	})
	return root
}

func commandRuntime(cmd *cobra.Command, opts *globalOptions, needAuth bool) (*runtimeContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	profileName, profile := cfg.ActiveProfile(opts.profile)
	if envEndpoint := config.EnvEndpoint(); envEndpoint != "" {
		profile.Endpoint = envEndpoint
	}
	if opts.endpoint != "" {
		profile.Endpoint = config.NormalizeEndpoint(opts.endpoint)
	}
	if opts.workspaceID != "" {
		profile.WorkspaceID = opts.workspaceID
	}
	format := opts.format
	if format == "" {
		format = profile.Format
	}
	if format == "" {
		format = cfg.DefaultFormat
	}
	if format == "" {
		format = "json"
	}

	apiKey, source := resolveAPIKey(profileName)
	if needAuth && apiKey == "" {
		return nil, errors.New("missing API key; run tmc setup in a browser terminal or tmc setup --no-wait in an agent environment")
	}

	apiClient := client.New(
		profile.Endpoint,
		apiKey,
		profile.WorkspaceID,
		"tmcopilot-cli/"+version.Version,
		opts.timeout,
	)
	apiClient.ExtraHeaders = map[string]string{
		"X-TMCopilot-CLI-Command": cmd.CommandPath(),
	}
	if strings.TrimSpace(opts.idempotencyKey) != "" {
		apiClient.ExtraHeaders["Idempotency-Key"] = strings.TrimSpace(opts.idempotencyKey)
	}
	return &runtimeContext{
		Config:      cfg,
		ProfileName: profileName,
		Profile:     profile,
		APIKey:      apiKey,
		APIKeySrc:   source,
		Client:      apiClient,
		Format:      format,
		OutputPath:  opts.output,
		RawEnvelope: opts.rawEnvelope,
		Out:         cmd.OutOrStdout(),
	}, nil
}

func resolveAPIKey(profileName string) (string, string) {
	if value, ok := config.EnvAPIKey(); ok {
		return value, "env"
	}
	creds, err := config.LoadCredentials()
	if err != nil || creds == nil {
		return "", ""
	}
	if cred, ok := creds.Profiles[profileName]; ok && strings.TrimSpace(cred.APIKey) != "" {
		return strings.TrimSpace(cred.APIKey), "credentials"
	}
	return "", ""
}

func writeResult(rt *runtimeContext, data any, meta map[string]any) error {
	data = normalizeCLIResponseData(data)
	format := "json"
	outputPath := ""
	if rt != nil {
		format = rt.Format
		outputPath = rt.OutputPath
		return output.WriteTo(rt.Out, format, outputPath, data, meta)
	}
	return output.Write(format, outputPath, data, meta)
}

func handleCommand(cmd *cobra.Command, fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}
	output.WriteFailure(cmd.ErrOrStderr(), classifyFailure(err))
	return reportedError{err: err}
}

func classifyCommandFailure(root *cobra.Command, args []string, err error) output.Failure {
	message := err.Error()
	failure := output.Failure{
		Type:    "validation_error",
		Message: message,
	}
	if parent, unknown := unknownCommandInArgs(root, args); parent != nil && unknown != "" {
		suggestions := suggestCommands(parent, unknown)
		failure.Message = fmt.Sprintf("unknown subcommand %q for %q", unknown, parent.CommandPath())
		failure.Hint = fmt.Sprintf("run `%s --help` to see available subcommands", parent.CommandPath())
		if len(suggestions) > 0 {
			failure.Suggestions = suggestions
			failure.Hint = fmt.Sprintf("did you mean %s? run `%s --help` for the full list", strings.Join(formatCodeList(suggestions), ", "), parent.CommandPath())
		}
		return failure
	}
	context := findCommandContext(root, args)
	if context == nil {
		context = root
	}
	switch {
	case strings.Contains(message, "unknown command"):
		unknown := extractQuoted(message)
		suggestions := suggestCommands(context, unknown)
		failure.Hint = fmt.Sprintf("run `%s --help` to see available subcommands", context.CommandPath())
		if len(suggestions) > 0 {
			failure.Suggestions = suggestions
			failure.Hint = fmt.Sprintf("did you mean %s? run `%s --help` for the full list", strings.Join(formatCodeList(suggestions), ", "), context.CommandPath())
		}
	case strings.Contains(message, "unknown flag"):
		flagName := extractUnknownFlag(message)
		suggestions := suggestFlags(context, flagName)
		failure.Hint = fmt.Sprintf("run `%s --help` to see valid flags", context.CommandPath())
		if len(suggestions) > 0 {
			failure.Suggestions = suggestions
			failure.Hint = fmt.Sprintf("did you mean %s? run `%s --help` for all flags", strings.Join(formatCodeList(suggestions), ", "), context.CommandPath())
		}
	case strings.Contains(message, "arg(s)") || strings.Contains(message, "requires"):
		failure.Hint = fmt.Sprintf("run `%s --help` for usage", context.CommandPath())
	default:
		failure.Type = "cli_error"
	}
	return failure
}

func unknownCommandInArgs(root *cobra.Command, args []string) (*cobra.Command, string) {
	current := root
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		child := findChildCommand(current, arg)
		if child != nil {
			current = child
			continue
		}
		if !current.Runnable() && hasVisibleSubcommands(current) {
			return current, arg
		}
		break
	}
	return nil, ""
}

func hasVisibleSubcommands(cmd *cobra.Command) bool {
	for _, child := range cmd.Commands() {
		if !child.Hidden && child.Name() != "help" && child.Name() != "completion" {
			return true
		}
	}
	return false
}

func exitCodeFor(err error) int {
	var reported reportedError
	if errors.As(err, &reported) {
		return 1
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unknown command") ||
		strings.Contains(message, "unknown flag") ||
		strings.Contains(message, "arg(s)") ||
		strings.Contains(message, "requires") {
		return 2
	}
	return 1
}

func findCommandContext(root *cobra.Command, args []string) *cobra.Command {
	current := root
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		child := findChildCommand(current, arg)
		if child == nil {
			break
		}
		current = child
	}
	return current
}

func findChildCommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, child := range cmd.Commands() {
		if child.Hidden {
			continue
		}
		if child.Name() == name {
			return child
		}
		for _, alias := range child.Aliases {
			if alias == name {
				return child
			}
		}
	}
	return nil
}

func suggestCommands(cmd *cobra.Command, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || cmd == nil {
		return nil
	}
	candidates := make([]string, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		if child.Hidden || child.Name() == "help" || child.Name() == "completion" {
			continue
		}
		candidates = append(candidates, child.Name())
		candidates = append(candidates, child.Aliases...)
	}
	return closest(value, candidates, 3)
}

func suggestFlags(cmd *cobra.Command, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || cmd == nil {
		return nil
	}
	candidates := map[string]bool{}
	visit := func(flag *pflag.Flag) {
		if flag == nil || flag.Hidden {
			return
		}
		candidates["--"+flag.Name] = true
		if flag.Shorthand != "" {
			candidates["-"+flag.Shorthand] = true
		}
	}
	cmd.Flags().VisitAll(visit)
	cmd.InheritedFlags().VisitAll(visit)
	if root := cmd.Root(); root != nil {
		root.PersistentFlags().VisitAll(visit)
	}
	values := make([]string, 0, len(candidates))
	for candidate := range candidates {
		values = append(values, candidate)
	}
	return closest(value, values, 3)
}

func closest(value string, candidates []string, limit int) []string {
	type scored struct {
		value string
		score int
	}
	seen := map[string]bool{}
	scores := make([]scored, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		score := levenshtein(strings.ToLower(value), strings.ToLower(candidate))
		if strings.Contains(strings.ToLower(candidate), strings.ToLower(value)) {
			score = 0
		}
		maxDistance := 2
		if len(value) > 8 {
			maxDistance = 3
		}
		if score <= maxDistance {
			scores = append(scores, scored{value: candidate, score: score})
		}
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			return scores[i].value < scores[j].value
		}
		return scores[i].score < scores[j].score
	})
	if len(scores) > limit {
		scores = scores[:limit]
	}
	out := make([]string, 0, len(scores))
	for _, item := range scores {
		out = append(out, item.value)
	}
	return out
}

func levenshtein(a string, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = minInt(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func minInt(values ...int) int {
	min := values[0]
	for _, value := range values[1:] {
		if value < min {
			min = value
		}
	}
	return min
}

func extractQuoted(message string) string {
	start := strings.Index(message, "\"")
	if start < 0 {
		return ""
	}
	rest := message[start+1:]
	end := strings.Index(rest, "\"")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func extractUnknownFlag(message string) string {
	const prefix = "unknown flag:"
	idx := strings.Index(strings.ToLower(message), prefix)
	if idx < 0 {
		return ""
	}
	value := strings.TrimSpace(message[idx+len(prefix):])
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return value
	}
	return fields[0]
}

func formatCodeList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, "`"+value+"`")
	}
	return out
}

func classifyFailure(err error) output.Failure {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		errType, retryable := classifyAPIError(apiErr.StatusCode)
		return output.Failure{
			Type:       errType,
			Message:    apiErr.Error(),
			StatusCode: apiErr.StatusCode,
			Code:       apiErr.Code,
			TraceID:    apiErr.TraceID,
			Retryable:  retryable,
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return output.Failure{Type: "timeout_error", Message: err.Error(), Retryable: true}
	}
	message := err.Error()
	if isNetworkErrorMessage(message) {
		return output.Failure{Type: "network_error", Message: message, Retryable: true}
	}
	if strings.Contains(message, "missing API key") {
		return output.Failure{
			Type:    "auth_error",
			Message: message,
			Hint:    "run `tmc setup` in a browser terminal, `tmc setup --no-wait` in an agent environment, or import an existing key with `tmc setup --api-key-stdin`",
		}
	}
	return output.Failure{Type: "cli_error", Message: message}
}

func classifyAPIError(statusCode int) (string, bool) {
	switch {
	case statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity:
		return "validation_error", false
	case statusCode == http.StatusUnauthorized:
		return "auth_error", false
	case statusCode == http.StatusForbidden:
		return "permission_denied", false
	case statusCode == http.StatusNotFound:
		return "not_found", false
	case statusCode == http.StatusConflict:
		return "conflict_error", false
	case statusCode == http.StatusTooManyRequests:
		return "rate_limited", true
	case statusCode >= 500:
		return "server_error", true
	default:
		return "api_error", false
	}
}

func isNetworkErrorMessage(message string) bool {
	message = strings.ToLower(message)
	for _, part := range []string{"connection refused", "no such host", "network is unreachable", "connection reset", "i/o timeout"} {
		if strings.Contains(message, part) {
			return true
		}
	}
	return false
}

func parseParams(values []string) (url.Values, error) {
	query := url.Values{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key, val, ok := strings.Cut(value, "=")
		if !ok {
			return nil, fmt.Errorf("invalid param %q, expected key=value", value)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid param %q, key is empty", value)
		}
		query.Add(key, val)
	}
	return query, nil
}

func readDataArg(value string) (any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if strings.HasPrefix(value, "@") {
		raw, err := os.ReadFile(strings.TrimPrefix(value, "@"))
		if err != nil {
			return nil, err
		}
		return decodeJSON(raw)
	}
	return decodeJSON([]byte(value))
}

func decodeJSON(raw []byte) (any, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}
