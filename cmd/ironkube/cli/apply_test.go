package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rohitg00/ironkube/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestConfig(t *testing.T, dir string) string {
	t.Helper()
	cfg := `apiVersion: ironkube.dev/v1alpha2
kind: ClusterConfig
metadata:
  name: test-cluster
spec:
  distro: k3s
  version: "v1.28.2+k3s1"
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
    profile: minimal`
	path := filepath.Join(dir, "test-config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0644))
	return path
}

func writeTestConfigWithAddons(t *testing.T, dir string) string {
	t.Helper()
	cfg := `apiVersion: ironkube.dev/v1alpha2
kind: ClusterConfig
metadata:
  name: addon-cluster
spec:
  distro: k3s
  version: "v1.28.2+k3s1"
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
  addons:
    - name: cilium
      enabled: true
      version: "1.16.0"
      namespace: kube-system`
	path := filepath.Join(dir, "addon-config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0644))
	return path
}

func writeExistingState(t *testing.T, stateDir, clusterName string, machines map[string]state.MachineState, addons map[string]state.AddonState, version string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(stateDir, 0700))
	st := &state.ClusterState{
		Metadata: state.StateMetadata{
			Name:      clusterName,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   version,
		},
		Machines: machines,
		Addons:   addons,
	}
	data, err := json.MarshalIndent(st, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, clusterName+".json"), data, 0600))
}

func TestApplyCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	applyCmd, _, err := cmd.Find([]string{"apply"})
	require.NoError(t, err)
	assert.Equal(t, "apply", applyCmd.Use)
}

func TestApplyDryRunNewCluster(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfig(t, dir)
	stateDir := filepath.Join(dir, "state")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"apply", "--config", configPath, "--dry-run", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "create-machine")
	assert.Contains(t, output, "10.0.0.1")
	assert.Contains(t, output, "Dry run")
}

func TestApplyDryRunShowsCreateActions(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfig(t, dir)
	stateDir := filepath.Join(dir, "state")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"apply", "--config", configPath, "--dry-run", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Planned actions")
	assert.Contains(t, output, "create-machine")
	assert.Contains(t, output, "control-plane")
}

func TestApplyMissingConfig(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"apply", "--config", "/nonexistent/config.yaml"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestApplyCreatesState(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfig(t, dir)
	stateDir := filepath.Join(dir, "state")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"apply", "--config", configPath, "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "applied successfully")

	_, err = os.Stat(filepath.Join(stateDir, "test-cluster.json"))
	assert.NoError(t, err)
}

func TestApplyInSyncCluster(t *testing.T) {
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
	cmd.SetArgs([]string{"apply", "--config", configPath, "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "already in sync")
}

func TestApplyWithAddons(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfigWithAddons(t, dir)
	stateDir := filepath.Join(dir, "state")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"apply", "--config", configPath, "--dry-run", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "install-addon")
	assert.Contains(t, output, "cilium")
}

func TestApplyDryRunDoesNotCreateState(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfig(t, dir)
	stateDir := filepath.Join(dir, "state")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"apply", "--config", configPath, "--dry-run", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(stateDir, "test-cluster.json"))
	assert.True(t, os.IsNotExist(err))
}
