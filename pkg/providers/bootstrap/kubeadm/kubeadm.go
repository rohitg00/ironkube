package kubeadm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type Kubeadm struct{}

func New() *Kubeadm {
	return &Kubeadm{}
}

func (k *Kubeadm) Name() string {
	return "kubeadm"
}

func (k *Kubeadm) ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	if !strings.HasPrefix(version, "v") {
		return fmt.Errorf("version must start with 'v': %s", version)
	}
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	if len(parts) != 3 {
		return fmt.Errorf("version must be valid semver (vMAJOR.MINOR.PATCH): %s", version)
	}
	return nil
}

func (k *Kubeadm) SupportedVersions() []string {
	return []string{
		"v1.31.2",
		"v1.31.1",
		"v1.30.6",
		"v1.30.5",
		"v1.29.10",
		"v1.29.9",
		"v1.28.14",
		"v1.28.13",
	}
}

func (k *Kubeadm) stripVersion(version string) string {
	return strings.TrimPrefix(version, "v")
}

func (k *Kubeadm) InitControlPlane(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, token string) (*provider.BootstrapData, error) {
	ver := k.stripVersion(cluster.Version)

	installScript := fmt.Sprintf(`apt-get update && apt-get install -y apt-transport-https ca-certificates curl
curl -fsSL https://pkgs.k8s.io/core:/stable:/v%s/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v%s/deb/ /" > /etc/apt/sources.list.d/kubernetes.list
apt-get update && apt-get install -y kubelet=%s-* kubeadm=%s-* kubectl=%s-*
apt-mark hold kubelet kubeadm kubectl`,
		ver[:strings.LastIndex(ver, ".")],
		ver[:strings.LastIndex(ver, ".")],
		ver, ver, ver,
	)

	var initFlags []string
	if cluster.PodCIDR != "" {
		initFlags = append(initFlags, fmt.Sprintf("--pod-network-cidr=%s", cluster.PodCIDR))
	}
	if cluster.ServiceCIDR != "" {
		initFlags = append(initFlags, fmt.Sprintf("--service-cidr=%s", cluster.ServiceCIDR))
	}
	if cluster.HAEnabled {
		endpoint := cluster.FirstCPHost
		if endpoint == "" {
			endpoint = machine.Spec.Host
		}
		initFlags = append(initFlags, fmt.Sprintf("--control-plane-endpoint=%s:6443", endpoint))
		initFlags = append(initFlags, "--upload-certs")
	}

	initCmd := "kubeadm init"
	if len(initFlags) > 0 {
		initCmd += " " + strings.Join(initFlags, " ")
	}

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "install-kubeadm-packages",
				Content: installScript,
				RunAs:   "root",
			},
			{
				Name:    "kubeadm-init",
				Content: initCmd,
				RunAs:   "root",
			},
		},
		WaitCheck: &provider.WaitCondition{
			Command:  "kubectl get nodes --kubeconfig=/etc/kubernetes/admin.conf",
			Expected: "Ready",
			Interval: 10 * time.Second,
			Timeout:  3 * time.Minute,
		},
	}, nil
}

func (k *Kubeadm) JoinControlPlane(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, joinEndpoint, token string) (*provider.BootstrapData, error) {
	ver := k.stripVersion(cluster.Version)

	installScript := fmt.Sprintf(`apt-get update && apt-get install -y apt-transport-https ca-certificates curl
curl -fsSL https://pkgs.k8s.io/core:/stable:/v%s/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v%s/deb/ /" > /etc/apt/sources.list.d/kubernetes.list
apt-get update && apt-get install -y kubelet=%s-* kubeadm=%s-* kubectl=%s-*
apt-mark hold kubelet kubeadm kubectl`,
		ver[:strings.LastIndex(ver, ".")],
		ver[:strings.LastIndex(ver, ".")],
		ver, ver, ver,
	)

	joinCmd := fmt.Sprintf("kubeadm join %s --token %s --control-plane --discovery-token-unsafe-skip-ca-verification", joinEndpoint, token)

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "install-kubeadm-packages",
				Content: installScript,
				RunAs:   "root",
			},
			{
				Name:    "kubeadm-join-controlplane",
				Content: joinCmd,
				RunAs:   "root",
			},
		},
		WaitCheck: &provider.WaitCondition{
			Command:  "kubectl get nodes --kubeconfig=/etc/kubernetes/admin.conf",
			Expected: "Ready",
			Interval: 10 * time.Second,
			Timeout:  3 * time.Minute,
		},
	}, nil
}

