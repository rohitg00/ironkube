package cli

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

func NewEtcdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "etcd",
		Short: "Manage etcd cluster operations",
	}

	cmd.AddCommand(newEtcdHealthCmd())
	cmd.AddCommand(newEtcdBackupCmd())
	cmd.AddCommand(newEtcdDefragCmd())

	return cmd
}

func newEtcdHealthCmd() *cobra.Command {
	var (
		name     string
		stateDir string
	)

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check etcd cluster health",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("cluster name required: use --name")
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

			fmt.Fprintln(cmd.OutOrStdout(), "etcd health check requires SSH access to control plane nodes.")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}

func newEtcdBackupCmd() *cobra.Command {
	var (
		name       string
		stateDir   string
		outputPath string
	)

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create an etcd snapshot backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("cluster name required: use --name")
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

			if outputPath == "" {
				outputPath = "/tmp/etcd-backup.db"
			}

			fmt.Fprintf(cmd.OutOrStdout(), "etcd backup to %s requires SSH access to control plane nodes.\n", outputPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output path for snapshot (default: /tmp/etcd-backup.db)")
	return cmd
}

func newEtcdDefragCmd() *cobra.Command {
	var (
		name     string
		stateDir string
	)

	cmd := &cobra.Command{
		Use:   "defrag",
		Short: "Defragment etcd storage across the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("cluster name required: use --name")
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

			fmt.Fprintln(cmd.OutOrStdout(), "etcd defrag requires SSH access to control plane nodes.")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}
