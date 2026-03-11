package fleet

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
	"github.com/rohitg00/ironkube/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBackend struct {
	states map[string]*state.ClusterState
	err    map[string]error
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		states: make(map[string]*state.ClusterState),
		err:    make(map[string]error),
	}
}

func (m *mockBackend) Load(name string) (*state.ClusterState, error) {
	if err, ok := m.err[name]; ok {
		return nil, err
	}
	st, ok := m.states[name]
	if !ok {
		return nil, fmt.Errorf("state not found for %q", name)
	}
	return st, nil
}

func (m *mockBackend) Save(s *state.ClusterState) error {
	m.states[s.Metadata.Name] = s
	return nil
}

func (m *mockBackend) Lock(name string) (state.UnlockFunc, error) {
	return func() {}, nil
}

func (m *mockBackend) List() ([]string, error) {
	var names []string
	for k := range m.states {
		names = append(names, k)
	}
	return names, nil
}

func (m *mockBackend) Delete(name string) error {
	delete(m.states, name)
	return nil
}

func TestLoadFleetConfigValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.yaml")
	content := `apiVersion: ironkube.dev/v1alpha2
kind: Fleet
metadata:
  name: test-fleet
spec:
  clusters:
    - path: clusters/eu-west.yaml
      labels:
        region: eu-west-1
    - path: clusters/us-east.yaml
      labels:
        region: us-east-1
  policies:
    maxConcurrentUpgrades: 2
    upgradeWindow: "Sat 02:00-06:00 UTC"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := LoadFleetConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "test-fleet", cfg.Metadata.Name)
	assert.Len(t, cfg.Spec.Clusters, 2)
	assert.Equal(t, "clusters/eu-west.yaml", cfg.Spec.Clusters[0].Path)
	assert.Equal(t, "eu-west-1", cfg.Spec.Clusters[0].Labels["region"])
	assert.Equal(t, 2, cfg.Spec.Policies.MaxConcurrentUpgrades)
}

func TestLoadFleetConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.yaml")
	require.NoError(t, os.WriteFile(path, []byte(":::invalid yaml"), 0644))

	_, err := LoadFleetConfig(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing fleet config YAML")
}

func TestLoadFleetConfigMissingFile(t *testing.T) {
	_, err := LoadFleetConfig("/nonexistent/fleet.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading fleet config")
}

func TestLoadFleetConfigWrongKind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.yaml")
	content := `apiVersion: ironkube.dev/v1alpha2
kind: ClusterConfig
metadata:
  name: not-a-fleet
spec:
  clusters:
    - path: a.yaml
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := LoadFleetConfig(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected kind Fleet")
}

func TestLoadFleetConfigMissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.yaml")
	content := `apiVersion: ironkube.dev/v1alpha2
kind: Fleet
metadata:
  name: ""
spec:
  clusters:
    - path: a.yaml
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := LoadFleetConfig(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name is required")
}

func TestLoadFleetConfigNoClusters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.yaml")
	content := `apiVersion: ironkube.dev/v1alpha2
kind: Fleet
metadata:
  name: empty-fleet
spec:
  clusters: []
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := LoadFleetConfig(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one cluster")
}

func TestCollectAllClustersAvailable(t *testing.T) {
	backend := newMockBackend()
	backend.states["eu-west"] = &state.ClusterState{
		Metadata: state.StateMetadata{
			Name:      "eu-west",
			Version:   "v1.28.0",
			CreatedAt: time.Now(),
		},
		Machines: map[string]state.MachineState{
			"node-1": {ID: "node-1", Host: "10.0.0.1", Role: "control-plane", Status: "ready", K8sVersion: "v1.28.0"},
		},
	}
	backend.states["us-east"] = &state.ClusterState{
		Metadata: state.StateMetadata{
			Name:      "us-east",
			Version:   "v1.28.0",
			CreatedAt: time.Now(),
		},
		Machines: map[string]state.MachineState{
			"node-2": {ID: "node-2", Host: "10.0.1.1", Role: "control-plane", Status: "ready", K8sVersion: "v1.28.0"},
		},
	}

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "clusters/eu-west.yaml", Labels: map[string]string{"region": "eu-west-1"}},
				{Path: "clusters/us-east.yaml", Labels: map[string]string{"region": "us-east-1"}},
			},
		},
	}

	inv := NewInventory(backend)
	summaries, err := inv.Collect(context.Background(), fleetCfg)
	require.NoError(t, err)
	assert.Len(t, summaries, 2)

	for _, s := range summaries {
		assert.True(t, s.Available)
		assert.NotNil(t, s.State)
		assert.Empty(t, s.Error)
	}
}

