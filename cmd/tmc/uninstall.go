package tmc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/config"
)

type uninstallPlan struct {
	BinaryPaths  []string `json:"binary_paths"`
	ConfigDir    string   `json:"config_dir,omitempty"`
	RemoveConfig bool     `json:"remove_config"`
}

type uninstallResult struct {
	DryRun        bool     `json:"dry_run"`
	Removed       []string `json:"removed,omitempty"`
	WouldRemove   []string `json:"would_remove,omitempty"`
	NotFound      []string `json:"not_found,omitempty"`
	ConfigKept    string   `json:"config_kept,omitempty"`
	ConfigRemoved string   `json:"config_removed,omitempty"`
	RemoveConfig  bool     `json:"remove_config"`
}

func newUninstallCommand(opts *globalOptions) *cobra.Command {
	var removeConfig bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall local TMCopilot CLI binaries",
		Long: `Uninstall local TMCopilot CLI binaries from the current install directory.

By default this removes the tmc and tmcopilot commands but keeps local config
and credentials under the TMCOPILOT_HOME directory. Pass --remove-config only
when you also want to delete local config and credentials.`,
		Example: `  tmc uninstall --dry-run
  tmc uninstall --yes
  tmc uninstall --yes --remove-config
  npx --yes @tmcopilot/cli@latest uninstall`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				plan, err := buildUninstallPlan(removeConfig)
				if err != nil {
					return err
				}
				if opts.dryRun {
					return writeResult(rt, dryRunUninstallResult(plan), nil)
				}
				if !opts.yes {
					return fmt.Errorf("uninstall removes local CLI binaries; rerun with --yes or preview with --dry-run")
				}
				result, err := runUninstallPlan(plan)
				if err != nil {
					return err
				}
				return writeResult(rt, result, nil)
			})
		},
	}
	cmd.Flags().BoolVar(&removeConfig, "remove-config", false, "also remove local config and credentials under TMCOPILOT_HOME")
	return cmd
}

func buildUninstallPlan(removeConfig bool) (uninstallPlan, error) {
	executable, err := os.Executable()
	if err != nil {
		return uninstallPlan{}, fmt.Errorf("resolve current executable: %w", err)
	}
	return buildUninstallPlanForExecutable(executable, removeConfig)
}

func buildUninstallPlanForExecutable(executable string, removeConfig bool) (uninstallPlan, error) {
	executable = filepath.Clean(executable)
	dir := filepath.Dir(executable)
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	paths := uniqueUninstallPaths([]string{
		filepath.Join(dir, "tmc"+ext),
		filepath.Join(dir, "tmcopilot"+ext),
	})
	configDir := ""
	if removeConfig {
		home, err := config.HomeDir()
		if err != nil {
			return uninstallPlan{}, err
		}
		configDir = filepath.Clean(home)
	}
	return uninstallPlan{
		BinaryPaths:  paths,
		ConfigDir:    configDir,
		RemoveConfig: removeConfig,
	}, nil
}

func uniqueUninstallPaths(paths []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(paths))
	for _, value := range paths {
		if value == "" {
			continue
		}
		clean := filepath.Clean(value)
		key := clean
		if runtime.GOOS == "windows" {
			key = filepath.ToSlash(clean)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, clean)
	}
	return out
}

func dryRunUninstallResult(plan uninstallPlan) uninstallResult {
	result := uninstallResult{
		DryRun:       true,
		WouldRemove:  append([]string{}, plan.BinaryPaths...),
		RemoveConfig: plan.RemoveConfig,
	}
	if plan.RemoveConfig {
		result.WouldRemove = append(result.WouldRemove, plan.ConfigDir)
	} else {
		result.ConfigKept = configDirForResult()
	}
	return result
}

func runUninstallPlan(plan uninstallPlan) (uninstallResult, error) {
	result := uninstallResult{RemoveConfig: plan.RemoveConfig}
	for _, path := range plan.BinaryPaths {
		err := os.Remove(path)
		switch {
		case err == nil:
			result.Removed = append(result.Removed, path)
		case errors.Is(err, os.ErrNotExist):
			result.NotFound = append(result.NotFound, path)
		default:
			return result, fmt.Errorf("remove %s: %w", path, err)
		}
	}
	if plan.RemoveConfig {
		if plan.ConfigDir == "" {
			return result, fmt.Errorf("config directory is empty")
		}
		if err := os.RemoveAll(plan.ConfigDir); err != nil {
			return result, fmt.Errorf("remove config directory %s: %w", plan.ConfigDir, err)
		}
		result.ConfigRemoved = plan.ConfigDir
	} else {
		result.ConfigKept = configDirForResult()
	}
	return result, nil
}

func configDirForResult() string {
	home, err := config.HomeDir()
	if err != nil {
		return ""
	}
	return filepath.Clean(home)
}
