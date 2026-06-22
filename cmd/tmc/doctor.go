package tmc

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/openapi"
	"github.com/huski-inc/tmcopilot-cli/internal/version"
)

func newDoctorCommand(opts *globalOptions) *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose CLI configuration and API connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				result := baseDoctorResult(rt)
				result["network"] = runDoctorRequest(cmd, rt, http.MethodGet, "/version", nil)
				if rt.APIKey != "" {
					result["auth"] = runDoctorRequest(cmd, rt, http.MethodGet, "/auth/me", nil)
				} else {
					result["auth"] = map[string]any{
						"ok":      false,
						"message": "missing API key",
					}
				}
				if err := writeResult(rt, result, nil); err != nil {
					return err
				}
				if !strict {
					return nil
				}
				return firstDoctorFailure(map[string]map[string]any{
					"network": result["network"].(map[string]any),
					"auth":    result["auth"].(map[string]any),
				})
			})
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", true, "return non-zero when a check fails")
	cmd.AddCommand(newDoctorNetworkCommand(opts))
	cmd.AddCommand(newDoctorAuthCommand(opts))
	return cmd
}

func newDoctorNetworkCommand(opts *globalOptions) *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Check endpoint reachability",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				result := baseDoctorResult(rt)
				result["network"] = runDoctorRequest(cmd, rt, http.MethodGet, "/version", nil)
				if err := writeResult(rt, result, nil); err != nil {
					return err
				}
				if !strict {
					return nil
				}
				return doctorFailure("network", result["network"].(map[string]any))
			})
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", true, "return non-zero when the check fails")
	return cmd
}

func newDoctorAuthCommand(opts *globalOptions) *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Check API key authentication",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				result := baseDoctorResult(rt)
				if rt.APIKey == "" {
					result["auth"] = map[string]any{
						"ok":      false,
						"message": "missing API key",
					}
				} else {
					result["auth"] = runDoctorRequest(cmd, rt, http.MethodGet, "/auth/me", nil)
				}
				if err := writeResult(rt, result, nil); err != nil {
					return err
				}
				if !strict {
					return nil
				}
				return doctorFailure("auth", result["auth"].(map[string]any))
			})
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", true, "return non-zero when the check fails")
	return cmd
}

func baseDoctorResult(rt *runtimeContext) map[string]any {
	return map[string]any{
		"cli": map[string]any{
			"version": version.Version,
			"commit":  version.Commit,
			"date":    version.Date,
		},
		"openapi": map[string]any{
			"source_hash": openapi.SourceHash,
			"source_path": openapi.SourcePath,
			"endpoints":   len(openapi.Endpoints),
		},
		"profile": map[string]any{
			"name":         rt.ProfileName,
			"endpoint":     rt.Profile.Endpoint,
			"workspace_id": rt.Profile.WorkspaceID,
		},
		"auth": map[string]any{
			"ok":             rt.APIKey != "",
			"api_key_source": rt.APIKeySrc,
		},
	}
}

func runDoctorRequest(cmd *cobra.Command, rt *runtimeContext, method string, path string, body any) map[string]any {
	resp, err := rt.Client.Do(cmd.Context(), method, path, nil, body)
	if err != nil {
		return map[string]any{
			"ok":      false,
			"message": err.Error(),
		}
	}
	return map[string]any{
		"ok":          true,
		"status_code": resp.StatusCode,
		"trace_id":    resp.Headers.Get("X-Trace-ID"),
	}
}

func firstDoctorFailure(checks map[string]map[string]any) error {
	for _, name := range []string{"network", "auth"} {
		if err := doctorFailure(name, checks[name]); err != nil {
			return err
		}
	}
	return nil
}

func doctorFailure(name string, result map[string]any) error {
	if result == nil {
		return fmt.Errorf("doctor %s failed", name)
	}
	if ok, _ := result["ok"].(bool); ok {
		return nil
	}
	message, _ := result["message"].(string)
	if message == "" {
		message = "check failed"
	}
	return fmt.Errorf("doctor %s failed: %s", name, message)
}