func TestCollectSomeClustersMissingState(t *testing.T) {
	backend := newMockBackend()
	backend.states["eu-west"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "eu-west", Version: "v1.28.0"},
		Machines: map[string]state.MachineState{
			"node-1": {ID: "node-1", Status: "ready"},
		},
	}

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "eu-west.yaml"},
				{Path: "us-east.yaml"},
			},
		},
	}

	inv := NewInventory(backend)
	summaries, err := inv.Collect(context.Background(), fleetCfg)
	require.NoError(t, err)
	assert.Len(t, summaries, 2)

	assert.True(t, summaries[0].Available)
	assert.False(t, summaries[1].Available)
	assert.Contains(t, summaries[1].Error, "failed to load state")
}

func TestCollectEmptyFleet(t *testing.T) {
	backend := newMockBackend()
	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "empty"},
		Spec:     v1alpha2.FleetSpec{},
	}

	inv := NewInventory(backend)
	summaries, err := inv.Collect(context.Background(), fleetCfg)
	require.NoError(t, err)
	assert.Len(t, summaries, 0)
}

func TestCollectDoesNotFailOnSingleClusterError(t *testing.T) {
	backend := newMockBackend()
	backend.err["broken"] = fmt.Errorf("disk failure")
	backend.states["healthy"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "healthy", Version: "v1.28.0"},
	}

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "broken.yaml"},
				{Path: "healthy.yaml"},
			},
		},
	}

	inv := NewInventory(backend)
	summaries, err := inv.Collect(context.Background(), fleetCfg)
	require.NoError(t, err)
	assert.Len(t, summaries, 2)

	assert.False(t, summaries[0].Available)
	assert.Contains(t, summaries[0].Error, "disk failure")
	assert.True(t, summaries[1].Available)
}

func TestFilterByLabelMatches(t *testing.T) {
	summaries := []ClusterSummary{
		{Name: "eu-west", Labels: map[string]string{"region": "eu-west-1", "env": "prod"}},
		{Name: "us-east", Labels: map[string]string{"region": "us-east-1", "env": "prod"}},
		{Name: "ap-south", Labels: map[string]string{"region": "ap-south-1", "env": "staging"}},
	}

	filtered := FilterByLabel(summaries, "region", "eu-west-1")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "eu-west", filtered[0].Name)
}

func TestFilterByLabelNoMatch(t *testing.T) {
	summaries := []ClusterSummary{
		{Name: "eu-west", Labels: map[string]string{"region": "eu-west-1"}},
		{Name: "us-east", Labels: map[string]string{"region": "us-east-1"}},
	}

	filtered := FilterByLabel(summaries, "region", "ap-south-1")
	assert.Len(t, filtered, 0)
}

func TestFilterByLabelMultipleMatches(t *testing.T) {
	summaries := []ClusterSummary{
		{Name: "prod-1", Labels: map[string]string{"env": "prod"}},
		{Name: "prod-2", Labels: map[string]string{"env": "prod"}},
		{Name: "staging", Labels: map[string]string{"env": "staging"}},
	}

	filtered := FilterByLabel(summaries, "env", "prod")
	assert.Len(t, filtered, 2)
}

func TestFilterByLabelNilLabels(t *testing.T) {
	summaries := []ClusterSummary{
		{Name: "no-labels", Labels: nil},
		{Name: "with-labels", Labels: map[string]string{"env": "prod"}},
	}

	filtered := FilterByLabel(summaries, "env", "prod")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "with-labels", filtered[0].Name)
}

