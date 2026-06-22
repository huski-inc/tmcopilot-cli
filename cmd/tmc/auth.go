package tmc

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/config"
	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
)

func newAuthCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}
	cmd.AddCommand(newAuthLoginCommand(opts))
	cmd.AddCommand(newAuthImportKeyCommand(opts))
	cmd.AddCommand(newAuthStatusCommand(opts))
	cmd.AddCommand(newAuthWhoamiCommand(opts))
	cmd.AddCommand(newAuthWorkspacesCommand(opts))
	cmd.AddCommand(newAuthLogoutCommand(opts))
	cmd.AddCommand(newAuthUISettingsCommand(opts))
	cmd.AddCommand(newAuthNotificationPreferencesCommand(opts))
	cmd.AddCommand(newAuthCollaboratorsCommand(opts))
	cmd.AddCommand(newAuthAPIKeysCommand(opts))
	return cmd
}

func newAuthLoginCommand(opts *globalOptions) *cobra.Command {
	loginOpts := loginCredentialOptions{Check: true}
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authorize this CLI and store an API key",
		Long: `Authorize this CLI in the browser, receive a one-time API key from
TMCopilot, and store it for the active profile.

The raw API key is stored locally and is never printed. For scripts and CI,
use tmc auth import-key or tmc setup --api-key-stdin with an existing API key.`,
		Example: `  tmc auth login
  tmc auth login --no-wait
  tmc auth login --request-id <request_id>
  tmc auth login --no-browser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				if opts.dryRun {
					return writeLoginSetupDryRun(cmd, opts, loginOpts)
				}
				return loginAndStoreAPIKey(cmd, opts, loginOpts)
			})
		},
	}
	cmd.Flags().BoolVar(&loginOpts.NoBrowser, "no-browser", false, "print the authorization URL instead of opening a browser")
	cmd.Flags().BoolVar(&loginOpts.NoWait, "no-wait", false, "create an authorization request, save it locally, print the URL, and exit without polling")
	cmd.Flags().StringVar(&loginOpts.RequestID, "request-id", "", "resume polling for a pending authorization request created by --no-wait")
	cmd.Flags().StringVar(&loginOpts.DeviceName, "device-name", "", "device name shown on the authorization page")
	cmd.Flags().StringVar(&loginOpts.KeyName, "key-name", "", "alias for --device-name; legacy API key name with --email")
	cmd.Flags().StringVar(&loginOpts.Email, "email", "", "legacy password login account email")
	cmd.Flags().BoolVar(&loginOpts.PasswordStdin, "password-stdin", false, "legacy password login: read password from stdin")
	cmd.Flags().StringVar(&loginOpts.TurnstileResponse, "turnstile-response", "", "legacy password login Turnstile verification token")
	cmd.Flags().Int64Var(&loginOpts.ExpiresIn, "expires-in", 0, "legacy password login generated API key expiry in seconds")
	cmd.Flags().BoolVar(&loginOpts.Check, "check", true, "verify the stored API key with /auth/me")
	return cmd
}

func newAuthImportKeyCommand(opts *globalOptions) *cobra.Command {
	var apiKey string
	var apiKeyStdin bool
	var name string
	var check bool
	cmd := &cobra.Command{
		Use:   "import-key",
		Short: "Store an API key for a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				if opts.dryRun {
					source, ok := detectAPIKeyInputSource(apiKey, apiKeyStdin, true)
					if !ok {
						return fmt.Errorf("api key is required; run tmc auth login, pass --api-key-stdin, pass --api-key, or set TMCOPILOT_API_KEY")
					}
					return writeAPIKeySetupDryRun(cmd, opts, source, check, name)
				}
				resolvedAPIKey, ok, err := readAPIKeyInput(cmd, apiKey, apiKeyStdin, true)
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("api key is required; run tmc auth login, pass --api-key-stdin, pass --api-key, or set TMCOPILOT_API_KEY")
				}
				source, _ := detectAPIKeyInputSource(apiKey, apiKeyStdin, true)
				return saveAPIKeyAndMaybeCheckForProfile(cmd, opts, resolvedAPIKey, check, name, source)
			})
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key value")
	cmd.Flags().BoolVar(&apiKeyStdin, "api-key-stdin", false, "read API key from stdin")
	cmd.Flags().StringVar(&name, "name", "", "profile name")
	cmd.Flags().BoolVar(&check, "check", false, "verify the stored API key with /auth/me")
	return cmd
}

func newAuthStatusCommand(opts *globalOptions) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				result := map[string]any{
					"profile":        rt.ProfileName,
					"endpoint":       rt.Profile.Endpoint,
					"workspace_id":   rt.Profile.WorkspaceID,
					"device_uuid":    rt.Profile.DeviceUUID,
					"has_api_key":    rt.APIKey != "",
					"api_key_source": rt.APIKeySrc,
				}
				if rt.APIKey == "" {
					result["next"] = "run `tmc setup` in a browser terminal or `tmc setup --no-wait` in an agent environment"
				}
				if check && rt.APIKey != "" {
					resp, err := rt.Client.Do(cmd.Context(), "GET", "/auth/me", nil, nil)
					if err != nil {
						return err
					}
					result["check"] = map[string]any{
						"ok":          true,
						"status_code": resp.StatusCode,
					}
					result["verified"] = true
				}
				return writeResult(rt, result, nil)
			})
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "call /auth/me to verify credentials")
	return cmd
}

func newAuthWhoamiCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "GET", "/auth/me", nil, nil)
		},
	}
}

func newAuthWorkspacesCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "workspaces",
		Short: "List accessible workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "GET", "/auth/workspaces", nil, nil)
		},
	}
}

func newAuthLogoutCommand(opts *globalOptions) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove locally stored CLI credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				if opts.dryRun {
					return writeResult(rt, map[string]any{
						"dry_run":          true,
						"profile":          rt.ProfileName,
						"all_profiles":     all,
						"credential_store": mustCredentialsPath(),
						"actions": []map[string]any{
							{
								"type":    "delete_local_credentials",
								"profile": rt.ProfileName,
								"all":     all,
							},
						},
					}, nil)
				}
				creds, err := config.LoadCredentials()
				if err != nil {
					return err
				}
				removed := 0
				if all {
					removed = len(creds.Profiles)
					creds.Profiles = map[string]config.Credential{}
				} else if _, ok := creds.Profiles[rt.ProfileName]; ok {
					delete(creds.Profiles, rt.ProfileName)
					removed = 1
				}
				if err := config.SaveCredentials(creds); err != nil {
					return err
				}
				result := map[string]any{
					"profile":          rt.ProfileName,
					"removed":          removed,
					"credential_store": mustCredentialsPath(),
				}
				if rt.APIKeySrc == "env" {
					result["env_override_active"] = true
					result["next"] = "unset TMCOPILOT_API_KEY or TMC_API_KEY to fully log out of this shell"
				}
				return writeResult(rt, result, nil)
			})
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "remove credentials for all profiles")
	return cmd
}

func newAuthUISettingsCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "ui-settings",
		Short: "Get dashboard UI settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "GET", "/auth/ui-settings", nil, nil)
		},
	}
}

func newAuthNotificationPreferencesCommand(opts *globalOptions) *cobra.Command {
	var data string
	cmd := &cobra.Command{
		Use:   "notification-preferences",
		Short: "Manage notification preferences",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Get notification preferences",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "GET", "/auth/notification-preferences", nil, nil)
		},
	})
	update := &cobra.Command{
		Use:   "update",
		Short: "Update notification preferences",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readDataArg(data)
			if err != nil {
				return err
			}
			if body == nil {
				return fmt.Errorf("--data is required")
			}
			return callAPIAndWrite(cmd, opts, "PUT", "/auth/notification-preferences", nil, body)
		},
	}
	update.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	cmd.AddCommand(update)
	return cmd
}

func newAuthCollaboratorsCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collaborators",
		Short: "Manage workspace collaborators",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List collaborators",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "GET", "/auth/collaborators", nil, nil)
		},
	})
	var inviteData string
	invite := &cobra.Command{
		Use:   "invite",
		Short: "Create collaborator invitation",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readDataArg(inviteData)
			if err != nil {
				return err
			}
			if body == nil {
				return fmt.Errorf("--data is required")
			}
			return callAPIAndWrite(cmd, opts, "POST", "/auth/collaborators/invitations", nil, body)
		},
	}
	invite.Flags().StringVar(&inviteData, "data", "", "JSON request body or @file")
	cmd.AddCommand(invite)
	cmd.AddCommand(&cobra.Command{
		Use:   "accept <token>",
		Short: "Accept collaborator invitation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "POST", "/auth/collaborators/invitations/"+url.PathEscape(args[0])+"/accept", nil, nil)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "delete-invitation <id>",
		Short: "Delete collaborator invitation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "DELETE", "/auth/collaborators/invitations/"+url.PathEscape(args[0]), nil, nil)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "remove <id>",
		Short: "Remove collaborator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "DELETE", "/auth/collaborators/"+url.PathEscape(args[0]), nil, nil)
		},
	})
	var roleData string
	role := &cobra.Command{
		Use:   "role <id>",
		Short: "Update collaborator role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readDataArg(roleData)
			if err != nil {
				return err
			}
			if body == nil {
				return fmt.Errorf("--data is required")
			}
			return callAPIAndWrite(cmd, opts, "PUT", "/auth/collaborators/"+url.PathEscape(args[0])+"/role", nil, body)
		},
	}
	role.Flags().StringVar(&roleData, "data", "", "JSON request body or @file")
	cmd.AddCommand(role)
	return cmd
}

func newAuthAPIKeysCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api-keys",
		Short: "Manage API keys",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List API keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "GET", "/auth/api-keys", nil, nil)
		},
	})
	var createName string
	var expiresIn int64
	create := &cobra.Command{
		Use:   "create",
		Short: "Create an API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				if strings.TrimSpace(createName) == "" {
					return fmt.Errorf("--name is required")
				}
				body := openapi.APIKeyCreateRequest{Name: createName, ExpiresIn: expiresIn}
				return executeAPIAndWrite(cmd, opts, "POST", "/auth/api-keys", nil, body)
			})
		},
	}
	create.Flags().StringVar(&createName, "name", "", "API key name")
	create.Flags().Int64Var(&expiresIn, "expires-in", 0, "expiry in seconds")
	cmd.AddCommand(create)

	cmd.AddCommand(&cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke an API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "DELETE", "/auth/api-keys/"+url.PathEscape(args[0]), nil, nil)
		},
	})
	return cmd
}

func mustCredentialsPath() string {
	path, err := config.CredentialsPath()
	if err != nil {
		return ""
	}
	return path
}
