package k3s

import (
	"fmt"
	"strings"

	"github.com/rohitg00/ironkube/pkg/config"
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

func (k *K3s) ServerInstallScript(node config.Node, cfg *config.ClusterConfig, token string, isInit bool, secFlags config.SecurityFlags, _ string) string {
	var sb strings.Builder

	sb.WriteString("curl -sfL https://get.k3s.io | ")
	sb.WriteString(fmt.Sprintf("INSTALL_K3S_VERSION=%s ", cfg.Spec.Version))

	if token != "" {
		sb.WriteString(fmt.Sprintf("K3S_TOKEN=%s ", token))
	}

	if !isInit && token != "" {
		firstNode := cfg.Spec.ControlPlane.Nodes[0]
		sb.WriteString(fmt.Sprintf("K3S_URL=https://%s:6443 ", firstNode.Host))
	}

	sb.WriteString("sh -s - server")

	if isInit && cfg.Spec.ControlPlane.Replicas > 1 {
		sb.WriteString(" --cluster-init")
	}

	if cfg.Spec.Networking.PodCIDR != "" {
		sb.WriteString(fmt.Sprintf(" --cluster-cidr=%s", cfg.Spec.Networking.PodCIDR))
	}

	if cfg.Spec.Networking.SvcCIDR != "" {
		sb.WriteString(fmt.Sprintf(" --service-cidr=%s", cfg.Spec.Networking.SvcCIDR))
	}

	for _, f := range secFlags.Kubelet {
		sb.WriteString(" ")
		sb.WriteString(shellQuote(toK3sFlag("kubelet-arg", f)))
	}
	for _, f := range secFlags.APIServer {
		sb.WriteString(" ")
		sb.WriteString(shellQuote(toK3sFlag("kube-apiserver-arg", f)))
	}
	for _, f := range secFlags.Etcd {
		sb.WriteString(" ")
		sb.WriteString(shellQuote(toK3sFlag("etcd-arg", f)))
	}

	return sb.String()
}

func toK3sFlag(wrapper, raw string) string {
	trimmed := strings.TrimPrefix(raw, "--")
	return "--" + wrapper + "=" + trimmed
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func (k *K3s) AgentInstallScript(node config.Node, cfg *config.ClusterConfig, serverURL string, token string) string {
	var sb strings.Builder

	sb.WriteString("curl -sfL https://get.k3s.io | ")
	sb.WriteString(fmt.Sprintf("INSTALL_K3S_VERSION=%s ", cfg.Spec.Version))
	sb.WriteString(fmt.Sprintf("K3S_URL=%s ", serverURL))
	sb.WriteString(fmt.Sprintf("K3S_TOKEN=%s ", token))
	sb.WriteString("sh -s - agent")

	return sb.String()
}

func (k *K3s) GetKubeconfigCmd() string {
	return "cat /etc/rancher/k3s/k3s.yaml"
}

func (k *K3s) KubeconfigPath() string {
	return "/etc/rancher/k3s/k3s.yaml"
}

func (k *K3s) UninstallCmd(role string) string {
	if role == "agent" {
		return "/usr/local/bin/k3s-agent-uninstall.sh"
	}
	return "/usr/local/bin/k3s-uninstall.sh"
}

func (k *K3s) UpgradeCmd(version string) string {
	return fmt.Sprintf("curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=%s sh -", version)
}