func TestSummaryCountsCorrectly(t *testing.T) {
	summaries := []ClusterSummary{
		{
			Name:      "cluster-a",
			Available: true,
			State: &state.ClusterState{
				Metadata: state.StateMetadata{Version: "v1.28.0"},
				Machines: map[string]state.MachineState{
					"n1": {Status: "ready"},
					"n2": {Status: "ready"},
				},
			},
		},
		{
			Name:      "cluster-b",
			Available: true,
			State: &state.ClusterState{
				Metadata: state.StateMetadata{Version: "v1.28.0"},
				Machines: map[string]state.MachineState{
					"n3": {Status: "ready"},
				},
			},
		},
	}

	fs := Summary(summaries)
	assert.Equal(t, 2, fs.TotalClusters)
	assert.Equal(t, 2, fs.AvailableClusters)
	assert.Equal(t, 3, fs.TotalMachines)
	assert.Equal(t, 3, fs.ReadyMachines)
	assert.Equal(t, 2, fs.Versions["v1.28.0"])
}

func TestSummaryWithMixedVersions(t *testing.T) {
	summaries := []ClusterSummary{
		{
			Name:      "cluster-a",
			Available: true,
			State: &state.ClusterState{
				Metadata: state.StateMetadata{Version: "v1.28.0"},
				Machines: map[string]state.MachineState{
					"n1": {Status: "ready"},
				},
			},
		},
		{
			Name:      "cluster-b",
			Available: true,
			State: &state.ClusterState{
				Metadata: state.StateMetadata{Version: "v1.27.0"},
				Machines: map[string]state.MachineState{
					"n2": {Status: "ready"},
					"n3": {Status: "not-ready"},
				},
			},
		},
		{
			Name:      "cluster-c",
			Available: false,
			Error:     "unavailable",
		},
	}

	fs := Summary(summaries)
	assert.Equal(t, 3, fs.TotalClusters)
	assert.Equal(t, 2, fs.AvailableClusters)
	assert.Equal(t, 3, fs.TotalMachines)
	assert.Equal(t, 2, fs.ReadyMachines)
	assert.Equal(t, 1, fs.Versions["v1.28.0"])
	assert.Equal(t, 1, fs.Versions["v1.27.0"])
}

func TestSummaryEmptyList(t *testing.T) {
	fs := Summary(nil)
	assert.Equal(t, 0, fs.TotalClusters)
	assert.Equal(t, 0, fs.AvailableClusters)
	assert.Equal(t, 0, fs.TotalMachines)
	assert.Equal(t, 0, fs.ReadyMachines)
	assert.Empty(t, fs.Versions)
}

func TestClusterNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"clusters/eu-west.yaml", "eu-west"},
		{"clusters/us-east.yml", "us-east"},
		{"simple.yaml", "simple"},
		{"no-extension", "no-extension"},
		{"deep/nested/path/cluster.yaml", "cluster"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, clusterNameFromPath(tt.path))
		})
	}
}

func TestCollectPreservesLabels(t *testing.T) {
	backend := newMockBackend()
	backend.states["cluster-a"] = &state.ClusterState{
		Metadata: state.StateMetadata{Name: "cluster-a"},
	}

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "cluster-a.yaml", Labels: map[string]string{"env": "prod", "tier": "critical"}},
			},
		},
	}

	inv := NewInventory(backend)
	summaries, err := inv.Collect(context.Background(), fleetCfg)
	require.NoError(t, err)
	assert.Equal(t, "prod", summaries[0].Labels["env"])
	assert.Equal(t, "critical", summaries[0].Labels["tier"])
}

func TestCollectContextCancellation(t *testing.T) {
	backend := newMockBackend()
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("cluster-%d", i)
		backend.states[name] = &state.ClusterState{
			Metadata: state.StateMetadata{Name: name},
		}
	}

	fleetCfg := &v1alpha2.FleetConfig{
		Metadata: v1alpha2.Metadata{Name: "test-fleet"},
		Spec: v1alpha2.FleetSpec{
			Clusters: []v1alpha2.ClusterRef{
				{Path: "cluster-0.yaml"},
				{Path: "cluster-1.yaml"},
				{Path: "cluster-2.yaml"},
				{Path: "cluster-3.yaml"},
				{Path: "cluster-4.yaml"},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	inv := NewInventory(backend)
	_, err := inv.Collect(ctx, fleetCfg)
	assert.ErrorIs(t, err, context.Canceled)
}
