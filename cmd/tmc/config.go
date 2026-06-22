package tmc

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/config"
)

func newConfigCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage local CLI configuration",
	}
	cmd.AddCommand(newConfigInitCommand(opts))
	cmd.AddCommand(newConfigShowCommand(opts))
	cmd.AddCommand(newConfigSetCommand(opts))
	cmd.AddCommand(newConfigProfileCommand(opts))
	return cmd
}

func newConfigInitCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a local config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				cfg := config.DefaultConfig()
				if opts.endpoint != "" {
					profile := cfg.Profiles[cfg.CurrentProfile]
					profile.Endpoint = config.NormalizeEndpoint(opts.endpoint)
					cfg.Profiles[cfg.CurrentProfile] = profile
				}
				if err := config.Save(cfg); err != nil {
					return err
				}
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				return writeResult(rt, map[string]any{
					"config_path": mustConfigPath(),
					"profile":     cfg.CurrentProfile,
					"endpoint":    cfg.Profiles[cfg.CurrentProfile].Endpoint,
				}, nil)
			})
		},
	}
	return cmd
}

func newConfigShowCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show local config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				return writeResult(rt, rt.Config, map[string]any{"config_path": mustConfigPath()})
			})
		},
	}
}

func newConfigSetCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "set <endpoint|format|workspace> <value>",
		Short: "Set a value on the active profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				profileName, profile := cfg.ActiveProfile(opts.profile)
				switch args[0] {
				case "endpoint":
					profile.Endpoint = config.NormalizeEndpoint(args[1])
				case "format":
					profile.Format = args[1]
				case "workspace":
					profile.WorkspaceID = args[1]
				default:
					return fmt.Errorf("unsupported config key %q", args[0])
				}
				cfg.Profiles[profileName] = profile
				if err := config.Save(cfg); err != nil {
					return err
				}
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				return writeResult(rt, map[string]any{"profile": profileName, "value": profile}, nil)
			})
		},
	}
}

func newConfigProfileCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage config profiles",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				names := make([]string, 0, len(rt.Config.Profiles))
				for name := range rt.Config.Profiles {
					names = append(names, name)
				}
				sort.Strings(names)
				return writeResult(rt, map[string]any{
					"current_profile": rt.Config.CurrentProfile,
					"profiles":        names,
				}, nil)
			})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "use <name>",
		Short: "Switch current profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				if _, ok := cfg.Profiles[args[0]]; !ok {
					return fmt.Errorf("profile %q does not exist", args[0])
				}
				cfg.CurrentProfile = args[0]
				if err := config.Save(cfg); err != nil {
					return err
				}
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				return writeResult(rt, map[string]any{"current_profile": args[0]}, nil)
			})
		},
	})
	add := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				endpoint := config.DefaultEndpoint
				if opts.endpoint != "" {
					endpoint = config.NormalizeEndpoint(opts.endpoint)
				}
				cfg.Profiles[args[0]] = config.Profile{Endpoint: endpoint}
				if err := config.Save(cfg); err != nil {
					return err
				}
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				return writeResult(rt, map[string]any{"profile": args[0], "endpoint": endpoint}, nil)
			})
		},
	}
	cmd.AddCommand(add)
	return cmd
}

func mustConfigPath() string {
	path, err := config.ConfigPath()
	if err != nil {
		return ""
	}
	return path
}
