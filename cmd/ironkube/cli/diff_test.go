package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/rohitg00/ironkube/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	diffCmd, _, err := cmd.Find([]string{"diff"})
	require.NoError(t, err)
	assert.Equal(t, "diff", diffCmd.Use)
}

func TestDiffMissingConfig(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"diff", "--config", "/nonexistent/config.yaml"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestDiffNewClusterShowsActions(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfig(t, dir)
	stateDir := filepath.Join(dir, "state")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"diff", "--config", configPath, "--state-dir", stateDir})

	err := cmd.Execute()
	assert.ErrorIs(t, err, errDiffChanges)

	output := buf.String()
	assert.Contains(t, output, "Planned actions")
	assert.Contains(t, output, "create-machine")
	assert.Contains(t, output, "10.0.0.1")
}

func TestDiffInSyncShowsNoChanges(t *testing.T) {
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
	cmd.SetArgs([]string{"diff", "--config", configPath, "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "in sync")
}

func TestDiffShowsAddonChanges(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfigWithAddons(t, dir)
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
	writeExistingState(t, stateDir, "addon-cluster", machines, nil, "v1.28.2+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"diff", "--config", configPath, "--state-dir", stateDir})

	err := cmd.Execute()
	assert.ErrorIs(t, err, errDiffChanges)

	output := buf.String()
	assert.Contains(t, output, "install-addon")
	assert.Contains(t, output, "cilium")
}

func TestDiffVersionUpgrade(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfig(t, dir)
	stateDir := filepath.Join(dir, "state")

	machines := map[string]state.MachineState{
		"10.0.0.1": {
			ID:         "10.0.0.1",
			Host:       "10.0.0.1",
			Role:       "control-plane",
			K8sVersion: "v1.27.0+k3s1",
			Status:     "ready",
		},
	}
	writeExistingState(t, stateDir, "test-cluster", machines, nil, "v1.27.0+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"diff", "--config", configPath, "--state-dir", stateDir})

	err := cmd.Execute()
	assert.ErrorIs(t, err, errDiffChanges)

	output := buf.String()
	assert.Contains(t, output, "upgrade-version")
}
