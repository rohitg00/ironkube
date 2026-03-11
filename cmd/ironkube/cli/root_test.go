package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	cmd := NewRootCmd()
	assert.Equal(t, "ironkube", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "ironkube version")
}

func TestRootHelpDoesNotError(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestInitCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	initCmd, _, err := cmd.Find([]string{"init"})
	require.NoError(t, err)
	assert.Equal(t, "init", initCmd.Use)
}

func TestHealthCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	healthCmd, _, err := cmd.Find([]string{"health"})
	require.NoError(t, err)
	assert.Equal(t, "health", healthCmd.Use)
}

func TestHealthCommandOutput(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"health"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Health check")
}

func TestInitDryRunWithValidConfig(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init", "--config", "../../../examples/single-node-k3s.yaml", "--dry-run"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Dry run")
}

func TestInitWithMissingConfig(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"init", "--config", "/nonexistent.yaml"})
	err := cmd.Execute()
	assert.Error(t, err)
}
