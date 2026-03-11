package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewAirgapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "airgap",
		Short: "Manage airgap bundles for offline installations",
	}

	cmd.AddCommand(newAirgapBundleCmd())
	cmd.AddCommand(newAirgapLoadCmd())

	return cmd
}

func newAirgapBundleCmd() *cobra.Command {
	var (
		configPath string
		output     string
		arch       string
	)

	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Create an airgap bundle from cluster config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				return fmt.Errorf("config required: use --config")
			}
			if output == "" {
				return fmt.Errorf("output directory required: use --output")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Creating airgap bundle from %s to %s (arch: %s)\n", configPath, output, arch)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to cluster config file")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory for the bundle")
	cmd.Flags().StringVar(&arch, "arch", "amd64", "Target architecture (amd64 or arm64)")
	return cmd
}

func newAirgapLoadCmd() *cobra.Command {
	var (
		bundleDir string
		name      string
		stateDir  string
	)

	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load an airgap bundle onto cluster machines",
		RunE: func(cmd *cobra.Command, args []string) error {
			if bundleDir == "" {
				return fmt.Errorf("bundle directory required: use --bundle")
			}
			if name == "" {
				return fmt.Errorf("cluster name required: use --name")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Loading airgap bundle from %s for cluster %s\n", bundleDir, name)
			return nil
		},
	}

	cmd.Flags().StringVar(&bundleDir, "bundle", "", "Path to the airgap bundle directory")
	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}
