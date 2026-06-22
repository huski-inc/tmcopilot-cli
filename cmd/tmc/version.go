package tmc

import (
	"github.com/spf13/cobra"

	"github.com/huski-inc/tmcopilot-cli/internal/version"
)

func newVersionCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleCommand(cmd, func() error {
				rt, err := commandRuntime(cmd, opts, false)
				if err != nil {
					return err
				}
				return writeResult(rt, version.Current(), nil)
			})
		},
	}
}
