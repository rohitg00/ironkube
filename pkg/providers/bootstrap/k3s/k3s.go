package k3s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type K3s struct{}

func New() *K3s {
	return &K3s{}
}

func (k *K3s) Name() string {
	return "k3s"
}

func (k *K3s) ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	if !strings.HasPrefix(version, "v") {
		return fmt.Errorf("version must start with 'v': %s", version)
	}
	if !strings.Contains(version, "+k3s") {
		return fmt.Errorf("version must contain '+k3s' suffix: %s", version)
	}
	return nil
}

func (k *K3s) SupportedVersions() []string {
	return []string{
		"v1.31.2+k3s1",
		"v1.30.6+k3s1",
		"v1.30.5+k3s1",
		"v1.29.10+k3s1",
		"v1.29.9+k3s1",
		"v1.28.14+k3s1",
		"v1.28.13+k3s1",
		"v1.27.16+k3s1",
	}
}

func (k *K3s) InitControlPlane(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, token string) (*provider.BootstrapData, error) {
	var flags []string

	if cluster.HAEnabled {
		flags = append(flags, "--cluster-init")
	}
	if cluster.PodCIDR != "" {
		flags = append(flags, fmt.Sprintf("--cluster-cidr=%s", cluster.PodCIDR))
	}
	if cluster.ServiceCIDR != "" {
		flags = append(flags, fmt.Sprintf("--service-cidr=%s", cluster.ServiceCIDR))
	}

	cmd := "curl -sfL https://get.k3s.io | sh -s - server"
	if len(flags) > 0 {
		cmd += " " + strings.Join(flags, " ")
	}

	envVars := map[string]string{
		"INSTALL_K3S_VERSION": cluster.Version,
	}
	if token != "" {
		envVars["K3S_TOKEN"] = token
	}

	runAs := machine.Spec.User
	if runAs == "" {
		runAs = "root"
	}

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "install-k3s-server",
				Content: cmd,
				RunAs:   runAs,
			},
		},
		EnvVars: envVars,
		WaitCheck: &provider.WaitCondition{
			Command:  "k3s kubectl get nodes",
			Expected: "Ready",
			Interval: 10 * time.Second,
			Timeout:  2 * time.Minute,
		},
	}, nil
}

func (k *K3s) JoinControlPlane(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, joinEndpoint, token string) (*provider.BootstrapData, error) {
	cmd := "curl -sfL https://get.k3s.io | sh -s - server"

	envVars := map[string]string{
		"INSTALL_K3S_VERSION": cluster.Version,
		"K3S_URL":            joinEndpoint,
		"K3S_TOKEN":          token,
	}

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "join-k3s-server",
				Content: cmd,
				RunAs:   "root",
			},
		},
		EnvVars: envVars,
		WaitCheck: &provider.WaitCondition{
			Command:  "k3s kubectl get nodes",
			Expected: "Ready",
			Interval: 10 * time.Second,
			Timeout:  2 * time.Minute,
		},
	}, nil
}

func (k *K3s) JoinWorker(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, joinEndpoint, token string) (*provider.BootstrapData, error) {
	cmd := "curl -sfL https://get.k3s.io | sh -s - agent"

	envVars := map[string]string{
		"INSTALL_K3S_VERSION": cluster.Version,
		"K3S_URL":            joinEndpoint,
		"K3S_TOKEN":          token,
	}

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "join-k3s-agent",
				Content: cmd,
				RunAs:   "root",
			},
		},
		EnvVars: envVars,
		WaitCheck: &provider.WaitCondition{
			Command:  "k3s kubectl get nodes",
			Expected: "Ready",
			Interval: 10 * time.Second,
			Timeout:  2 * time.Minute,
		},
	}, nil
}

func (k *K3s) Uninstall(ctx context.Context, machine provider.Machine, role string) (*provider.BootstrapData, error) {
	script := "/usr/local/bin/k3s-uninstall.sh"
	if role == "worker" || role == "agent" {
		script = "/usr/local/bin/k3s-agent-uninstall.sh"
	}

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "uninstall-k3s",
				Content: script,
				RunAs:   "root",
			},
		},
	}, nil
}

func (k *K3s) Upgrade(ctx context.Context, machine provider.Machine, toVersion string) (*provider.BootstrapData, error) {
	cmd := fmt.Sprintf("curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=%s sh -", toVersion)

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "upgrade-k3s",
				Content: cmd,
				RunAs:   "root",
			},
		},
	}, nil
}

func (k *K3s) SecurityArgs(profile provider.SecurityProfile) []string {
	if profile == nil {
		return nil
	}
	if profile.Name() == "cis" {
		return []string{
			"--protect-kernel-defaults",
			"--secrets-encryption",
		}
	}
	return nil
}

func (k *K3s) KubeconfigPath() string {
	return "/etc/rancher/k3s/k3s.yaml"
}

func (k *K3s) TokenCommand() string {
	return "cat /var/lib/rancher/k3s/server/node-token"
}
