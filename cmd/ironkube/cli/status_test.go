package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rohitg00/ironkube/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	statusCmd, _, err := cmd.Find([]string{"status"})
	require.NoError(t, err)
	assert.Equal(t, "status", statusCmd.Use)
}

func TestStatusRequiresNameOrConfig(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"status"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestStatusNoClusterFound(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0700))

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"status", "--name", "nonexistent", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No state found")
	assert.Contains(t, output, "ironkube apply")
}

func TestStatusShowsMachines(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")

	machines := map[string]state.MachineState{
		"10.0.0.1": {
			ID:         "10.0.0.1",
			Host:       "10.0.0.1",
			Role:       "control-plane",
			K8sVersion: "v1.28.2+k3s1",
			Status:     "ready",
		},
	}
	writeExistingState(t, stateDir, "my-cluster", machines, nil, "v1.28.2+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"status", "--name", "my-cluster", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Cluster: my-cluster")
	assert.Contains(t, output, "Version: v1.28.2+k3s1")
	assert.Contains(t, output, "Machines (1)")
	assert.Contains(t, output, "10.0.0.1")
	assert.Contains(t, output, "control-plane")
	assert.Contains(t, output, "ready")
}

func TestStatusShowsAddons(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")

	addons := map[string]state.AddonState{
		"cilium": {
			Name:      "cilium",
			Version:   "1.16.0",
			Installed: true,
		},
	}
	writeExistingState(t, stateDir, "my-cluster", nil, addons, "v1.28.2+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"status", "--name", "my-cluster", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Addons (1)")
	assert.Contains(t, output, "cilium")
	assert.Contains(t, output, "1.16.0")
	assert.Contains(t, output, "installed")
}

func TestStatusWithConfigFlag(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfig(t, dir)
	stateDir := filepath.Join(dir, "state")

	machines := map[string]state.MachineState{
		"10.0.0.1": {
			ID:         "10.0.0.1",
			Host:       "10.0.0.1",
			Role:       "control-plane",
			K8sVersion: "v1.28.2+k3s1",
			Status:     "ready",
		},
	}
	writeExistingState(t, stateDir, "test-cluster", machines, nil, "v1.28.2+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"status", "--config", configPath, "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Cluster: test-cluster")
}

func TestStatusEmptyCluster(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")

	writeExistingState(t, stateDir, "empty-cluster", nil, nil, "v1.28.2+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"status", "--name", "empty-cluster", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Machines (0)")
	assert.Contains(t, output, "(none)")
	assert.Contains(t, output, "Addons (0)")
}
