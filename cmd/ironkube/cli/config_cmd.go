package cli

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Config file operations",
	}

	cmd.AddCommand(newConfigValidateCmd())
	cmd.AddCommand(newConfigDefaultsCmd())

	return cmd
}

func newConfigValidateCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a cluster config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := v1alpha2.Load(configPath)
			if err != nil {
				return fmt.Errorf("config validation failed: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Config %s is valid.\n", configPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "ironkube.yaml", "Path to cluster config file")

	return cmd
}

func newConfigDefaultsCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "defaults",
		Short: "Show config with defaults applied",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := v1alpha2.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}

			fmt.Fprint(cmd.OutOrStdout(), string(data))
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "ironkube.yaml", "Path to cluster config file")

	return cmd
}
