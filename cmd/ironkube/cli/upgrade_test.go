package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	upgradeCmd, _, err := cmd.Find([]string{"upgrade"})
	require.NoError(t, err)
	assert.Equal(t, "upgrade", upgradeCmd.Use)
}

func TestUpgradePlanSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	planCmd, _, err := cmd.Find([]string{"upgrade", "plan"})
	require.NoError(t, err)
	assert.Equal(t, "plan", planCmd.Use)
}

func TestUpgradeApplySubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	applyCmd, _, err := cmd.Find([]string{"upgrade", "apply"})
	require.NoError(t, err)
	assert.Equal(t, "apply", applyCmd.Use)
}

func TestUpgradePlanRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"upgrade", "plan", "--version", "v1.30.0"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestUpgradePlanRequiresVersion(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"upgrade", "plan", "--name", "test"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target version required")
}

func TestUpgradeApplyRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"upgrade", "apply", "--version", "v1.30.0"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestUpgradeApplyRequiresVersion(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"upgrade", "apply", "--name", "test"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target version required")
}
