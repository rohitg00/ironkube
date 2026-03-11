package cli

import (
	"context"
	"fmt"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/phases"
	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	var (
		configPath string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap a new Kubernetes cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			pipeline := engine.NewPipeline(
				&phases.ValidatePhase{Config: cfg},
			)

			if err := pipeline.Execute(context.Background()); err != nil {
				return err
			}

			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "Dry run: cluster %q validated (distro: %s, control-plane: %d, workers: %d)\n",
					cfg.Metadata.Name, cfg.Spec.Distro,
					len(cfg.Spec.ControlPlane.Nodes), workerCount(cfg))
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Cluster %q initialized successfully\n", cfg.Metadata.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "ironkube.yaml", "Path to cluster config file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate config without bootstrapping")
	return cmd
}

func workerCount(cfg *config.ClusterConfig) int {
	count := 0
	for _, pool := range cfg.Spec.Workers {
		count += len(pool.Nodes)
	}
	return count
}
