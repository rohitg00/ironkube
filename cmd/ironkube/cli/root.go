package cli

import (
	"github.com/spf13/cobra"
)

var (
	Version = "0.1.0"
	Commit  = "dev"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ironkube",
		Short: "Unified Kubernetes lifecycle management",
		Long:  "IronKube bootstraps production-hardened Kubernetes clusters across distributions with declarative config, security profiles, and health checks.",
	}

	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewInitCmd())
	cmd.AddCommand(NewHealthCmd())
	cmd.AddCommand(NewDestroyCmd())

	return cmd
}

func Execute() error {
	return NewRootCmd().Execute()
}
