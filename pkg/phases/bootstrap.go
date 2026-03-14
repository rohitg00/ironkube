package phases

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/distro"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/security"
	"github.com/rohitg00/ironkube/pkg/ssh"
)

type BootstrapPhase struct {
	Config *config.ClusterConfig
}

func (p *BootstrapPhase) Name() string { return "bootstrap" }

func (p *BootstrapPhase) Run(ctx context.Context, state *engine.State) error {
	v, ok := state.Get("executor")
	if !ok {
		return fmt.Errorf("executor not found in state")
	}
	exec := v.(*ssh.Executor)

	v, ok = state.Get("distro_plugin")
	if !ok {
		return fmt.Errorf("distro_plugin not found in state")
	}
	plugin := v.(distro.Plugin)

	token := generateToken()
	state.Set("cluster_token", token)

	secFlags, err := security.FlagsForProfile(p.Config.Spec.Security.Profile)
	if err != nil {
		return fmt.Errorf("resolving security flags: %w", err)
	}

	firstNode := p.Config.Spec.ControlPlane.Nodes[0]
	installCmd := plugin.ServerInstallScript(firstNode, p.Config, token, true, secFlags, "")

	out, err := exec.RunOnHost(firstNode.Host, installCmd)
	if err != nil {
		p.Cleanup(ctx, state)
		return fmt.Errorf("failed to bootstrap first control plane node %s: %w\nOutput: %s", firstNode.Host, err, out)
	}

	certKey := parseCertificateKey(out)
	if certKey != "" {
		state.Set("certificate_key", certKey)
	}

	if err := waitForNode(ctx, exec, firstNode.Host, plugin); err != nil {
		p.Cleanup(ctx, state)
		return fmt.Errorf("first control plane node did not become ready: %w", err)
	}

	for i := 1; i < len(p.Config.Spec.ControlPlane.Nodes); i++ {
		node := p.Config.Spec.ControlPlane.Nodes[i]
		joinCmd := plugin.ServerInstallScript(node, p.Config, token, false, secFlags, certKey)

		out, err := exec.RunOnHost(node.Host, joinCmd)
		if err != nil {
			p.Cleanup(ctx, state)
			return fmt.Errorf("failed to join control plane node %s: %w\nOutput: %s", node.Host, err, out)
		}
	}

	for _, pool := range p.Config.Spec.Workers {
		for _, node := range pool.Nodes {
			serverURL := fmt.Sprintf("https://%s:6443", firstNode.Host)
			agentCmd := plugin.AgentInstallScript(node, p.Config, serverURL, token)

			out, err := exec.RunOnHost(node.Host, agentCmd)
			if err != nil {
				p.Cleanup(ctx, state)
				return fmt.Errorf("failed to join worker %s: %w\nOutput: %s", node.Host, err, out)
			}
		}
	}

	expectedNodes := len(p.Config.Spec.ControlPlane.Nodes)
	for _, pool := range p.Config.Spec.Workers {
		expectedNodes += len(pool.Nodes)
	}

	if expectedNodes > 1 {
		if err := waitForAllNodes(ctx, exec, firstNode.Host, plugin, expectedNodes); err != nil {
			p.Cleanup(ctx, state)
			return fmt.Errorf("not all nodes became ready: %w", err)
		}
	}

	return nil
}

func (p *BootstrapPhase) Cleanup(ctx context.Context, state *engine.State) error {
	v, ok := state.Get("executor")
	if !ok {
		return nil
	}
	exec := v.(*ssh.Executor)

	v, ok = state.Get("distro_plugin")
	if !ok {
		return nil
	}
	plugin := v.(distro.Plugin)

	for _, node := range p.Config.Spec.ControlPlane.Nodes {
		_, _ = exec.RunOnHost(node.Host, plugin.UninstallCmd("server"))
	}

	for _, pool := range p.Config.Spec.Workers {
		for _, node := range pool.Nodes {
			_, _ = exec.RunOnHost(node.Host, plugin.UninstallCmd("agent"))
		}
	}

	return nil
}

func waitForNode(ctx context.Context, exec *ssh.Executor, host string, plugin distro.Plugin) error {
	checkCmd := fmt.Sprintf("%s get nodes --no-headers 2>/dev/null", kubectlCmd(plugin))

	deadline := time.After(3 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("timed out waiting for node to be ready on %s", host)
		case <-ticker.C:
			out, err := exec.RunOnHost(host, checkCmd)
			if err != nil {
				continue
			}
			if hasReadyNode(out) {
				return nil
			}
		}
	}
}

func waitForAllNodes(ctx context.Context, exec *ssh.Executor, host string, plugin distro.Plugin, expected int) error {
	checkCmd := fmt.Sprintf("%s get nodes --no-headers 2>/dev/null", kubectlCmd(plugin))

	deadline := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("timed out waiting for %d nodes to be ready on %s", expected, host)
		case <-ticker.C:
			out, err := exec.RunOnHost(host, checkCmd)
			if err != nil {
				continue
			}
			if countReadyNodes(out) >= expected {
				return nil
			}
		}
	}
}

func hasReadyNode(output string) bool {
	return countReadyNodes(output) > 0
}

func countReadyNodes(output string) int {
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "Ready" {
			count++
		}
	}
	return count
}

func kubectlCmd(plugin distro.Plugin) string {
	switch plugin.Name() {
	case "k3s":
		return "k3s kubectl"
	default:
		return "kubectl --kubeconfig=/etc/kubernetes/admin.conf"
	}
}

func parseCertificateKey(output string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--certificate-key") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return parts[len(parts)-1]
			}
		}
	}
	return ""
}

func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
