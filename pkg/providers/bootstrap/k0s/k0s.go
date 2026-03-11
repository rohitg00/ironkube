package k0s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type K0s struct{}

func New() *K0s {
	return &K0s{}
}

func (k *K0s) Name() string {
	return "k0s"
}

func (k *K0s) ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	if !strings.HasPrefix(version, "v") {
		return fmt.Errorf("version must start with 'v': %s", version)
	}
	if !strings.Contains(version, "+k0s") {
		return fmt.Errorf("version must contain '+k0s' suffix: %s", version)
	}
	return nil
}

func (k *K0s) SupportedVersions() []string {
	return []string{
		"v1.31.2+k0s.0",
		"v1.30.6+k0s.0",
		"v1.30.5+k0s.0",
		"v1.29.10+k0s.0",
		"v1.29.9+k0s.0",
		"v1.28.14+k0s.0",
		"v1.28.13+k0s.0",
	}
}

func (k *K0s) generateConfig(cluster *provider.ClusterSpec) string {
	podCIDR := cluster.PodCIDR
	if podCIDR == "" {
		podCIDR = "10.244.0.0/16"
	}
	serviceCIDR := cluster.ServiceCIDR
	if serviceCIDR == "" {
		serviceCIDR = "10.96.0.0/12"
	}

	return fmt.Sprintf(`apiVersion: k0s.k0sproject.io/v1beta1
kind: ClusterConfig
metadata:
  name: %s
spec:
  network:
    podCIDR: %s
    serviceCIDR: %s
    provider: kuberouter`, cluster.Name, podCIDR, serviceCIDR)
}

func (k *K0s) InitControlPlane(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, token string) (*provider.BootstrapData, error) {
	config := k.generateConfig(cluster)

	installCmd := fmt.Sprintf("curl -sSLf https://get.k0s.sh | K0S_VERSION=%s sh", cluster.Version)
	controllerCmd := "k0s install controller --config /etc/k0s/k0s.yaml"
	if cluster.HAEnabled {
		controllerCmd += " --enable-worker"
	}
	startCmd := "k0s start"

	return &provider.BootstrapData{
		Files: []provider.FileToWrite{
			{
				Path:    "/etc/k0s/k0s.yaml",
				Content: config,
				Mode:    0644,
			},
		},
		Scripts: []provider.Script{
			{
				Name:    "install-k0s",
				Content: installCmd,
				RunAs:   "root",
			},
			{
				Name:    "setup-k0s-controller",
				Content: controllerCmd,
				RunAs:   "root",
			},
			{
				Name:    "start-k0s",
				Content: startCmd,
				RunAs:   "root",
			},
		},
		WaitCheck: &provider.WaitCondition{
			Command:  "k0s kubectl get nodes",
			Expected: "Ready",
			Interval: 10 * time.Second,
			Timeout:  3 * time.Minute,
		},
	}, nil
}

func (k *K0s) JoinControlPlane(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, joinEndpoint, token string) (*provider.BootstrapData, error) {
	installCmd := fmt.Sprintf("curl -sSLf https://get.k0s.sh | K0S_VERSION=%s sh", cluster.Version)
	controllerCmd := "k0s install controller --token-file /etc/k0s/token"
	startCmd := "k0s start"

	return &provider.BootstrapData{
		Files: []provider.FileToWrite{
			{
				Path:    "/etc/k0s/token",
				Content: token,
				Mode:    0600,
			},
		},
		Scripts: []provider.Script{
			{
				Name:    "install-k0s",
				Content: installCmd,
				RunAs:   "root",
			},
			{
				Name:    "join-k0s-controller",
				Content: controllerCmd,
				RunAs:   "root",
			},
			{
				Name:    "start-k0s",
				Content: startCmd,
				RunAs:   "root",
			},
		},
		WaitCheck: &provider.WaitCondition{
			Command:  "k0s kubectl get nodes",
			Expected: "Ready",
			Interval: 10 * time.Second,
			Timeout:  3 * time.Minute,
		},
	}, nil
}

func (k *K0s) JoinWorker(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, joinEndpoint, token string) (*provider.BootstrapData, error) {
	installCmd := fmt.Sprintf("curl -sSLf https://get.k0s.sh | K0S_VERSION=%s sh", cluster.Version)
	workerCmd := "k0s install worker --token-file /etc/k0s/token"
	startCmd := "k0s start"

	return &provider.BootstrapData{
		Files: []provider.FileToWrite{
			{
				Path:    "/etc/k0s/token",
				Content: token,
				Mode:    0600,
			},
		},
		Scripts: []provider.Script{
			{
				Name:    "install-k0s",
				Content: installCmd,
				RunAs:   "root",
			},
			{
				Name:    "join-k0s-worker",
				Content: workerCmd,
				RunAs:   "root",
			},
			{
				Name:    "start-k0s",
				Content: startCmd,
				RunAs:   "root",
			},
		},
		WaitCheck: &provider.WaitCondition{
			Command:  "k0s kubectl get nodes",
			Expected: "Ready",
			Interval: 10 * time.Second,
			Timeout:  3 * time.Minute,
		},
	}, nil
}

func (k *K0s) Uninstall(ctx context.Context, machine provider.Machine, role string) (*provider.BootstrapData, error) {
	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "stop-k0s",
				Content: "k0s stop",
				RunAs:   "root",
			},
			{
				Name:    "reset-k0s",
				Content: "k0s reset",
				RunAs:   "root",
			},
		},
	}, nil
}

func (k *K0s) Upgrade(ctx context.Context, machine provider.Machine, toVersion string) (*provider.BootstrapData, error) {
	installCmd := fmt.Sprintf("curl -sSLf https://get.k0s.sh | K0S_VERSION=%s sh", toVersion)

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "stop-k0s",
				Content: "k0s stop",
				RunAs:   "root",
			},
			{
				Name:    "install-k0s-upgrade",
				Content: installCmd,
				RunAs:   "root",
			},
			{
				Name:    "start-k0s",
				Content: "k0s start",
				RunAs:   "root",
			},
		},
	}, nil
}

func (k *K0s) SecurityArgs(profile provider.SecurityProfile) []string {
	if profile == nil {
		return nil
	}
	if profile.Name() == "default" {
		return nil
	}

	var args []string
	args = append(args, profile.APIServerFlags()...)
	args = append(args, profile.KubeletFlags()...)
	return args
}

func (k *K0s) KubeconfigPath() string {
	return "/var/lib/k0s/pki/admin.conf"
}

func (k *K0s) TokenCommand() string {
	return "k0s token create --role=controller"
}
