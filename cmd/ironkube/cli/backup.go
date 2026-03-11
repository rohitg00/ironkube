package cli

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

func NewBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage cluster backups",
	}

	cmd.AddCommand(newBackupCreateCmd())
	cmd.AddCommand(newBackupRestoreCmd())

	return cmd
}

func newBackupCreateCmd() *cobra.Command {
	var (
		name     string
		output   string
		stateDir string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an etcd snapshot and optional state backup",
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

			if output == "" {
				output = "/tmp"
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Creating backup of cluster %q to %s requires SSH access.\n", name, output)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&output, "output", "", "Output directory for backup (default: /tmp)")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}

func newBackupRestoreCmd() *cobra.Command {
	var (
		name     string
		snapshot string
		stateDir string
	)

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore cluster from an etcd snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("cluster name required: use --name")
			}
			if snapshot == "" {
				return fmt.Errorf("snapshot path required: use --snapshot")
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

			fmt.Fprintf(cmd.OutOrStdout(), "Restoring cluster %q from %s requires SSH access.\n", name, snapshot)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "Path to etcd snapshot")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}
