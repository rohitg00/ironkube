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

func TestSecurityCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	secCmd, _, err := cmd.Find([]string{"security"})
	require.NoError(t, err)
	assert.Equal(t, "security", secCmd.Use)
}

func TestSecurityProfilesListsProfiles(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"security", "profiles"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "minimal")
	assert.Contains(t, output, "cis")
	assert.Contains(t, output, "hardened")
	assert.Contains(t, output, "Available security profiles")
}

func TestSecurityAuditSubcommandExists(t *testing.T) {
	cmd := NewRootCmd()
	auditCmd, _, err := cmd.Find([]string{"security", "audit"})
	require.NoError(t, err)
	assert.Equal(t, "audit", auditCmd.Use)
}

func TestSecurityAuditRequiresProfile(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0700))

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"security", "audit", "--name", "test", "--state-dir", stateDir})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--profile is required")
}

func TestSecurityAuditRequiresName(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"security", "audit", "--profile", "cis"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}

func TestSecurityAuditMinimalProfile(t *testing.T) {
	dir := t.TempDir()
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
	writeExistingState(t, stateDir, "audit-cluster", machines, nil, "v1.28.2+k3s1")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"security", "audit", "--name", "audit-cluster", "--profile", "minimal", "--state-dir", stateDir})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Security Audit")
	assert.Contains(t, output, "100.0%")
	assert.Contains(t, output, "all checks passed")
}
