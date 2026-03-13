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

func (k *Kubeadm) ServerInstallScript(node config.Node, cfg *config.ClusterConfig, token string, isInit bool, secFlags config.SecurityFlags) string {
	ver := strings.TrimPrefix(cfg.Spec.Version, "v")
	shortVer := ver
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) >= 2 {
		shortVer = parts[0] + "." + parts[1]
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("apt-get update && apt-get install -y kubelet=%s-* kubeadm=%s-* kubectl=%s-*", ver, ver, ver))
	sb.WriteString(" && apt-mark hold kubelet kubeadm kubectl")

	if isInit {
		if secFlags.IsEmpty() {
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
			sb.WriteString(" && ")
			sb.WriteString(generateInitConfig(node, cfg, shortVer, secFlags))
			sb.WriteString(" && kubeadm init --config /tmp/kubeadm-config.yaml")
			if cfg.Spec.ControlPlane.Replicas > 1 {
				sb.WriteString(" --upload-certs")
			}
		}
	} else {
		firstNode := cfg.Spec.ControlPlane.Nodes[0]
		if secFlags.IsEmpty() {
			sb.WriteString(fmt.Sprintf(" && kubeadm join %s:6443 --token %s --control-plane", firstNode.Host, token))
		} else {
			sb.WriteString(" && ")
			sb.WriteString(generateJoinConfig(node, firstNode, token, secFlags))
			sb.WriteString(" && kubeadm join --config /tmp/kubeadm-join-config.yaml")
		}
	}

	return sb.String()
}

func generateInitConfig(node config.Node, cfg *config.ClusterConfig, shortVer string, secFlags config.SecurityFlags) string {
	var sb strings.Builder
	sb.WriteString("cat > /tmp/kubeadm-config.yaml <<'KUBEADM_EOF'\napiVersion: kubeadm.k8s.io/v1beta3\nkind: InitConfiguration\n")

	if !secFlags.IsEmpty() && len(secFlags.Kubelet) > 0 {
		sb.WriteString("nodeRegistration:\n  kubeletExtraArgs:\n")
		for _, f := range secFlags.Kubelet {
			k, v := parseFlag(f)
			sb.WriteString(fmt.Sprintf("    %s: \"%s\"\n", k, v))
		}
	}

	sb.WriteString("---\napiVersion: kubeadm.k8s.io/v1beta3\nkind: ClusterConfiguration\n")
	sb.WriteString(fmt.Sprintf("kubernetesVersion: \"%s\"\n", shortVer))

	if cfg.Spec.Networking.PodCIDR != "" || cfg.Spec.Networking.SvcCIDR != "" {
		sb.WriteString("networking:\n")
		if cfg.Spec.Networking.PodCIDR != "" {
			sb.WriteString(fmt.Sprintf("  podSubnet: \"%s\"\n", cfg.Spec.Networking.PodCIDR))
		}
		if cfg.Spec.Networking.SvcCIDR != "" {
			sb.WriteString(fmt.Sprintf("  serviceSubnet: \"%s\"\n", cfg.Spec.Networking.SvcCIDR))
		}
	}

	if cfg.Spec.ControlPlane.Replicas > 1 {
		sb.WriteString(fmt.Sprintf("controlPlaneEndpoint: \"%s:6443\"\n", node.Host))
	}

	if len(secFlags.APIServer) > 0 {
		sb.WriteString("apiServer:\n  extraArgs:\n")
		for _, f := range secFlags.APIServer {
			k, v := parseFlag(f)
			sb.WriteString(fmt.Sprintf("    %s: \"%s\"\n", k, v))
		}
	}

	if len(secFlags.Etcd) > 0 {
		sb.WriteString("etcd:\n  local:\n    extraArgs:\n")
		for _, f := range secFlags.Etcd {
			k, v := parseFlag(f)
			sb.WriteString(fmt.Sprintf("    %s: \"%s\"\n", k, v))
		}
	}

	sb.WriteString("KUBEADM_EOF")
	return sb.String()
}

func generateJoinConfig(node, firstNode config.Node, token string, secFlags config.SecurityFlags) string {
	var sb strings.Builder
	sb.WriteString("cat > /tmp/kubeadm-join-config.yaml <<'KUBEADM_EOF'\napiVersion: kubeadm.k8s.io/v1beta3\nkind: JoinConfiguration\n")
	sb.WriteString("discovery:\n  bootstrapToken:\n")
	sb.WriteString(fmt.Sprintf("    apiServerEndpoint: \"%s:6443\"\n", firstNode.Host))
	sb.WriteString(fmt.Sprintf("    token: \"%s\"\n", token))
	sb.WriteString("    unsafeSkipCAVerification: true\n")
	sb.WriteString("controlPlane: {}\n")

	if len(secFlags.Kubelet) > 0 {
		sb.WriteString("nodeRegistration:\n  kubeletExtraArgs:\n")
		for _, f := range secFlags.Kubelet {
			k, v := parseFlag(f)
			sb.WriteString(fmt.Sprintf("    %s: \"%s\"\n", k, v))
		}
	}

	sb.WriteString("KUBEADM_EOF")
	return sb.String()
}

func parseFlag(flag string) (string, string) {
	trimmed := strings.TrimPrefix(flag, "--")
	if idx := strings.Index(trimmed, "="); idx >= 0 {
		return trimmed[:idx], trimmed[idx+1:]
	}
	return trimmed, "true"
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
