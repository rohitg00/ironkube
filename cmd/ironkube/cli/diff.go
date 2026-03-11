package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/state"
	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

var errDiffChanges = fmt.Errorf("changes detected")

func NewDiffCmd() *cobra.Command {
	var (
		configPath string
		stateDir   string
	)

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between config and current state",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := v1alpha2.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if stateDir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("determining home directory: %w", err)
				}
				stateDir = filepath.Join(home, ".ironkube", "state")
			}

			backend := local.New(stateDir)

			clusterState, err := backend.Load(cfg.Metadata.Name)
			if err != nil {
				clusterState = &state.ClusterState{
					Metadata: state.StateMetadata{
						Name:      cfg.Metadata.Name,
						CreatedAt: time.Now(),
					},
				}
			}

			desired := configToDesired(cfg)
			actual := stateToActual(clusterState)
			result := engine.Reconcile(desired, actual)

			w := cmd.OutOrStdout()

			if result.InSync {
				fmt.Fprintln(w, "Cluster is in sync. No changes needed.")
				return nil
			}

			printActions(w, result)
			return errDiffChanges
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "ironkube.yaml", "Path to cluster config")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}
