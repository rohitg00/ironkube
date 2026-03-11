package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

func NewDestroyCmd() *cobra.Command {
	var (
		name       string
		configPath string
		stateDir   string
		yes        bool
	)

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy a cluster and clean up resources",
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
				return fmt.Errorf("no state found for cluster %q: %w", clusterName, err)
			}

			machineCount := len(clusterState.Machines)
			addonCount := len(clusterState.Addons)

			fmt.Fprintf(w, "Cluster %q will be destroyed:\n", clusterName)
			fmt.Fprintf(w, "  Machines: %d\n", machineCount)
			fmt.Fprintf(w, "  Addons:   %d\n", addonCount)

			for _, m := range clusterState.Machines {
				fmt.Fprintf(w, "    - %s (%s) %s\n", m.Host, m.Role, m.Status)
			}

			for _, a := range clusterState.Addons {
				fmt.Fprintf(w, "    - %s %s\n", a.Name, a.Version)
			}

			if !yes {
				fmt.Fprintln(w, "\nUse --yes to confirm destruction.")
				return nil
			}

			fmt.Fprintln(w, "\nDestroying cluster...")

			if err := backend.Delete(clusterName); err != nil {
				return fmt.Errorf("deleting state: %w", err)
			}

			fmt.Fprintf(w, "Cluster %q destroyed successfully.\n", clusterName)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name to destroy")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to cluster config (to read cluster name)")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