func (k *Kubeadm) JoinWorker(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, joinEndpoint, token string) (*provider.BootstrapData, error) {
	ver := k.stripVersion(cluster.Version)

	installScript := fmt.Sprintf(`apt-get update && apt-get install -y apt-transport-https ca-certificates curl
curl -fsSL https://pkgs.k8s.io/core:/stable:/v%s/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v%s/deb/ /" > /etc/apt/sources.list.d/kubernetes.list
apt-get update && apt-get install -y kubelet=%s-* kubeadm=%s-* kubectl=%s-*
apt-mark hold kubelet kubeadm kubectl`,
		ver[:strings.LastIndex(ver, ".")],
		ver[:strings.LastIndex(ver, ".")],
		ver, ver, ver,
	)

	joinCmd := fmt.Sprintf("kubeadm join %s --token %s --discovery-token-unsafe-skip-ca-verification", joinEndpoint, token)

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "install-kubeadm-packages",
				Content: installScript,
				RunAs:   "root",
			},
			{
				Name:    "kubeadm-join-worker",
				Content: joinCmd,
				RunAs:   "root",
			},
		},
		WaitCheck: &provider.WaitCondition{
			Command:  "kubectl get nodes --kubeconfig=/etc/kubernetes/admin.conf",
			Expected: "Ready",
			Interval: 10 * time.Second,
			Timeout:  3 * time.Minute,
		},
	}, nil
}

func (k *Kubeadm) Uninstall(ctx context.Context, machine provider.Machine, role string) (*provider.BootstrapData, error) {
	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "kubeadm-reset",
				Content: "kubeadm reset -f",
				RunAs:   "root",
			},
			{
				Name:    "purge-packages",
				Content: "apt-get purge -y kubelet kubeadm kubectl && apt-get autoremove -y",
				RunAs:   "root",
			},
		},
	}, nil
}

func (k *Kubeadm) Upgrade(ctx context.Context, machine provider.Machine, toVersion string) (*provider.BootstrapData, error) {
	ver := k.stripVersion(toVersion)

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "upgrade-kubeadm",
				Content: fmt.Sprintf("apt-get update && apt-get install -y kubeadm=%s-*", ver),
				RunAs:   "root",
			},
			{
				Name:    "kubeadm-upgrade-apply",
				Content: fmt.Sprintf("kubeadm upgrade apply v%s -y", ver),
				RunAs:   "root",
			},
			{
				Name:    "upgrade-kubelet-kubectl",
				Content: fmt.Sprintf("apt-get install -y kubelet=%s-* kubectl=%s-*", ver, ver),
				RunAs:   "root",
			},
		},
	}, nil
}

func (k *Kubeadm) SecurityArgs(profile provider.SecurityProfile) []string {
	if profile == nil {
		return nil
	}
	if profile.Name() == "default" {
		return nil
	}

	var args []string
	args = append(args, profile.APIServerFlags()...)
	args = append(args, profile.KubeletFlags()...)
	args = append(args, profile.EtcdFlags()...)
	return args
}

func (k *Kubeadm) KubeconfigPath() string {
	return "/etc/kubernetes/admin.conf"
}

func (k *Kubeadm) TokenCommand() string {
	return "kubeadm token create --print-join-command"
}
