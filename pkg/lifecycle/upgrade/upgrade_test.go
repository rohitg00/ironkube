package upgrade

import (
	"context"
	"fmt"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDrainer struct {
	drainErr    map[string]error
	uncordonErr map[string]error
	readyResult map[string]bool
	readyErr    map[string]error
	drainCalls  []string
	uncordCalls []string
	readyCalls  []string
}

func newMockDrainer() *mockDrainer {
	return &mockDrainer{
		drainErr:    make(map[string]error),
		uncordonErr: make(map[string]error),
		readyResult: make(map[string]bool),
		readyErr:    make(map[string]error),
	}
}

func (m *mockDrainer) DrainNode(_ context.Context, _ []byte, nodeName string, _ provider.DrainOptions) error {
	m.drainCalls = append(m.drainCalls, nodeName)
	if err, ok := m.drainErr[nodeName]; ok {
		return err
	}
	return nil
}

func (m *mockDrainer) UncordonNode(_ context.Context, _ []byte, nodeName string) error {
	m.uncordCalls = append(m.uncordCalls, nodeName)
	if err, ok := m.uncordonErr[nodeName]; ok {
		return err
	}
	return nil
}

func (m *mockDrainer) NodeReady(_ context.Context, _ []byte, nodeName string) (bool, error) {
	m.readyCalls = append(m.readyCalls, nodeName)
	if err, ok := m.readyErr[nodeName]; ok {
		return false, err
	}
	if ready, ok := m.readyResult[nodeName]; ok {
		return ready, nil
	}
	return true, nil
}

type mockUpgrader struct {
	results map[string]*provider.BootstrapData
	errors  map[string]error
	calls   []string
}

func newMockUpgrader() *mockUpgrader {
	return &mockUpgrader{
		results: make(map[string]*provider.BootstrapData),
		errors:  make(map[string]error),
	}
}

func (m *mockUpgrader) Upgrade(_ context.Context, machine provider.Machine, _ string) (*provider.BootstrapData, error) {
	m.calls = append(m.calls, machine.ID)
	if err, ok := m.errors[machine.ID]; ok {
		return nil, err
	}
	if data, ok := m.results[machine.ID]; ok {
		return data, nil
	}
	return &provider.BootstrapData{
		Scripts: []provider.Script{{Name: "upgrade", Content: "apt upgrade"}},
	}, nil
}

type mockRunner struct {
	errors map[string]error
	calls  []string
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		errors: make(map[string]error),
	}
}

func (m *mockRunner) RunScripts(_ context.Context, machine provider.Machine, _ *provider.BootstrapData) error {
	m.calls = append(m.calls, machine.ID)
	if err, ok := m.errors[machine.ID]; ok {
		return err
	}
	return nil
}

func cpMachine(id string) provider.Machine {
	return provider.Machine{
		ID: id,
		Spec: provider.MachineSpec{
			Role: "control-plane",
			Host: "10.0.0.1",
			User: "root",
			Port: 22,
		},
	}
}

func workerMachine(id string) provider.Machine {
	return provider.Machine{
		ID: id,
		Spec: provider.MachineSpec{
			Role: "worker",
			Host: "10.0.0.10",
			User: "root",
			Port: 22,
		},
	}
}

func TestSingleNodeUpgradeSucceeds(t *testing.T) {
	drainer := newMockDrainer()
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{cpMachine("cp-1")}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Completed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 1, result.TotalNodes)
	assert.True(t, result.Nodes[0].Success)

	assert.Equal(t, []string{"cp-1"}, drainer.drainCalls)
	assert.Equal(t, []string{"cp-1"}, upgrader.calls)
	assert.Equal(t, []string{"cp-1"}, runner.calls)
	assert.Equal(t, []string{"cp-1"}, drainer.uncordCalls)
	assert.Equal(t, []string{"cp-1"}, drainer.readyCalls)
}

func TestControlPlaneUpgradedBeforeWorkers(t *testing.T) {
	drainer := newMockDrainer()
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{
		workerMachine("w-1"),
		cpMachine("cp-1"),
		workerMachine("w-2"),
		cpMachine("cp-2"),
	}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	require.NoError(t, err)

	assert.Equal(t, 4, result.Completed)
	assert.Equal(t, []string{"cp-1", "cp-2", "w-1", "w-2"}, drainer.drainCalls)
}

func TestDrainFailureStopsUpgrade(t *testing.T) {
	drainer := newMockDrainer()
	drainer.drainErr["cp-1"] = fmt.Errorf("eviction timeout")
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{cpMachine("cp-1"), workerMachine("w-1")}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "eviction timeout")

	assert.Equal(t, 0, result.Completed)
	assert.Equal(t, 1, result.Failed)
	assert.Len(t, upgrader.calls, 0)
	assert.Len(t, drainer.uncordCalls, 0)
}

func TestUpgradeScriptFailureStopsUpgrade(t *testing.T) {
	drainer := newMockDrainer()
	upgrader := newMockUpgrader()
	upgrader.errors["cp-1"] = fmt.Errorf("version not supported")
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{cpMachine("cp-1")}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version not supported")

	assert.Equal(t, 1, result.Failed)
	assert.Len(t, drainer.uncordCalls, 0)
}

func TestScriptRunnerFailureReturnsError(t *testing.T) {
	drainer := newMockDrainer()
	upgrader := newMockUpgrader()
	runner := newMockRunner()
	runner.errors["cp-1"] = fmt.Errorf("ssh connection lost")

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{cpMachine("cp-1")}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ssh connection lost")
	assert.Equal(t, 1, result.Failed)
	assert.Len(t, drainer.uncordCalls, 0)
}

