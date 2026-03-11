package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	var (
		name       string
		configPath string
		stateDir   string
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show cluster status from state",
		RunE: func(cmd *cobra.Command, args []string) error {
			clusterName := name

			if clusterName == "" && configPath != "" {
				cfg, err := v1alpha2.Load(configPath)
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
				clusterName = cfg.Metadata.Name
			}

			if clusterName == "" {
				return fmt.Errorf("cluster name required: use --name or --config")
			}

			if stateDir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("determining home directory: %w", err)
				}
				stateDir = filepath.Join(home, ".ironkube", "state")
			}

			backend := local.New(stateDir)
			w := cmd.OutOrStdout()

			clusterState, err := backend.Load(clusterName)
			if err != nil {
				fmt.Fprintf(w, "No state found for cluster %q. Run 'ironkube apply' first.\n", clusterName)
				return nil
			}

			fmt.Fprintf(w, "Cluster: %s\n", clusterState.Metadata.Name)
			fmt.Fprintf(w, "Version: %s\n", clusterState.Metadata.Version)
			fmt.Fprintf(w, "Created: %s\n", clusterState.Metadata.CreatedAt.Format("2006-01-02 15:04:05"))
			if !clusterState.Metadata.UpdatedAt.IsZero() {
				fmt.Fprintf(w, "Updated: %s\n", clusterState.Metadata.UpdatedAt.Format("2006-01-02 15:04:05"))
			}

			fmt.Fprintf(w, "\nMachines (%d):\n", len(clusterState.Machines))
			if len(clusterState.Machines) == 0 {
				fmt.Fprintln(w, "  (none)")
			}
			for _, m := range clusterState.Machines {
				fmt.Fprintf(w, "  %-16s %-14s %-8s %s\n", m.Host, m.Role, m.Status, m.K8sVersion)
			}

			fmt.Fprintf(w, "\nAddons (%d):\n", len(clusterState.Addons))
			if len(clusterState.Addons) == 0 {
				fmt.Fprintln(w, "  (none)")
			}
			for _, a := range clusterState.Addons {
				installedStr := "not installed"
				if a.Installed {
					installedStr = "installed"
				}
				fmt.Fprintf(w, "  %-20s %-12s %s\n", a.Name, a.Version, installedStr)
			}

			if clusterState.LastOperation.Type != "" {
				fmt.Fprintf(w, "\nLast operation: %s (%s)\n", clusterState.LastOperation.Type, clusterState.LastOperation.Status)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to cluster config (to read cluster name)")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}
