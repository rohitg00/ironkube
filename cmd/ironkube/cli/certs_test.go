package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCertsCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	certsCmd, _, err := cmd.Find([]string{"certs"})
	require.NoError(t, err)
	assert.Equal(t, "certs", certsCmd.Use)
}

func TestCertsCheckSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	checkCmd, _, err := cmd.Find([]string{"certs", "check"})
	require.NoError(t, err)
	assert.Equal(t, "check", checkCmd.Use)
}

func TestCertsRotateSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	rotateCmd, _, err := cmd.Find([]string{"certs", "rotate"})
	require.NoError(t, err)
	assert.Equal(t, "rotate", rotateCmd.Use)
}

func TestCertsCheckRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"certs", "check"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestCertsCheckWithMissingState(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0700))

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"certs", "check", "--name", "nonexistent", "--state-dir", stateDir})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading state")
}

func TestCertsRotateRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"certs", "rotate"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name required")
}

func TestCertsRotateWithMissingState(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0700))

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"certs", "rotate", "--name", "nonexistent", "--state-dir", stateDir})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading state")
}
