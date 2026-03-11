package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rohitg00/ironkube/pkg/lifecycle/certs"
	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

func NewCertsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certs",
		Short: "Manage cluster TLS certificates",
	}

	cmd.AddCommand(newCertsCheckCmd())
	cmd.AddCommand(newCertsRotateCmd())

	return cmd
}

func newCertsCheckCmd() *cobra.Command {
	var (
		name     string
		stateDir string
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check certificate expiry for control plane machines",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("cluster name required: use --name")
			}

			resolvedStateDir, err := resolveStateDir(stateDir)
			if err != nil {
				return err
			}

			backend := local.New(resolvedStateDir)
			clusterState, err := backend.Load(name)
			if err != nil {
				return fmt.Errorf("loading state for cluster %q: %w", name, err)
			}

			w := cmd.OutOrStdout()
			mgr := certs.New(nil)

			for _, m := range clusterState.Machines {
				if m.Role != "control-plane" {
					continue
				}

				machine := provider.Machine{
					ID: m.ID,
					Spec: provider.MachineSpec{
						Host: m.Host,
						User: "root",
						Port: 22,
					},
				}

				certList, err := mgr.CheckExpiry(cmd.Context(), machine)
				if err != nil {
					fmt.Fprintf(w, "Error checking certs on %s: %v\n", m.Host, err)
					continue
				}

				fmt.Fprintf(w, "\nCertificates on %s:\n", m.Host)
				if len(certList) == 0 {
					fmt.Fprintln(w, "  (none found)")
					continue
				}
				for _, c := range certList {
					remaining := time.Until(c.ExpiresAt)
					days := int(remaining.Hours() / 24)
					fmt.Fprintf(w, "  %-35s expires in %d days (%s)\n", c.Name, days, c.ExpiresAt.Format("2006-01-02"))
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}

func newCertsRotateCmd() *cobra.Command {
	var (
		name     string
		stateDir string
	)

	cmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotate certificates on all control plane machines",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("cluster name required: use --name")
			}

			resolvedStateDir, err := resolveStateDir(stateDir)
			if err != nil {
				return err
			}

			backend := local.New(resolvedStateDir)
			clusterState, err := backend.Load(name)
			if err != nil {
				return fmt.Errorf("loading state for cluster %q: %w", name, err)
			}

			w := cmd.OutOrStdout()
			mgr := certs.New(nil)

			for _, m := range clusterState.Machines {
				if m.Role != "control-plane" {
					continue
				}

				machine := provider.Machine{
					ID: m.ID,
					Spec: provider.MachineSpec{
						Host: m.Host,
						User: "root",
						Port: 22,
					},
				}

				fmt.Fprintf(w, "Rotating certificates on %s...\n", m.Host)
				if err := mgr.Rotate(cmd.Context(), machine, "kubeadm"); err != nil {
					fmt.Fprintf(w, "  Error: %v\n", err)
					continue
				}
				fmt.Fprintf(w, "  Done.\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")
	return cmd
}

func resolveStateDir(stateDir string) (string, error) {
	if stateDir != "" {
		return stateDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".ironkube", "state"), nil
}
