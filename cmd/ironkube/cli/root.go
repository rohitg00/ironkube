package cli

import (
	"github.com/spf13/cobra"
)

var (
	Version = "0.0.0"
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
	cmd.AddCommand(NewApplyCmd())
	cmd.AddCommand(NewDestroyCmd())
	cmd.AddCommand(NewStatusCmd())
	cmd.AddCommand(NewDiffCmd())
	cmd.AddCommand(NewCertsCmd())
	cmd.AddCommand(NewEtcdCmd())
	cmd.AddCommand(NewUpgradeCmd())
	cmd.AddCommand(NewBackupCmd())

	return cmd
}

func Execute() error {
	return NewRootCmd().Execute()
}
