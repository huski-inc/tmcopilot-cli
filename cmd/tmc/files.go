package tmc

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newFilesCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "Work with file APIs",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callAPIAndWrite(cmd, opts, "GET", "/files", nil, nil)
		},
	})
	cmd.AddCommand(newFilePresignCommand(opts, "presign", "Create a file upload URL", "/files/presign"))
	cmd.AddCommand(newFilePresignCommand(opts, "upload-presign", "Create an upload presign URL", "/upload/presign"))
	return cmd
}

func newFilePresignCommand(opts *globalOptions, use string, short string, path string) *cobra.Command {
	var data string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readDataArg(data)
			if err != nil {
				return err
			}
			if body == nil {
				return fmt.Errorf("--data is required")
			}
			return callAPIAndWrite(cmd, opts, "POST", path, nil, body)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON request body or @file")
	return cmd
}
