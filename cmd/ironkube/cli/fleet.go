package cli

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/fleet"
	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

func NewFleetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fleet",
		Short: "Manage fleet of clusters",
	}

	cmd.AddCommand(newFleetStatusCmd())
	cmd.AddCommand(newFleetDiffCmd())
	cmd.AddCommand(newFleetUpgradeCmd())

	return cmd
}

func newFleetStatusCmd() *cobra.Command {
	var (
		configPath string
		stateDir   string
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all clusters in the fleet",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				return fmt.Errorf("fleet config required: use --config")
			}

			fleetCfg, err := fleet.LoadFleetConfig(configPath)
			if err != nil {
				return fmt.Errorf("loading fleet config: %w", err)
			}

			resolvedDir, err := resolveStateDir(stateDir)
			if err != nil {
				return err
			}

			backend := local.New(resolvedDir)
			inv := fleet.NewInventory(backend)

			summaries, err := inv.Collect(cmd.Context(), fleetCfg)
			if err != nil {
				return fmt.Errorf("collecting fleet status: %w", err)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Fleet: %s\n\n", fleetCfg.Metadata.Name)

			for _, s := range summaries {
				if s.Available {
					machineCount := 0
					version := "unknown"
					if s.State != nil {
						machineCount = len(s.State.Machines)
						if s.State.Metadata.Version != "" {
							version = s.State.Metadata.Version
						}
					}
					fmt.Fprintf(w, "  %-20s available   %s  (%d machines)\n", s.Name, version, machineCount)
				} else {
					fmt.Fprintf(w, "  %-20s unavailable (%s)\n", s.Name, s.Error)
				}
			}

			fs := fleet.Summary(summaries)
			fmt.Fprintf(w, "\nTotal: %d clusters, %d available, %d machines (%d ready)\n",
				fs.TotalClusters, fs.AvailableClusters, fs.TotalMachines, fs.ReadyMachines)

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to fleet config")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}

func newFleetDiffCmd() *cobra.Command {
	var (
		configPath string
		stateDir   string
	)

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show differences for all clusters in the fleet",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				return fmt.Errorf("fleet config required: use --config")
			}

			fleetCfg, err := fleet.LoadFleetConfig(configPath)
			if err != nil {
				return fmt.Errorf("loading fleet config: %w", err)
			}

			resolvedDir, err := resolveStateDir(stateDir)
			if err != nil {
				return err
			}

			backend := local.New(resolvedDir)
			inv := fleet.NewInventory(backend)
			op := fleet.NewOperator(inv, backend)

			results, err := op.DiffAll(cmd.Context(), fleetCfg)
			if err != nil {
				return fmt.Errorf("running fleet diff: %w", err)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Fleet: %s\n\n", fleetCfg.Metadata.Name)

			allInSync := true
			for name, result := range results {
				if result.InSync {
					fmt.Fprintf(w, "  %s: in sync\n", name)
				} else {
					allInSync = false
					fmt.Fprintf(w, "  %s: %d actions needed\n", name, len(result.Actions))
					for _, a := range result.Actions {
						fmt.Fprintf(w, "    [%s] %s\n", a.Type, a.Description)
					}
				}
			}

			if allInSync {
				fmt.Fprintln(w, "\nAll clusters are in sync.")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to fleet config")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}

func newFleetUpgradeCmd() *cobra.Command {
	var (
		configPath string
		stateDir   string
		version    string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade all clusters in the fleet to a target version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				return fmt.Errorf("fleet config required: use --config")
			}
			if version == "" {
				return fmt.Errorf("target version required: use --version")
			}

			fleetCfg, err := fleet.LoadFleetConfig(configPath)
			if err != nil {
				return fmt.Errorf("loading fleet config: %w", err)
			}

			resolvedDir, err := resolveStateDir(stateDir)
			if err != nil {
				return err
			}

			backend := local.New(resolvedDir)
			inv := fleet.NewInventory(backend)
			op := fleet.NewOperator(inv, backend)

			result, err := op.UpgradeAll(cmd.Context(), fleetCfg, version, fleet.FleetUpgradeOpts{
				DryRun: dryRun,
			})
			if err != nil {
				return fmt.Errorf("running fleet upgrade: %w", err)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Fleet upgrade to %s:\n\n", version)

			for _, cr := range result.Clusters {
				if cr.Success {
					fmt.Fprintf(w, "  %s: %d upgrade actions\n", cr.Name, len(cr.Actions))
					for _, a := range cr.Actions {
						fmt.Fprintf(w, "    [%s] %s\n", a.Type, a.Description)
					}
				} else {
					fmt.Fprintf(w, "  %s: FAILED (%s)\n", cr.Name, cr.Error)
				}
			}

			fmt.Fprintf(w, "\nSucceeded: %d, Failed: %d\n", result.Succeeded, result.Failed)

			if dryRun {
				fmt.Fprintln(w, "\nDry run: no changes applied.")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to fleet config")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	cmd.Flags().StringVar(&version, "version", "", "Target Kubernetes version")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be upgraded without applying")
	return cmd
}
