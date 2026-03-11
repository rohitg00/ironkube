package config

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

	if cfg.Spec.Distro != "k3s" && cfg.Spec.Distro != "kubeadm" {
		errs = append(errs, fmt.Sprintf("distro must be k3s or kubeadm, got %q", cfg.Spec.Distro))
	}

	if cfg.Spec.Version == "" {
		errs = append(errs, "spec.version is required")
	}

	if len(cfg.Spec.ControlPlane.Nodes) == 0 {
		errs = append(errs, "at least 1 control plane node is required")
	}

	replicas := cfg.Spec.ControlPlane.Replicas
	if replicas > 1 && replicas%2 == 0 {
		errs = append(errs, fmt.Sprintf("HA requires odd number of replicas, got %d", replicas))
	}

	if replicas != len(cfg.Spec.ControlPlane.Nodes) {
		errs = append(errs, fmt.Sprintf("replicas (%d) must match control plane node count (%d)", replicas, len(cfg.Spec.ControlPlane.Nodes)))
	}

	if cfg.Spec.Security.Profile != "minimal" && cfg.Spec.Security.Profile != "cis-hardened" {
		errs = append(errs, fmt.Sprintf("security profile must be minimal or cis-hardened, got %q", cfg.Spec.Security.Profile))
	}

	switch cfg.Spec.Networking.CNI {
	case "flannel", "cilium", "calico":
	default:
		errs = append(errs, fmt.Sprintf("CNI must be flannel, cilium, or calico, got %q", cfg.Spec.Networking.CNI))
	}

	for i, node := range cfg.Spec.ControlPlane.Nodes {
		if node.Host == "" {
			errs = append(errs, fmt.Sprintf("controlPlane.nodes[%d].host is required", i))
		}
	}

	for i, pool := range cfg.Spec.Workers {
		for j, node := range pool.Nodes {
			if node.Host == "" {
				errs = append(errs, fmt.Sprintf("workers[%d].nodes[%d].host is required", i, j))
			}
		}
	}

	if cfg.Spec.Networking.PodCIDR != "" {
		if _, _, err := net.ParseCIDR(cfg.Spec.Networking.PodCIDR); err != nil {
			errs = append(errs, fmt.Sprintf("invalid podCIDR %q: %v", cfg.Spec.Networking.PodCIDR, err))
		}
	}

	if cfg.Spec.Networking.SvcCIDR != "" {
		if _, _, err := net.ParseCIDR(cfg.Spec.Networking.SvcCIDR); err != nil {
			errs = append(errs, fmt.Sprintf("invalid serviceCIDR %q: %v", cfg.Spec.Networking.SvcCIDR, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}
