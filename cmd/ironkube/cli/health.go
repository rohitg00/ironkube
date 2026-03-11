package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check cluster health",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "Health check requires a running cluster. Use 'ironkube init' first.")
			return nil
		},
	}
}
