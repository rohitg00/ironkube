package cli

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/health"
	"github.com/rohitg00/ironkube/pkg/phases"
	"github.com/spf13/cobra"
)

func NewHealthCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check cluster health",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			pipeline := engine.NewPipeline(
				&phases.ValidatePhase{Config: cfg},
				&phases.ConnectPhase{Config: cfg},
				&phases.HealthCheckPhase{Config: cfg},
			)

			if err := pipeline.Execute(cmd.Context()); err != nil {
				return err
			}

			v, ok := pipeline.State().Get("cluster_health")
			if !ok {
				return fmt.Errorf("health check results not found")
			}
			ch := v.(*health.ClusterHealth)

			fmt.Fprintf(cmd.OutOrStdout(), "Cluster: %s\n\n", ch.ClusterName)
			for _, r := range ch.Results {
				icon := "+"
				if r.Status == health.StatusUnhealthy {
					icon = "x"
				} else if r.Status == health.StatusUnknown {
					icon = "?"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  [%s] %-25s %s\n", icon, r.Name, r.Message)
			}

			summary := ch.Summary()
			fmt.Fprintf(cmd.OutOrStdout(), "\n%d/%d healthy\n", summary.Healthy, summary.Total)

			if !ch.IsHealthy() {
				return fmt.Errorf("cluster is not healthy")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "ironkube.yaml", "Path to cluster config file")
	return cmd
}
