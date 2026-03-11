package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print IronKube version",
		Run: func(cmd *cobra.Command, args []string) {
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ironkube version %s (%s)\n", Version, Commit)
			fmt.Fprintf(w, "go: %s\n", runtime.Version())
			fmt.Fprintf(w, "platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}
