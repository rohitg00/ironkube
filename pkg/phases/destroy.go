package phases

import (
	"context"
	"fmt"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/distro"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/ssh"
)

type DestroyPhase struct {
	Config *config.ClusterConfig
}

func (p *DestroyPhase) Name() string { return "destroy" }

func (p *DestroyPhase) Run(ctx context.Context, state *engine.State) error {
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

	for _, pool := range p.Config.Spec.Workers {
		for _, node := range pool.Nodes {
			exec.RunOnHost(node.Host, plugin.UninstallCmd("agent"))
		}
	}

	for i := len(p.Config.Spec.ControlPlane.Nodes) - 1; i >= 0; i-- {
		node := p.Config.Spec.ControlPlane.Nodes[i]
		out, err := exec.RunOnHost(node.Host, plugin.UninstallCmd("server"))
		if err != nil {
			return fmt.Errorf("failed to uninstall on %s: %w\nOutput: %s", node.Host, err, out)
		}
	}

	return nil
}