func TestNodeNotReadyAfterUncordonTriggersError(t *testing.T) {
	drainer := newMockDrainer()
	drainer.readyResult["cp-1"] = false
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{cpMachine("cp-1")}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not ready after upgrade")
	assert.Equal(t, 1, result.Failed)
}

func TestNodeReadyCheckErrorReturnsError(t *testing.T) {
	drainer := newMockDrainer()
	drainer.readyErr["cp-1"] = fmt.Errorf("api server unavailable")
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{cpMachine("cp-1")}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api server unavailable")
	assert.Equal(t, 1, result.Failed)
}

func TestMultipleNodesUpgradedInOrder(t *testing.T) {
	drainer := newMockDrainer()
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{
		cpMachine("cp-1"),
		cpMachine("cp-2"),
		workerMachine("w-1"),
	}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	require.NoError(t, err)

	assert.Equal(t, 3, result.Completed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, []string{"cp-1", "cp-2", "w-1"}, upgrader.calls)
}

func TestPartialFailureReturnsResultWithCounts(t *testing.T) {
	drainer := newMockDrainer()
	upgrader := newMockUpgrader()
	upgrader.errors["cp-2"] = fmt.Errorf("disk full")
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{
		cpMachine("cp-1"),
		cpMachine("cp-2"),
		workerMachine("w-1"),
	}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	assert.Error(t, err)

	assert.Equal(t, 1, result.Completed)
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, 3, result.TotalNodes)
	assert.Len(t, result.Nodes, 2)
	assert.True(t, result.Nodes[0].Success)
	assert.False(t, result.Nodes[1].Success)
	assert.Contains(t, result.Nodes[1].Error, "disk full")
}

func TestEmptyMachineListReturnsEmptyResult(t *testing.T) {
	drainer := newMockDrainer()
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))

	result, err := mgr.RollingUpgrade(context.Background(), nil, "v1.30.0", UpgradeOpts{})
	require.NoError(t, err)

	assert.Equal(t, 0, result.TotalNodes)
	assert.Equal(t, 0, result.Completed)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, result.Nodes)
}

func TestContextCancellationDuringUpgrade(t *testing.T) {
	drainer := newMockDrainer()
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{cpMachine("cp-1")}

	_, err := mgr.RollingUpgrade(ctx, machines, "v1.30.0", UpgradeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upgrade cancelled")
}

func TestUpgradeResultNodeStatus(t *testing.T) {
	drainer := newMockDrainer()
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{cpMachine("cp-1")}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	require.NoError(t, err)

	assert.Equal(t, "v1.30.0", result.Nodes[0].ToVersion)
	assert.Equal(t, "cp-1", result.Nodes[0].Machine.ID)
	assert.Empty(t, result.Nodes[0].Error)
}

func TestDrainOptsPassedThrough(t *testing.T) {
	var capturedOpts provider.DrainOptions
	drainer := &captureOptsDrainer{captured: &capturedOpts}
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{cpMachine("cp-1")}

	opts := UpgradeOpts{
		DrainOpts: provider.DrainOptions{
			GracePeriod:      60,
			IgnoreDaemonSets: true,
			Force:            true,
		},
	}

	_, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", opts)
	require.NoError(t, err)

	assert.Equal(t, 60, capturedOpts.GracePeriod)
	assert.True(t, capturedOpts.IgnoreDaemonSets)
	assert.True(t, capturedOpts.Force)
}

type captureOptsDrainer struct {
	captured *provider.DrainOptions
}

func (c *captureOptsDrainer) DrainNode(_ context.Context, _ []byte, _ string, opts provider.DrainOptions) error {
	*c.captured = opts
	return nil
}

func (c *captureOptsDrainer) UncordonNode(_ context.Context, _ []byte, _ string) error {
	return nil
}

func (c *captureOptsDrainer) NodeReady(_ context.Context, _ []byte, _ string) (bool, error) {
	return true, nil
}

func TestUncordonFailureReturnsError(t *testing.T) {
	drainer := newMockDrainer()
	drainer.uncordonErr["cp-1"] = fmt.Errorf("node not found")
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{cpMachine("cp-1")}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node not found")
	assert.Equal(t, 1, result.Failed)
}

func TestAllWorkersUpgraded(t *testing.T) {
	drainer := newMockDrainer()
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{
		workerMachine("w-1"),
		workerMachine("w-2"),
		workerMachine("w-3"),
	}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	require.NoError(t, err)

	assert.Equal(t, 3, result.Completed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, []string{"w-1", "w-2", "w-3"}, drainer.drainCalls)
}

func TestUpgradeStopsAtFirstFailure(t *testing.T) {
	drainer := newMockDrainer()
	drainer.drainErr["w-2"] = fmt.Errorf("pdb violation")
	upgrader := newMockUpgrader()
	runner := newMockRunner()

	mgr := New(drainer, upgrader, runner, []byte("kubeconfig"))
	machines := []provider.Machine{
		workerMachine("w-1"),
		workerMachine("w-2"),
		workerMachine("w-3"),
	}

	result, err := mgr.RollingUpgrade(context.Background(), machines, "v1.30.0", UpgradeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pdb violation")

	assert.Equal(t, 1, result.Completed)
	assert.Equal(t, 1, result.Failed)
	assert.Len(t, drainer.drainCalls, 2)
	assert.Len(t, upgrader.calls, 1)
}
