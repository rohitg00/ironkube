package kubeadm

import (
	"fmt"
	"strings"

	"github.com/rohitg00/ironkube/pkg/config"
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
	return nil
}

func (k *Kubeadm) ServerInstallScript(node config.Node, cfg *config.ClusterConfig, token string, isInit bool, extraFlags []string) string {
	ver := strings.TrimPrefix(cfg.Spec.Version, "v")
	shortVer := ver
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) >= 2 {
		shortVer = parts[0] + "." + parts[1]
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("apt-get update && apt-get install -y kubelet=%s-* kubeadm=%s-* kubectl=%s-*", ver, ver, ver))
	sb.WriteString(fmt.Sprintf(" && apt-mark hold kubelet kubeadm kubectl"))

	if isInit {
		sb.WriteString(fmt.Sprintf(" && kubeadm init --kubernetes-version=%s", shortVer))

		if cfg.Spec.Networking.PodCIDR != "" {
			sb.WriteString(fmt.Sprintf(" --pod-network-cidr=%s", cfg.Spec.Networking.PodCIDR))
		}

		if cfg.Spec.Networking.SvcCIDR != "" {
			sb.WriteString(fmt.Sprintf(" --service-cidr=%s", cfg.Spec.Networking.SvcCIDR))
		}

		if cfg.Spec.ControlPlane.Replicas > 1 {
			sb.WriteString(fmt.Sprintf(" --control-plane-endpoint=%s:6443", node.Host))
			sb.WriteString(" --upload-certs")
		}
	} else {
		firstNode := cfg.Spec.ControlPlane.Nodes[0]
		sb.WriteString(fmt.Sprintf(" && kubeadm join %s:6443 --token %s --control-plane", firstNode.Host, token))
	}

	for _, f := range extraFlags {
		sb.WriteString(" ")
		sb.WriteString(shellQuote(f))
	}

	return sb.String()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func (k *Kubeadm) AgentInstallScript(node config.Node, cfg *config.ClusterConfig, serverURL string, token string) string {
	ver := strings.TrimPrefix(cfg.Spec.Version, "v")

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("apt-get update && apt-get install -y kubelet=%s-* kubeadm=%s-*", ver, ver))
	sb.WriteString(" && apt-mark hold kubelet kubeadm")
	sb.WriteString(fmt.Sprintf(" && kubeadm join %s --token %s", serverURL, token))

	return sb.String()
}

func (k *Kubeadm) GetKubeconfigCmd() string {
	return "cat /etc/kubernetes/admin.conf"
}

func (k *Kubeadm) KubeconfigPath() string {
	return "/etc/kubernetes/admin.conf"
}

func (k *Kubeadm) UninstallCmd(role string) string {
	return "kubeadm reset -f && apt-get purge -y kubelet kubeadm kubectl"
}

func (k *Kubeadm) UpgradeCmd(version string) string {
	ver := strings.TrimPrefix(version, "v")
	return fmt.Sprintf("apt-get install -y kubeadm=%s-* && kubeadm upgrade apply v%s", ver, ver)
}
