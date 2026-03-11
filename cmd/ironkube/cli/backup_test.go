package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	backupCmd, _, err := cmd.Find([]string{"backup"})
	require.NoError(t, err)
	assert.Equal(t, "backup", backupCmd.Use)
}

func TestBackupCreateSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	createCmd, _, err := cmd.Find([]string{"backup", "create"})
	require.NoError(t, err)
	assert.Equal(t, "create", createCmd.Use)
}

func TestBackupRestoreSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	restoreCmd, _, err := cmd.Find([]string{"backup", "restore"})
	require.NoError(t, err)
	assert.Equal(t, "restore", restoreCmd.Use)
}

func TestBackupCreateRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"backup", "create"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestBackupRestoreRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"backup", "restore", "--snapshot", "/tmp/snap.db"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestBackupRestoreRequiresSnapshot(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"backup", "restore", "--name", "test"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot path required")
}
