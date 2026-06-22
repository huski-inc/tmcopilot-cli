package tmc

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/config"
	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
	"github.com/huski-inc/tmcopilot-cli/internal/skills"
	"github.com/huski-inc/tmcopilot-cli/internal/version"
)

type agentBootstrapResult struct {
	CLI                  agentCLIInfo        `json:"cli"`
	Auth                 agentAuthInfo       `json:"auth"`
	Config               agentConfigInfo     `json:"config"`
	Discovery            agentDiscoveryInfo  `json:"discovery"`
	Skills               []skills.Info       `json:"skills"`
	RecommendedNextSteps []string            `json:"recommended_next_steps"`
	Safety               agentSafetyGuidance `json:"safety"`
}

type agentCLIInfo struct {
	Name     string   `json:"name"`
	Commands []string `json:"commands"`
	Version  string   `json:"version"`
	Commit   string   `json:"commit"`
	Date     string   `json:"date"`
}

type agentAuthInfo struct {
	Configured    bool   `json:"configured"`
	Source        string `json:"source,omitempty"`
	Verified      bool   `json:"verified"`
	CheckSkipped  bool   `json:"check_skipped,omitempty"`
	CheckRequired bool   `json:"check_required,omitempty"`
	Next          string `json:"next,omitempty"`
}

type agentConfigInfo struct {
	Profile     string `json:"profile"`
	Endpoint    string `json:"endpoint"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Format      string `json:"format"`
	ConfigPath  string `json:"config_path,omitempty"`
}

type agentDiscoveryInfo struct {
	ReadFirst       []string `json:"read_first"`
	Schema          string   `json:"schema"`
	Catalog         string   `json:"catalog"`
	RawAPI          string   `json:"raw_api"`
	DomainSkillHint string   `json:"domain_skill_hint"`
}

type agentSafetyGuidance struct {
	PreferDryRunForWrites  bool   `json:"prefer_dry_run_for_writes"`
	DestructiveRequiresYes bool   `json:"destructive_requires_yes"`
	LargeOutputHint        string `json:"large_output_hint"`
	SecretHandling         string `json:"secret_handling"`
}

func newAgentCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent-oriented discovery and bootstrap helpers",
		Long:  "Agent-oriented helpers that return machine-readable CLI discovery, auth, safety, and next-step guidance.",
	}
	cmd.AddCommand(newAgentBootstrapCommand(opts))
	return cmd
}

func newAgentBootstrapCommand(opts *globalOptions) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Print machine-readable guidance for AI agents",
		Long: `Print the minimum context an AI agent should inspect before using
TMCopilot CLI: command aliases, auth status, configured endpoint, embedded
skills, discovery commands, output rules, and recommended next steps.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				items, err := skills.List()
				if err != nil {
					return err
				}
				result := buildAgentBootstrapResult(rt, items, check)
				if check {
					if rt.APIKey == "" {
						result.Auth.CheckRequired = true
						result.Auth.Next = "run `tmc setup --no-wait`, ask the user to approve the URL, resume with `tmc setup --request-id <request_id>`, then rerun `tmc agent bootstrap --check`"
					} else {
						resp, err := rt.Client.Do(cmd.Context(), "GET", "/auth/me", nil, nil)
						if err != nil {
							return err
						}
						result.Auth.Verified = resp.StatusCode >= 200 && resp.StatusCode < 300
					}
				}
				return writeResult(rt, result, map[string]any{
					"openapi_source_hash": openapi.SourceHash,
					"openapi_source_path": openapi.SourcePath,
				})
			})
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "verify stored credentials with /auth/me when an API key is configured")
	return cmd
}

func buildAgentBootstrapResult(rt *runtimeContext, skillItems []skills.Info, check bool) agentBootstrapResult {
	info := version.Current()
	result := agentBootstrapResult{
		CLI: agentCLIInfo{
			Name:     info.Name,
			Commands: []string{"tmc", "tmcopilot"},
			Version:  info.Version,
			Commit:   info.Commit,
			Date:     info.Date,
		},
		Auth: agentAuthInfo{
			Configured: rt.APIKey != "",
			Source:     rt.APIKeySrc,
		},
		Config: agentConfigInfo{
			Profile:     rt.ProfileName,
			Endpoint:    rt.Profile.Endpoint,
			WorkspaceID: rt.Profile.WorkspaceID,
			Format:      rt.Format,
			ConfigPath:  configPathOrEmpty(),
		},
		Discovery: agentDiscoveryInfo{
			ReadFirst: []string{
				"tmc skills read tmc-shared",
				"tmc skills read <domain-skill>",
			},
			Schema:          "tmc schema <command...>",
			Catalog:         "tmc api catalog --coverage typed",
			RawAPI:          "tmc api METHOD /path",
			DomainSkillHint: "choose tmc-trademark-search, tmc-portfolio-export, tmc-gap-analysis, or tmc-openapi based on the user's task",
		},
		Skills: skillItems,
		Safety: agentSafetyGuidance{
			PreferDryRunForWrites:  true,
			DestructiveRequiresYes: true,
			LargeOutputHint:        "use --output for large JSON and --page-all --format ndjson --manifest for large paginated exports",
			SecretHandling:         "do not print API keys or authorization tokens; use tmc setup, tmc auth login, or --api-key-stdin",
		},
	}
	if !check {
		result.Auth.CheckSkipped = true
	}
	result.RecommendedNextSteps = agentNextSteps(result)
	return result
}

func agentNextSteps(result agentBootstrapResult) []string {
	steps := []string{"tmc skills read tmc-shared"}
	if !result.Auth.Configured {
		steps = append(steps, "tmc setup --no-wait")
	} else if !result.Auth.Verified && !result.Auth.CheckSkipped {
		steps = append(steps, "tmc auth status --check")
	} else if result.Auth.CheckSkipped {
		steps = append(steps, "tmc agent bootstrap --check")
	}
	if strings.TrimSpace(result.Config.WorkspaceID) == "" {
		steps = append(steps, "tmc auth workspaces")
	}
	steps = append(steps, "tmc schema <command...>")
	return steps
}

func configPathOrEmpty() string {
	path, err := config.ConfigPath()
	if err != nil {
		return ""
	}
	return path
}
