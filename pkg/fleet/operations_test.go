package fleet

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeClusterConfigFile(t *testing.T, dir, name string) string {
	t.Helper()
	content := fmt.Sprintf(`apiVersion: ironkube.dev/v1alpha2
kind: ClusterConfig
metadata:
  name: %s
spec:
  distro: k3s
  version: "v1.28.0"
  infrastructure:
    provider: ssh
  controlPlane:
    replicas: 1
    nodes:
    - host: "10.0.0.1"
      user: root
      port: 22
  networking:
    podCIDR: "10.244.0.0/16"
    serviceCIDR: "10.96.0.0/12"
  security:
    profile: minimal
`, name)
	path := filepath.Join(dir, name+".yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestDiffAllReturnsResultsForEachCluster(t *testing.T) {
	dir := t.TempDir()
	writeClusterConfigFile(t, dir, "cluster-a")
	writeClusterConfigFile(t, dir, "cluster-b")

	backend := newMockBackend()
	backend.states["cluster-a"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "cluster-a", Version: "v1.28.0"},
		Machines: map[string]state.MachineState{
			"10.0.0.1": {ID: "10.0.0.1", Host: "10.0.0.1", Role: "control-plane", Status: "ready", K8sVersion: "v1.28.0"},
		},
	}

	inv := NewInventory(backend)
	op := NewOperator(inv, backend)

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: filepath.Join(dir, "cluster-a.yaml")},
				{Path: filepath.Join(dir, "cluster-b.yaml")},
			},
		},
	}

	results, err := op.DiffAll(context.Background(), fleetCfg)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	resultA := results["cluster-a"]
	assert.True(t, resultA.InSync)

	resultB := results["cluster-b"]
	assert.False(t, resultB.InSync)
	assert.Greater(t, len(resultB.Actions), 0)
}

func TestDiffAllHandlesMissingClusterConfigs(t *testing.T) {
	backend := newMockBackend()
	inv := NewInventory(backend)
	op := NewOperator(inv, backend)

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "/nonexistent/missing-cluster.yaml"},
			},
		},
	}

	results, err := op.DiffAll(context.Background(), fleetCfg)
	require.NoError(t, err)

	result := results["missing-cluster"]
	assert.Len(t, result.Actions, 1)
	assert.Equal(t, engine.ActionType("error"), result.Actions[0].Type)
	assert.Contains(t, result.Actions[0].Description, "failed to load config")
}

func TestDiffAllEmptyFleet(t *testing.T) {
	backend := newMockBackend()
	inv := NewInventory(backend)
	op := NewOperator(inv, backend)

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "empty-fleet"},
		Spec:     v1alpha2.FleetSpec{},
	}

	results, err := op.DiffAll(context.Background(), fleetCfg)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestStatusAllReturnsSummaries(t *testing.T) {
	backend := newMockBackend()
	backend.states["cluster-a"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "cluster-a", Version: "v1.28.0"},
		Machines: map[string]state.MachineState{
			"n1": {ID: "n1", Status: "ready"},
		},
	}
	backend.states["cluster-b"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "cluster-b", Version: "v1.27.0"},
	}

	inv := NewInventory(backend)
	op := NewOperator(inv, backend)

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "cluster-a.yaml"},
				{Path: "cluster-b.yaml"},
			},
		},
	}

	summaries, err := op.StatusAll(context.Background(), fleetCfg)
	require.NoError(t, err)
	assert.Len(t, summaries, 2)
	assert.True(t, summaries[0].Available)
	assert.True(t, summaries[1].Available)
}

func TestStatusAllWithUnavailableClusters(t *testing.T) {
	backend := newMockBackend()
	backend.states["cluster-a"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "cluster-a"},
	}

	inv := NewInventory(backend)
	op := NewOperator(inv, backend)

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "cluster-a.yaml"},
				{Path: "cluster-missing.yaml"},
			},
		},
	}

	summaries, err := op.StatusAll(context.Background(), fleetCfg)
	require.NoError(t, err)
	assert.Len(t, summaries, 2)
	assert.True(t, summaries[0].Available)
	assert.False(t, summaries[1].Available)
}

