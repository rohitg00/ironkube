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

func TestDestroyCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	destroyCmd, _, err := cmd.Find([]string{"destroy"})
	require.NoError(t, err)
	assert.Equal(t, "destroy", destroyCmd.Use)
}

func TestDestroyRequiresNameOrConfig(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"destroy"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestDestroyShowsConfirmation(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")

	machines := map[string]state.MachineState{
		"10.0.0.1": {
			ID:     "10.0.0.1",
			Host:   "10.0.0.1",
			Role:   "control-plane",
			Status: "ready",
		},
	}
	writeExistingState(t, stateDir, "test-cluster", machines, nil, "v1.28.2+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"destroy", "--name", "test-cluster", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "will be destroyed")
	assert.Contains(t, output, "Machines: 1")
	assert.Contains(t, output, "--yes")
}

func TestDestroyWithYesDeletesState(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")

	machines := map[string]state.MachineState{
		"10.0.0.1": {
			ID:     "10.0.0.1",
			Host:   "10.0.0.1",
			Role:   "control-plane",
			Status: "ready",
		},
	}
	writeExistingState(t, stateDir, "test-cluster", machines, nil, "v1.28.2+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"destroy", "--name", "test-cluster", "--yes", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "destroyed successfully")

	_, err = os.Stat(filepath.Join(stateDir, "test-cluster.json"))
	assert.True(t, os.IsNotExist(err))
}

func TestDestroyWithConfigFlag(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfig(t, dir)
	stateDir := filepath.Join(dir, "state")

	machines := map[string]state.MachineState{
		"10.0.0.1": {
			ID:     "10.0.0.1",
			Host:   "10.0.0.1",
			Role:   "control-plane",
			Status: "ready",
		},
	}
	writeExistingState(t, stateDir, "test-cluster", machines, nil, "v1.28.2+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"destroy", "--config", configPath, "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "will be destroyed")
}

func TestDestroyMissingClusterState(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0700))

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"destroy", "--name", "nonexistent", "--state-dir", stateDir})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no state found")
}

func TestDestroyShowsMachinesAndAddons(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")

	machines := map[string]state.MachineState{
		"10.0.0.1": {
			ID:     "10.0.0.1",
			Host:   "10.0.0.1",
			Role:   "control-plane",
			Status: "ready",
		},
	}
	addons := map[string]state.AddonState{
		"cilium": {
			Name:      "cilium",
			Version:   "1.16.0",
			Installed: true,
		},
	}
	writeExistingState(t, stateDir, "test-cluster", machines, addons, "v1.28.2+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"destroy", "--name", "test-cluster", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Machines: 1")
	assert.Contains(t, output, "Addons:   1")
	assert.Contains(t, output, "10.0.0.1")
	assert.Contains(t, output, "cilium")
}
