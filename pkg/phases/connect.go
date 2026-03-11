package phases

import (
	"context"
	"fmt"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/ssh"
)

type ConnectPhase struct {
	Config *config.ClusterConfig
}

func (p *ConnectPhase) Name() string { return "connect" }

func (p *ConnectPhase) Run(ctx context.Context, state *engine.State) error {
	exec := ssh.NewExecutor()
	state.Set("executor", exec)

	for _, node := range p.Config.Spec.ControlPlane.Nodes {
		cfg := ssh.Config{
			Host:    node.Host,
			Port:    node.Port,
			User:    node.User,
			KeyPath: node.KeyPath,
		}
		if err := exec.Connect(cfg); err != nil {
			return fmt.Errorf("failed to connect to %s: %w", node.Host, err)
		}
	}

	for _, pool := range p.Config.Spec.Workers {
		for _, node := range pool.Nodes {
			cfg := ssh.Config{
				Host:    node.Host,
				Port:    node.Port,
				User:    node.User,
				KeyPath: node.KeyPath,
			}
			if err := exec.Connect(cfg); err != nil {
				return fmt.Errorf("failed to connect to worker %s: %w", node.Host, err)
			}
		}
	}

	return nil
}

func (p *ConnectPhase) Cleanup(ctx context.Context, state *engine.State) error {
	v, ok := state.Get("executor")
	if !ok {
		return nil
	}
	if exec, ok := v.(*ssh.Executor); ok {
		exec.Close()
	}
	return nil
}
