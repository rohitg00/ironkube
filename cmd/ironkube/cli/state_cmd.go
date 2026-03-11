package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

func NewStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Manage cluster state",
	}

	cmd.AddCommand(newStateListCmd())
	cmd.AddCommand(newStateInspectCmd())
	cmd.AddCommand(newStateDeleteCmd())

	return cmd
}

func stateDirectory(stateDir string) (string, error) {
	if stateDir != "" {
		return stateDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".ironkube", "state"), nil
}

func newStateListCmd() *cobra.Command {
	var stateDir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all clusters in state",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := stateDirectory(stateDir)
			if err != nil {
				return err
			}

			backend := local.New(dir)
			names, err := backend.List()
			if err != nil {
				return fmt.Errorf("listing state: %w", err)
			}

			w := cmd.OutOrStdout()
			if len(names) == 0 {
				fmt.Fprintln(w, "No clusters found in state.")
				return nil
			}

			fmt.Fprintf(w, "Clusters (%d):\n", len(names))
			for _, n := range names {
				fmt.Fprintf(w, "  %s\n", n)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")

	return cmd
}

func newStateInspectCmd() *cobra.Command {
	var (
		name     string
		stateDir string
	)

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Show raw state JSON for a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			dir, err := stateDirectory(stateDir)
			if err != nil {
				return err
			}

			backend := local.New(dir)
			clusterState, err := backend.Load(name)
			if err != nil {
				return fmt.Errorf("loading state for cluster %q: %w", name, err)
			}

			redacted := *clusterState
			if len(redacted.Kubeconfig) > 0 {
				redacted.Kubeconfig = []byte("[REDACTED]")
			}

			data, err := json.MarshalIndent(redacted, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling state: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")

	return cmd
}

func newStateDeleteCmd() *cobra.Command {
	var (
		name     string
		stateDir string
		yes      bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete cluster state",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			if !yes {
				return fmt.Errorf("--yes is required to confirm deletion")
			}

			dir, err := stateDirectory(stateDir)
			if err != nil {
				return err
			}

			backend := local.New(dir)
			if err := backend.Delete(name); err != nil {
				return fmt.Errorf("deleting state for cluster %q: %w", name, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "State for cluster %q deleted.\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm deletion")

	return cmd
}
