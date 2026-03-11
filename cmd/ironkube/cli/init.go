package cli

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/phases"
	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	var (
		configPath     string
		dryRun         bool
		kubeconfigPath string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap a new Kubernetes cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if dryRun {
				validatePipeline := engine.NewPipeline(
					&phases.ValidatePhase{Config: cfg},
				)
				if err := validatePipeline.Execute(cmd.Context()); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Dry run: cluster %q validated (distro: %s, control-plane: %d, workers: %d)\n",
					cfg.Metadata.Name, cfg.Spec.Distro,
					len(cfg.Spec.ControlPlane.Nodes), workerCount(cfg))
				return nil
			}

			pipelinePhases := []engine.Phase{
				&phases.ValidatePhase{Config: cfg},
				&phases.ConnectPhase{Config: cfg},
				&phases.BootstrapPhase{Config: cfg},
				&phases.FetchKubeconfigPhase{Config: cfg, OutputPath: kubeconfigPath},
			}

			if cfg.Spec.Addons.CNI != "" || cfg.Spec.Addons.CertManager || cfg.Spec.Addons.Monitoring {
				pipelinePhases = append(pipelinePhases, &phases.AddonsPhase{Config: cfg})
			}

			pipeline := engine.NewPipeline(pipelinePhases...)

			if err := pipeline.Execute(cmd.Context()); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Cluster %q initialized successfully\n", cfg.Metadata.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "ironkube.yaml", "Path to cluster config file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate config without bootstrapping")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to save kubeconfig (default: ~/.kube/config)")
	return cmd
}

func workerCount(cfg *config.ClusterConfig) int {
	count := 0
	for _, pool := range cfg.Spec.Workers {
		count += len(pool.Nodes)
	}
	return count
}
