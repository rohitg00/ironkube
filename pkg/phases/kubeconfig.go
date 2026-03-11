package phases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/distro"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/kubeconfig"
	"github.com/rohitg00/ironkube/pkg/ssh"
)

type FetchKubeconfigPhase struct {
	Config     *config.ClusterConfig
	OutputPath string
}

func (p *FetchKubeconfigPhase) Name() string { return "kubeconfig" }

func (p *FetchKubeconfigPhase) Run(ctx context.Context, state *engine.State) error {
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

	firstNode := p.Config.Spec.ControlPlane.Nodes[0]
	kubeconfigData, err := exec.RunOnHost(firstNode.Host, plugin.GetKubeconfigCmd())
	if err != nil {
		return fmt.Errorf("failed to fetch kubeconfig from %s: %w", firstNode.Host, err)
	}

	rewritten, err := kubeconfig.RewriteServer([]byte(kubeconfigData), firstNode.Host, p.Config.Metadata.Name)
	if err != nil {
		return fmt.Errorf("failed to rewrite kubeconfig: %w", err)
	}

	outputPath := p.OutputPath
	if outputPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		outputPath = filepath.Join(home, ".kube", "config")
	}

	if err := kubeconfig.Merge(outputPath, rewritten); err != nil {
		return fmt.Errorf("failed to merge kubeconfig: %w", err)
	}

	state.Set("kubeconfig_path", outputPath)
	return nil
}
