package tmc

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/output"
	"github.com/huski-inc/tmcopilot-cli/internal/skills"
)

func newSkillsCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Read embedded agent skill guidance",
		Long:  "Read agent-readable skill guidance embedded in this CLI build, so it stays in sync with the command version.",
	}
	cmd.AddCommand(newSkillsListCommand(opts))
	cmd.AddCommand(newSkillsReadCommand(opts))
	return cmd
}

func newSkillsListCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list [skill[/path]]",
		Short: "List skills or files under a skill path",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				if len(args) == 0 {
					items, err := skills.List()
					if err != nil {
						return err
					}
					return writeResult(rt, map[string]any{
						"skills": items,
						"count":  len(items),
					}, nil)
				}
				name, relPath := splitSkillTarget(args[0])
				entries, err := skills.ListPath(name, relPath)
				if err != nil {
					return err
				}
				return writeResult(rt, map[string]any{
					"skill":   name,
					"path":    relPath,
					"entries": entries,
					"count":   len(entries),
				}, nil)
			})
		},
	}
}

func newSkillsReadCommand(opts *globalOptions) *cobra.Command {
	var asJSON bool
	var plain bool
	cmd := &cobra.Command{
		Use:   "read <skill[/path]> [path]",
		Short: "Read a skill's SKILL.md or a reference file",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				name, relPath := splitSkillTarget(args[0])
				if len(args) == 2 {
					relPath = args[1]
				}
				raw, err := skills.Read(name, relPath)
				if err != nil {
					return err
				}
				isMainSkill := strings.TrimSpace(relPath) == ""
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				if asJSON && strings.EqualFold(rt.Format, "raw") {
					rt.Format = "json"
				}
				if plain || (!asJSON && strings.EqualFold(rt.Format, "raw")) {
					if strings.TrimSpace(rt.OutputPath) != "" {
						if err := output.WriteRawFile(rt.OutputPath, raw); err != nil {
							return err
						}
						return output.WriteTo(cmd.OutOrStdout(), "json", "", map[string]any{
							"path":  rt.OutputPath,
							"bytes": len(raw),
						}, nil)
					}
					_, err = cmd.OutOrStdout().Write(raw)
					if err == nil && isMainSkill {
						fmt.Fprintln(cmd.ErrOrStderr(), skillReadGuidance(name))
					}
					return err
				}
				path := relPath
				if strings.TrimSpace(path) == "" {
					path = "SKILL.md"
				}
				data := map[string]any{
					"skill":   name,
					"path":    path,
					"content": string(raw),
				}
				if isMainSkill {
					data["guidance"] = skillReadGuidance(name)
				}
				return writeResult(rt, data, nil)
			})
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "force a JSON envelope; JSON is the default unless --format raw or --plain is set")
	cmd.Flags().BoolVar(&plain, "plain", false, "output raw markdown instead of a JSON envelope")
	return cmd
}

func skillReadGuidance(name string) string {
	return fmt.Sprintf("> Tip: read referenced files with `tmc skills read %s <relative-path>` so guidance stays version-matched with this CLI build; use `--format raw` only when plain markdown is required.", name)
}

func splitSkillTarget(value string) (string, string) {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "/")
	name, relPath, ok := strings.Cut(value, "/")
	if !ok {
		return value, ""
	}
	return name, relPath
}
