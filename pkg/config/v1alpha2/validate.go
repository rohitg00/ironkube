package v1alpha2

import (
	"fmt"
	"net"
	"strings"
)

func Validate(cfg *ClusterConfig) error {
	var errs []string

	if cfg.Metadata.Name == "" {
		errs = append(errs, "metadata.name is required")
	}

	validDistros := map[string]bool{"k3s": true, "kubeadm": true, "k0s": true, "talos": true}
	if !validDistros[cfg.Spec.Distro] {
		errs = append(errs, fmt.Sprintf("distro must be one of [k3s, kubeadm, k0s, talos], got %q", cfg.Spec.Distro))
	}

	if cfg.Spec.Version == "" {
		errs = append(errs, "version is required")
	}

	validProviders := map[string]bool{"ssh": true, "hetzner": true, "docker": true}
	if !validProviders[cfg.Spec.Infrastructure.Provider] {
		errs = append(errs, fmt.Sprintf("infrastructure.provider must be one of [ssh, hetzner, docker], got %q", cfg.Spec.Infrastructure.Provider))
	}

	if cfg.Spec.ControlPlane.Replicas < 1 {
		errs = append(errs, "controlPlane.replicas must be >= 1")
	}

	if cfg.Spec.ControlPlane.Replicas > 1 && cfg.Spec.ControlPlane.Replicas%2 == 0 {
		errs = append(errs, fmt.Sprintf("HA requires odd number of control plane replicas, got %d", cfg.Spec.ControlPlane.Replicas))
	}

	if cfg.Spec.Infrastructure.Provider == "ssh" {
		for i, n := range cfg.Spec.ControlPlane.Nodes {
			if n.Host == "" {
				errs = append(errs, fmt.Sprintf("controlPlane.nodes[%d].host is required for ssh provider", i))
			}
		}
		for i, pool := range cfg.Spec.Workers {
			for j, n := range pool.Nodes {
				if n.Host == "" {
					errs = append(errs, fmt.Sprintf("workers[%d].nodes[%d].host is required for ssh provider", i, j))
				}
			}
		}
	}

	validProfiles := map[string]bool{"minimal": true, "cis-hardened": true, "soc2": true, "pci-dss": true}
	if !validProfiles[cfg.Spec.Security.Profile] {
		errs = append(errs, fmt.Sprintf("security.profile must be one of [minimal, cis-hardened, soc2, pci-dss], got %q", cfg.Spec.Security.Profile))
	}

	if cfg.Spec.Networking.PodCIDR != "" {
		if _, _, err := net.ParseCIDR(cfg.Spec.Networking.PodCIDR); err != nil {
			errs = append(errs, fmt.Sprintf("networking.podCIDR is not a valid CIDR: %s", cfg.Spec.Networking.PodCIDR))
		}
	}

	if cfg.Spec.Networking.ServiceCIDR != "" {
		if _, _, err := net.ParseCIDR(cfg.Spec.Networking.ServiceCIDR); err != nil {
			errs = append(errs, fmt.Sprintf("networking.serviceCIDR is not a valid CIDR: %s", cfg.Spec.Networking.ServiceCIDR))
		}
	}

	for i, addon := range cfg.Spec.Addons {
		if addon.Name == "" {
			errs = append(errs, fmt.Sprintf("addons[%d].name is required", i))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}
