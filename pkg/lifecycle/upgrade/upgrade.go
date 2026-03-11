package upgrade

import (
	"context"
	"fmt"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type NodeDrainer interface {
	DrainNode(ctx context.Context, kubeconfig []byte, nodeName string, opts provider.DrainOptions) error
	UncordonNode(ctx context.Context, kubeconfig []byte, nodeName string) error
	NodeReady(ctx context.Context, kubeconfig []byte, nodeName string) (bool, error)
}

type NodeUpgrader interface {
	Upgrade(ctx context.Context, machine provider.Machine, toVersion string) (*provider.BootstrapData, error)
}

type ScriptRunner interface {
	RunScripts(ctx context.Context, machine provider.Machine, data *provider.BootstrapData) error
}

type UpgradeOpts struct {
	MaxUnavailable int
	DrainOpts      provider.DrainOptions
}

type NodeUpgradeStatus struct {
	Machine     provider.Machine
	FromVersion string
	ToVersion   string
	Success     bool
	Error       string
}

type UpgradeResult struct {
	Nodes      []NodeUpgradeStatus
	Completed  int
	Failed     int
	TotalNodes int
}

type Manager struct {
	drainer    NodeDrainer
	upgrader   NodeUpgrader
	runner     ScriptRunner
	kubeconfig []byte
}

func New(drainer NodeDrainer, upgrader NodeUpgrader, runner ScriptRunner, kubeconfig []byte) *Manager {
	return &Manager{
		drainer:    drainer,
		upgrader:   upgrader,
		runner:     runner,
		kubeconfig: kubeconfig,
	}
}

func (m *Manager) RollingUpgrade(ctx context.Context, machines []provider.Machine, toVersion string, opts UpgradeOpts) (*UpgradeResult, error) {
	result := &UpgradeResult{
		TotalNodes: len(machines),
	}

	if len(machines) == 0 {
		return result, nil
	}

	var controlPlane []provider.Machine
	var workers []provider.Machine
	for _, machine := range machines {
		if machine.Spec.Role == "control-plane" {
			controlPlane = append(controlPlane, machine)
		} else {
			workers = append(workers, machine)
		}
	}

	ordered := append(controlPlane, workers...)

	for _, machine := range ordered {
		if err := ctx.Err(); err != nil {
			return result, fmt.Errorf("upgrade cancelled: %w", err)
		}

		nodeStatus := NodeUpgradeStatus{
			Machine:   machine,
			ToVersion: toVersion,
		}

		if err := m.upgradeNode(ctx, machine, toVersion, opts); err != nil {
			nodeStatus.Success = false
			nodeStatus.Error = err.Error()
			result.Nodes = append(result.Nodes, nodeStatus)
			result.Failed++
			return result, fmt.Errorf("upgrading node %s: %w", machine.ID, err)
		}

		nodeStatus.Success = true
		result.Nodes = append(result.Nodes, nodeStatus)
		result.Completed++
	}

	return result, nil
}

func (m *Manager) upgradeNode(ctx context.Context, machine provider.Machine, toVersion string, opts UpgradeOpts) error {
	nodeName := machine.ID

	if err := m.drainer.DrainNode(ctx, m.kubeconfig, nodeName, opts.DrainOpts); err != nil {
		return fmt.Errorf("draining node %s: %w", nodeName, err)
	}

	data, err := m.upgrader.Upgrade(ctx, machine, toVersion)
	if err != nil {
		return fmt.Errorf("generating upgrade scripts for %s: %w", nodeName, err)
	}

	if err := m.runner.RunScripts(ctx, machine, data); err != nil {
		return fmt.Errorf("running upgrade scripts on %s: %w", nodeName, err)
	}

	if err := m.drainer.UncordonNode(ctx, m.kubeconfig, nodeName); err != nil {
		return fmt.Errorf("uncordoning node %s: %w", nodeName, err)
	}

	ready, err := m.drainer.NodeReady(ctx, m.kubeconfig, nodeName)
	if err != nil {
		return fmt.Errorf("checking readiness of %s: %w", nodeName, err)
	}
	if !ready {
		return fmt.Errorf("node %s not ready after upgrade", nodeName)
	}

	return nil
}
