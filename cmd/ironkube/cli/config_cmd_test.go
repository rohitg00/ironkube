package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	cfgCmd, _, err := cmd.Find([]string{"config"})
	require.NoError(t, err)
	assert.Equal(t, "config", cfgCmd.Use)
}

func TestConfigValidateSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	valCmd, _, err := cmd.Find([]string{"config", "validate"})
	require.NoError(t, err)
	assert.Equal(t, "validate", valCmd.Use)
}

func TestConfigValidateWithValidFile(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfig(t, dir)

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "validate", "--config", configPath})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "is valid")
}

func TestConfigValidateWithInvalidFile(t *testing.T) {
	dir := t.TempDir()
	badConfig := `apiVersion: ironkube.dev/v1alpha2
kind: ClusterConfig
metadata:
  name: ""
spec:
  distro: invalid
  version: ""
`
	path := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(path, []byte(badConfig), 0644))

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"config", "validate", "--config", path})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestConfigValidateWithMissingFile(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"config", "validate", "--config", "/nonexistent/file.yaml"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestConfigDefaultsSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	defCmd, _, err := cmd.Find([]string{"config", "defaults"})
	require.NoError(t, err)
	assert.Equal(t, "defaults", defCmd.Use)
}

func TestConfigDefaultsShowsConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTestConfig(t, dir)

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "defaults", "--config", configPath})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "test-cluster")
	assert.Contains(t, output, "k3s")
	assert.Contains(t, output, "minimal")
}

func TestConfigDefaultsAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	minimalConfig := `apiVersion: ironkube.dev/v1alpha2
kind: ClusterConfig
metadata:
  name: my-cluster
spec:
  distro: k3s
  version: "v1.28.2+k3s1"
  controlPlane:
    replicas: 1
    nodes:
    - host: "10.0.0.1"
`
	path := filepath.Join(dir, "minimal.yaml")
	require.NoError(t, os.WriteFile(path, []byte(minimalConfig), 0644))

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "defaults", "--config", path})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "10.42.0.0/16")
	assert.Contains(t, output, "10.43.0.0/16")
	assert.Contains(t, output, "minimal")
}
