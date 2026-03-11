package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAirgapCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	airgapCmd, _, err := cmd.Find([]string{"airgap"})
	require.NoError(t, err)
	assert.Equal(t, "airgap", airgapCmd.Use)
}

func TestAirgapBundleSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	bundleCmd, _, err := cmd.Find([]string{"airgap", "bundle"})
	require.NoError(t, err)
	assert.Equal(t, "bundle", bundleCmd.Use)
}

func TestAirgapLoadSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	loadCmd, _, err := cmd.Find([]string{"airgap", "load"})
	require.NoError(t, err)
	assert.Equal(t, "load", loadCmd.Use)
}

func TestAirgapBundleRequiresConfig(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"airgap", "bundle", "--output", "/tmp/bundle"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config required")
}

func TestAirgapBundleRequiresOutput(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"airgap", "bundle", "--config", "cluster.yaml"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output directory required")
}

func TestAirgapLoadRequiresBundle(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"airgap", "load", "--name", "mycluster"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bundle directory required")
}

func TestAirgapLoadRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"airgap", "load", "--bundle", "/tmp/bundle"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestAirgapBundleSuccess(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"airgap", "bundle", "--config", "cluster.yaml", "--output", "/tmp/bundle"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Creating airgap bundle")
}
