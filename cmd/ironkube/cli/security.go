package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rohitg00/ironkube/pkg/security"
	"github.com/rohitg00/ironkube/pkg/state/local"
	"github.com/spf13/cobra"
)

func NewSecurityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Security profiles and audit",
	}

	cmd.AddCommand(newSecurityProfilesCmd())
	cmd.AddCommand(newSecurityAuditCmd())

	return cmd
}

func newSecurityProfilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profiles",
		Short: "List available security profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			profiles := []struct {
				name        string
				description string
			}{
				{"minimal", "No additional flags (default)"},
				{"cis", "CIS Kubernetes Benchmark flags"},
				{"hardened", "CIS + additional hardening"},
			}

			fmt.Fprintln(w, "Available security profiles:")
			fmt.Fprintln(w, "")
			for _, p := range profiles {
				profile, err := security.GetProfile(p.name)
				if err != nil {
					continue
				}
				apiCount := len(profile.APIServerFlags())
				kubeletCount := len(profile.KubeletFlags())
				etcdCount := len(profile.EtcdFlags())
				fmt.Fprintf(w, "  %-12s %s (api-server: %d, kubelet: %d, etcd: %d flags)\n",
					p.name, p.description, apiCount, kubeletCount, etcdCount)
			}
			return nil
		},
	}
}

func newSecurityAuditCmd() *cobra.Command {
	var (
		name       string
		profileArg string
		stateDir   string
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit cluster against a security profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileArg == "" {
				return fmt.Errorf("--profile is required")
			}

			if name == "" {
				return fmt.Errorf("--name is required")
			}

			if stateDir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("determining home directory: %w", err)
				}
				stateDir = filepath.Join(home, ".ironkube", "state")
			}

			backend := local.New(stateDir)
			clusterState, err := backend.Load(name)
			if err != nil {
				return fmt.Errorf("loading state for cluster %q: %w", name, err)
			}

			currentFlags := make(map[string][]string)
			if clusterState.DesiredSpec != nil {
				if specMap, ok := clusterState.DesiredSpec.(map[string]any); ok {
					for key, val := range specMap {
						if flagList, ok := val.([]any); ok {
							for _, f := range flagList {
								if s, ok := f.(string); ok {
									currentFlags[key] = append(currentFlags[key], s)
								}
							}
						}
					}
				}
			}

			auditor := security.NewAuditor()
			result, err := auditor.Audit(profileArg, currentFlags)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Security Audit: %s (profile: %s)\n", name, result.Profile)
			fmt.Fprintf(w, "Score: %.1f%% (%d passed, %d failed)\n\n", result.Score, result.Passed, result.Failed)

			if len(result.Findings) == 0 {
				fmt.Fprintln(w, "No findings - all checks passed.")
			} else {
				fmt.Fprintf(w, "Findings (%d):\n", len(result.Findings))
				for _, f := range result.Findings {
					fmt.Fprintf(w, "  [%s] %s: %s\n", f.Severity, f.Category, f.Description)
					fmt.Fprintf(w, "         Remediation: %s\n", f.Remediation)
				}
			}

			if cmd.Flags().Changed("json") {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Fprintln(w, string(data))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&profileArg, "profile", "", "Security profile to audit against")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "State directory (default: ~/.ironkube/state)")

	return cmd
}
