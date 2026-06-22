package tmc

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSetupCommand(opts *globalOptions) *cobra.Command {
	var apiKey string
	var apiKeyStdin bool
	loginOpts := loginCredentialOptions{Check: true}

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up local TMCopilot CLI access",
		Long: `Set up local TMCopilot CLI access in one command.

By default, setup opens a browser authorization page, receives a one-time API
key from TMCopilot, stores it locally, and verifies the stored key. For scripts
or CI, pass an existing API key with --api-key-stdin or --api-key.`,
		Example: `  tmc setup
  tmc setup --no-browser
  tmc setup --no-wait
  tmc setup --request-id <request_id>
  printf '%s' "$TMCOPILOT_API_KEY" | tmc setup --api-key-stdin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				loginOptionRequested := strings.TrimSpace(loginOpts.Email) != "" ||
					loginOpts.PasswordStdin ||
					strings.TrimSpace(loginOpts.TurnstileResponse) != "" ||
					strings.TrimSpace(loginOpts.KeyName) != "" ||
					strings.TrimSpace(loginOpts.DeviceName) != "" ||
					loginOpts.NoBrowser ||
					loginOpts.NoWait ||
					strings.TrimSpace(loginOpts.RequestID) != "" ||
					loginOpts.ExpiresIn > 0

				if opts.dryRun {
					source, hasAPIKey := detectAPIKeyInputSource(apiKey, apiKeyStdin, false)
					if hasAPIKey {
						if loginOptionRequested {
							return fmt.Errorf("use either login options or API key options, not both")
						}
						return writeAPIKeySetupDryRun(cmd, opts, source, loginOpts.Check, "")
					}
					return writeLoginSetupDryRun(cmd, opts, loginOpts)
				}

				resolvedAPIKey, hasAPIKey, err := readAPIKeyInput(cmd, apiKey, apiKeyStdin, false)
				if err != nil {
					return err
				}
				if hasAPIKey {
					if loginOptionRequested {
						return fmt.Errorf("use either login options or API key options, not both")
					}
					return saveAPIKeyAndMaybeCheck(cmd, opts, resolvedAPIKey, loginOpts.Check)
				}
				return loginAndStoreAPIKey(cmd, opts, loginOpts)
			})
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "existing API key value")
	cmd.Flags().BoolVar(&apiKeyStdin, "api-key-stdin", false, "read existing API key from stdin")
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
