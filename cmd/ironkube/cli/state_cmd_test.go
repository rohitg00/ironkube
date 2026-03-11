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

func TestStateCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	stateCmd, _, err := cmd.Find([]string{"state"})
	require.NoError(t, err)
	assert.Equal(t, "state", stateCmd.Use)
}

func TestStateListSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	listCmd, _, err := cmd.Find([]string{"state", "list"})
	require.NoError(t, err)
	assert.Equal(t, "list", listCmd.Use)
}

func TestStateListShowsClusters(t *testing.T) {
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
	writeExistingState(t, stateDir, "cluster-a", machines, nil, "v1.28.0")
	writeExistingState(t, stateDir, "cluster-b", nil, nil, "v1.29.0")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"state", "list", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "cluster-a")
	assert.Contains(t, output, "cluster-b")
	assert.Contains(t, output, "Clusters (2)")
}

func TestStateListEmpty(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0700))

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"state", "list", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No clusters found")
}

func TestStateInspectSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	inspCmd, _, err := cmd.Find([]string{"state", "inspect"})
	require.NoError(t, err)
	assert.Equal(t, "inspect", inspCmd.Use)
}

func TestStateInspectRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"state", "inspect"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}

func TestStateInspectShowsJSON(t *testing.T) {
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
	writeExistingState(t, stateDir, "my-cluster", machines, nil, "v1.28.0")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"state", "inspect", "--name", "my-cluster", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "my-cluster")
	assert.Contains(t, output, "10.0.0.1")
	assert.Contains(t, output, "control-plane")
}

func TestStateDeleteSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	delCmd, _, err := cmd.Find([]string{"state", "delete"})
	require.NoError(t, err)
	assert.Equal(t, "delete", delCmd.Use)
}

func TestStateDeleteRequiresYes(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	writeExistingState(t, stateDir, "my-cluster", nil, nil, "v1.28.0")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"state", "delete", "--name", "my-cluster", "--state-dir", stateDir})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--yes is required")
}

func TestStateDeleteRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"state", "delete", "--yes"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}

func TestStateDeleteRemovesState(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	writeExistingState(t, stateDir, "doomed", nil, nil, "v1.28.0")

	_, err := os.Stat(filepath.Join(stateDir, "doomed.json"))
	require.NoError(t, err)

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"state", "delete", "--name", "doomed", "--yes", "--state-dir", stateDir})

	err = cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "deleted")

	_, err = os.Stat(filepath.Join(stateDir, "doomed.json"))
	assert.True(t, os.IsNotExist(err))
}
