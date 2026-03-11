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

func writeFleetConfig(t *testing.T, dir string) string {
	t.Helper()
	content := `apiVersion: ironkube.dev/v1alpha2
kind: Fleet
metadata:
  name: test-fleet
spec:
  clusters:
    - path: cluster-a.yaml
      labels:
        region: eu-west-1
    - path: cluster-b.yaml
      labels:
        region: us-east-1
  policies:
    maxConcurrentUpgrades: 1
`
	path := filepath.Join(dir, "fleet.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestFleetCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	fleetCmd, _, err := cmd.Find([]string{"fleet"})
	require.NoError(t, err)
	assert.Equal(t, "fleet", fleetCmd.Use)
}

func TestFleetStatusSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	statusCmd, _, err := cmd.Find([]string{"fleet", "status"})
	require.NoError(t, err)
	assert.Equal(t, "status", statusCmd.Use)
}

func TestFleetDiffSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	diffCmd, _, err := cmd.Find([]string{"fleet", "diff"})
	require.NoError(t, err)
	assert.Equal(t, "diff", diffCmd.Use)
}

func TestFleetUpgradeSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	upgradeCmd, _, err := cmd.Find([]string{"fleet", "upgrade"})
	require.NoError(t, err)
	assert.Equal(t, "upgrade", upgradeCmd.Use)
}

func TestFleetStatusRequiresConfig(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"fleet", "status"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fleet config required")
}

func TestFleetDiffRequiresConfig(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"fleet", "diff"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fleet config required")
}

func TestFleetUpgradeRequiresConfigAndVersion(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"fleet", "upgrade"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fleet config required")
}

func TestFleetUpgradeRequiresVersion(t *testing.T) {
	dir := t.TempDir()
	fleetPath := writeFleetConfig(t, dir)

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"fleet", "upgrade", "--config", fleetPath, "--state-dir", filepath.Join(dir, "state")})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target version required")
}

func TestFleetStatusWithValidConfig(t *testing.T) {
	dir := t.TempDir()
	fleetPath := writeFleetConfig(t, dir)
	stateDir := filepath.Join(dir, "state")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"fleet", "status", "--config", fleetPath, "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Fleet: test-fleet")
	assert.Contains(t, output, "cluster-a")
	assert.Contains(t, output, "cluster-b")
	assert.Contains(t, output, "unavailable")
}

func TestFleetStatusWithExistingState(t *testing.T) {
	dir := t.TempDir()
	fleetPath := writeFleetConfig(t, dir)
	stateDir := filepath.Join(dir, "state")

	writeExistingState(t, stateDir, "cluster-a", map[string]state.MachineState{
		"10.0.0.1": {ID: "10.0.0.1", Host: "10.0.0.1", Role: "control-plane", K8sVersion: "v1.28.0", Status: "ready"},
	}, nil, "v1.28.0")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"fleet", "status", "--config", fleetPath, "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "cluster-a")
	assert.Contains(t, output, "available")
	assert.Contains(t, output, "v1.28.0")
}