func TestUpgradeAllDryRunShowsActions(t *testing.T) {
	backend := newMockBackend()
	backend.states["cluster-a"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "cluster-a", Version: "v1.27.0"},
		Machines: map[string]state.MachineState{
			"n1": {ID: "n1", Host: "10.0.0.1", Role: "control-plane", K8sVersion: "v1.27.0", Status: "ready"},
			"n2": {ID: "n2", Host: "10.0.0.2", Role: "worker", K8sVersion: "v1.27.0", Status: "ready"},
		},
	}

	inv := NewInventory(backend)
	op := NewOperator(inv, backend)

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "cluster-a.yaml"},
			},
		},
	}

	result, err := op.UpgradeAll(context.Background(), fleetCfg, "v1.28.0", FleetUpgradeOpts{DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
	assert.Len(t, result.Clusters, 1)
	assert.Len(t, result.Clusters[0].Actions, 2)
	for _, a := range result.Clusters[0].Actions {
		assert.Equal(t, engine.ActionUpgradeVersion, a.Type)
		assert.Equal(t, "v1.27.0", a.FromVersion)
		assert.Equal(t, "v1.28.0", a.ToVersion)
	}
}

func TestUpgradeAllWithNoClusters(t *testing.T) {
	backend := newMockBackend()
	inv := NewInventory(backend)
	op := NewOperator(inv, backend)

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "empty-fleet"},
		Spec:     v1alpha2.FleetSpec{},
	}

	result, err := op.UpgradeAll(context.Background(), fleetCfg, "v1.28.0", FleetUpgradeOpts{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
	assert.Len(t, result.Clusters, 0)
}

func TestUpgradeAllRespectsSequentialExecution(t *testing.T) {
	backend := newMockBackend()
	now := time.Now()
	backend.states["cluster-a"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "cluster-a", Version: "v1.27.0", CreatedAt: now},
		Machines: map[string]state.MachineState{
			"n1": {ID: "n1", Host: "10.0.0.1", K8sVersion: "v1.27.0", Status: "ready"},
		},
	}
	backend.states["cluster-b"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "cluster-b", Version: "v1.27.0", CreatedAt: now},
		Machines: map[string]state.MachineState{
			"n2": {ID: "n2", Host: "10.0.1.1", K8sVersion: "v1.27.0", Status: "ready"},
		},
	}

	inv := NewInventory(backend)
	op := NewOperator(inv, backend)

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "cluster-a.yaml"},
				{Path: "cluster-b.yaml"},
			},
			Policies: v1alpha2.FleetPolicies{MaxConcurrentUpgrades: 1},
		},
	}

	result, err := op.UpgradeAll(context.Background(), fleetCfg, "v1.28.0", FleetUpgradeOpts{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
	assert.Len(t, result.Clusters, 2)
	assert.Equal(t, "cluster-a", result.Clusters[0].Name)
	assert.Equal(t, "cluster-b", result.Clusters[1].Name)
}

func TestUpgradeAllPartialFailure(t *testing.T) {
	backend := newMockBackend()
	backend.states["cluster-a"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "cluster-a", Version: "v1.27.0"},
		Machines: map[string]state.MachineState{
			"n1": {ID: "n1", Host: "10.0.0.1", K8sVersion: "v1.27.0"},
		},
	}
	backend.err["cluster-b"] = fmt.Errorf("connection refused")

	inv := NewInventory(backend)
	op := NewOperator(inv, backend)

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "cluster-a.yaml"},
				{Path: "cluster-b.yaml"},
			},
		},
	}

	result, err := op.UpgradeAll(context.Background(), fleetCfg, "v1.28.0", FleetUpgradeOpts{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Succeeded)
	assert.Equal(t, 1, result.Failed)
	assert.True(t, result.Clusters[0].Success)
	assert.False(t, result.Clusters[1].Success)
	assert.Contains(t, result.Clusters[1].Error, "connection refused")
}

func TestFleetUpgradeResultCounts(t *testing.T) {
	result := &FleetUpgradeResult{
		Clusters: []ClusterUpgradeResult{
			{Name: "a", Success: true},
			{Name: "b", Success: false, Error: "fail"},
			{Name: "c", Success: true},
		},
		Succeeded: 2,
		Failed:    1,
	}

	assert.Equal(t, 2, result.Succeeded)
	assert.Equal(t, 1, result.Failed)
	assert.Len(t, result.Clusters, 3)
}

func TestUpgradeAllAlreadyAtVersion(t *testing.T) {
	backend := newMockBackend()
	backend.states["cluster-a"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "cluster-a", Version: "v1.28.0"},
		Machines: map[string]state.MachineState{
			"n1": {ID: "n1", Host: "10.0.0.1", K8sVersion: "v1.28.0", Status: "ready"},
		},
	}

	inv := NewInventory(backend)
	op := NewOperator(inv, backend)

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "cluster-a.yaml"},
			},
		},
	}

	result, err := op.UpgradeAll(context.Background(), fleetCfg, "v1.28.0", FleetUpgradeOpts{DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Succeeded)
	assert.Len(t, result.Clusters[0].Actions, 0)
}

func TestNewOperatorCreation(t *testing.T) {
	backend := newMockBackend()
	inv := NewInventory(backend)
	op := NewOperator(inv, backend)
	assert.NotNil(t, op)
}
