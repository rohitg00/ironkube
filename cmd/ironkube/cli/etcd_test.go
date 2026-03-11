package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEtcdCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	etcdCmd, _, err := cmd.Find([]string{"etcd"})
	require.NoError(t, err)
	assert.Equal(t, "etcd", etcdCmd.Use)
}

func TestEtcdHealthSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	healthCmd, _, err := cmd.Find([]string{"etcd", "health"})
	require.NoError(t, err)
	assert.Equal(t, "health", healthCmd.Use)
}

func TestEtcdBackupSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	backupCmd, _, err := cmd.Find([]string{"etcd", "backup"})
	require.NoError(t, err)
	assert.Equal(t, "backup", backupCmd.Use)
}

func TestEtcdDefragSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	defragCmd, _, err := cmd.Find([]string{"etcd", "defrag"})
	require.NoError(t, err)
	assert.Equal(t, "defrag", defragCmd.Use)
}

func TestEtcdHealthRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"etcd", "health"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestEtcdHealthWithMissingState(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0700))

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"etcd", "health", "--name", "nonexistent", "--state-dir", stateDir})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading state")
}

func TestEtcdBackupRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"etcd", "backup"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestEtcdBackupWithMissingState(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0700))

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"etcd", "backup", "--name", "nonexistent", "--state-dir", stateDir})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading state")
}

func TestEtcdDefragRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"etcd", "defrag"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}
