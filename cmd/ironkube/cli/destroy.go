package cli

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/phases"
	"github.com/spf13/cobra"
)

func NewDestroyCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Tear down a Kubernetes cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			pipeline := engine.NewPipeline(
				&phases.ValidatePhase{Config: cfg},
				&phases.ConnectPhase{Config: cfg},
				&phases.DestroyPhase{Config: cfg},
			)

			if err := pipeline.Execute(cmd.Context()); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Cluster %q destroyed successfully\n", cfg.Metadata.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "ironkube.yaml", "Path to cluster config file")
	return cmd
}
