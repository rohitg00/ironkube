package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/rohitg00/ironkube/pkg/state"
	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

func NewApplyCmd() *cobra.Command {
	var (
		configPath string
		dryRun     bool
		stateDir   string
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply cluster configuration (create or update)",
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
				fmt.Fprintln(w, "Cluster is already in sync. No changes needed.")
				return nil
			}

			printActions(w, result)

			if dryRun {
				fmt.Fprintln(w, "\nDry run: no changes applied.")
				return nil
			}

			fmt.Fprintln(w, "\nApplying changes...")
			for _, action := range result.Actions {
				fmt.Fprintf(w, "  Executing: %s\n", action.Description)
			}

			clusterState.Metadata.UpdatedAt = time.Now()
			clusterState.Metadata.Version = cfg.Spec.Version

			for _, m := range desired.Machines {
				clusterState.SetMachine(m.Host, state.MachineState{
					ID:         m.Host,
					Host:       m.Host,
					Role:       m.Role,
					K8sVersion: cfg.Spec.Version,
					Status:     "ready",
					JoinedAt:   time.Now(),
				})
			}

			for _, a := range desired.Addons {
				clusterState.SetAddon(a.Name, state.AddonState{
					Name:      a.Name,
					Version:   a.Version,
					Installed: true,
					Ready:     true,
				})
			}

			clusterState.RecordOperation(state.OperationRecord{
				Type:       state.OpApply,
				Status:     state.OpStatusSuccess,
				StartedAt:  time.Now(),
				FinishedAt: time.Now(),
				Version:    cfg.Spec.Version,
			})

			if err := backend.Save(clusterState); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}

			fmt.Fprintf(w, "Cluster %q applied successfully.\n", cfg.Metadata.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "ironkube.yaml", "Path to cluster config")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without applying")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}

func configToDesired(cfg *v1alpha2.ClusterConfig) engine.DesiredState {
	var machines []provider.MachineSpec

	for _, n := range cfg.Spec.ControlPlane.Nodes {
		machines = append(machines, provider.MachineSpec{
			Role:    "control-plane",
			Host:    n.Host,
			User:    n.User,
			Port:    n.Port,
			KeyPath: n.KeyPath,
		})
	}

	for _, pool := range cfg.Spec.Workers {
		for _, n := range pool.Nodes {
			machines = append(machines, provider.MachineSpec{
				Role:    "worker",
				Host:    n.Host,
				User:    n.User,
				Port:    n.Port,
				KeyPath: n.KeyPath,
				Labels:  pool.Labels,
			})
		}
	}

	var addons []provider.AddonSpec
	for _, a := range cfg.Spec.Addons {
		if !a.Enabled {
			continue
		}
		addons = append(addons, provider.AddonSpec{
			Name:       a.Name,
			Version:    a.Version,
			Namespace:  a.Namespace,
			Repository: a.Repository,
			Chart:      a.Chart,
			Values:     a.Values,
		})
	}

	return engine.DesiredState{
		Machines: machines,
		Addons:   addons,
		Version:  cfg.Spec.Version,
	}
}

func stateToActual(st *state.ClusterState) engine.ActualState {
	actual := engine.ActualState{
		Machines: st.Machines,
		Addons:   st.Addons,
		Version:  st.Metadata.Version,
	}

	if actual.Machines == nil {
		actual.Machines = make(map[string]state.MachineState)
	}
	if actual.Addons == nil {
		actual.Addons = make(map[string]state.AddonState)
	}

	return actual
}

func printActions(w io.Writer, result engine.ReconcileResult) {
	fmt.Fprintf(w, "Planned actions (%d):\n", len(result.Actions))
	for i, action := range result.Actions {
		fmt.Fprintf(w, "  %d. [%s] %s\n", i+1, action.Type, action.Description)
	}
}
