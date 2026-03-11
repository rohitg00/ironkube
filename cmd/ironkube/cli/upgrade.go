package cli

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

func NewUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Manage cluster version upgrades",
	}

	cmd.AddCommand(newUpgradePlanCmd())
	cmd.AddCommand(newUpgradeApplyCmd())

	return cmd
}

func newUpgradePlanCmd() *cobra.Command {
	var (
		name     string
		version  string
		stateDir string
	)

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show what would be upgraded",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("cluster name required: use --name")
			}
			if version == "" {
				return fmt.Errorf("target version required: use --version")
			}

			resolvedStateDir, err := resolveStateDir(stateDir)
			if err != nil {
				return err
			}

			backend := local.New(resolvedStateDir)
			clusterState, err := backend.Load(name)
			if err != nil {
				return fmt.Errorf("loading state for cluster %q: %w", name, err)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Upgrade plan for cluster %q to version %s:\n", name, version)
			for _, m := range clusterState.Machines {
				fmt.Fprintf(w, "  %s (%s) %s -> %s\n", m.Host, m.Role, m.K8sVersion, version)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&version, "version", "", "Target Kubernetes version")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}

func newUpgradeApplyCmd() *cobra.Command {
	var (
		name     string
		version  string
		stateDir string
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Perform rolling upgrade on cluster nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("cluster name required: use --name")
			}
			if version == "" {
				return fmt.Errorf("target version required: use --version")
			}

			resolvedStateDir, err := resolveStateDir(stateDir)
			if err != nil {
				return err
			}

			backend := local.New(resolvedStateDir)
			_, err = backend.Load(name)
			if err != nil {
				return fmt.Errorf("loading state for cluster %q: %w", name, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Rolling upgrade of cluster %q to %s requires SSH access.\n", name, version)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&version, "version", "", "Target Kubernetes version")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}
