package backup

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEtcd struct {
	backupErr  error
	restoreErr error
	backupPath string
	restoreSrc string
	restoreDir string
}

func (m *mockEtcd) Backup(_ context.Context, _ provider.Machine, outputPath string) error {
	m.backupPath = outputPath
	return m.backupErr
}

func (m *mockEtcd) Restore(_ context.Context, _ provider.Machine, snapshotPath, dataDir string) error {
	m.restoreSrc = snapshotPath
	m.restoreDir = dataDir
	return m.restoreErr
}

type mockStateExporter struct {
	data []byte
	err  error
}

func (m *mockStateExporter) ExportState(_ context.Context, _ string) ([]byte, error) {
	return m.data, m.err
}

func testMachine() provider.Machine {
	return provider.Machine{
		ID: "cp-1",
		Spec: provider.MachineSpec{
			Role: "control-plane",
			Host: "10.0.0.1",
			User: "root",
			Port: 22,
		},
	}
}

func TestBackupCreatesEtcdSnapshot(t *testing.T) {
	etcd := &mockEtcd{}
	state := &mockStateExporter{}
	mgr := New(etcd, state)

	result, err := mgr.Backup(context.Background(), testMachine(), "/backups", BackupOpts{})
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(result.EtcdSnapshotPath, "/backups/etcd-snapshot-"))
	assert.True(t, strings.HasSuffix(result.EtcdSnapshotPath, ".db"))
	assert.Equal(t, etcd.backupPath, result.EtcdSnapshotPath)
}

func TestBackupIncludesStateWhenOptSet(t *testing.T) {
	etcd := &mockEtcd{}
	state := &mockStateExporter{data: []byte(`{"cluster":"test"}`)}
	mgr := New(etcd, state)

	result, err := mgr.Backup(context.Background(), testMachine(), "/backups", BackupOpts{IncludeState: true})
	require.NoError(t, err)

	assert.NotEmpty(t, result.StatePath)
	assert.True(t, strings.HasPrefix(result.StatePath, "/backups/etcd-snapshot-state-"))
	assert.True(t, strings.HasSuffix(result.StatePath, ".json"))
}

func TestBackupWithoutStateExport(t *testing.T) {
	etcd := &mockEtcd{}
	state := &mockStateExporter{}
	mgr := New(etcd, state)

	result, err := mgr.Backup(context.Background(), testMachine(), "/backups", BackupOpts{})
	require.NoError(t, err)

	assert.Empty(t, result.StatePath)
}

func TestBackupEtcdErrorPropagated(t *testing.T) {
	etcd := &mockEtcd{backupErr: fmt.Errorf("disk full")}
	state := &mockStateExporter{}
	mgr := New(etcd, state)

	_, err := mgr.Backup(context.Background(), testMachine(), "/backups", BackupOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}

func TestBackupStateExportErrorPropagated(t *testing.T) {
	etcd := &mockEtcd{}
	state := &mockStateExporter{err: fmt.Errorf("state corrupted")}
	mgr := New(etcd, state)

	_, err := mgr.Backup(context.Background(), testMachine(), "/backups", BackupOpts{IncludeState: true})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state corrupted")
}

func TestRestoreRunsWithCorrectPaths(t *testing.T) {
	etcd := &mockEtcd{}
	state := &mockStateExporter{}
	mgr := New(etcd, state)

	err := mgr.Restore(context.Background(), testMachine(), "/backups/snap.db", RestoreOpts{})
	require.NoError(t, err)

	assert.Equal(t, "/backups/snap.db", etcd.restoreSrc)
	assert.Equal(t, "/var/lib/etcd", etcd.restoreDir)
}

func TestRestoreWithCustomDataDir(t *testing.T) {
	etcd := &mockEtcd{}
	state := &mockStateExporter{}
	mgr := New(etcd, state)

	err := mgr.Restore(context.Background(), testMachine(), "/backups/snap.db", RestoreOpts{DataDir: "/custom/etcd"})
	require.NoError(t, err)

	assert.Equal(t, "/custom/etcd", etcd.restoreDir)
}

func TestRestoreErrorPropagated(t *testing.T) {
	etcd := &mockEtcd{restoreErr: fmt.Errorf("snapshot corrupted")}
	state := &mockStateExporter{}
	mgr := New(etcd, state)

	err := mgr.Restore(context.Background(), testMachine(), "/backups/snap.db", RestoreOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot corrupted")
}

func TestBackupResultHasCorrectTimestampAndMachine(t *testing.T) {
	etcd := &mockEtcd{}
	state := &mockStateExporter{}
	mgr := New(etcd, state)

	machine := testMachine()
	result, err := mgr.Backup(context.Background(), machine, "/backups", BackupOpts{})
	require.NoError(t, err)

	assert.Equal(t, machine.ID, result.Machine.ID)
	assert.Equal(t, machine.Spec.Host, result.Machine.Spec.Host)
	assert.False(t, result.Timestamp.IsZero())
}

func TestEmptyOutputDirReturnsError(t *testing.T) {
	etcd := &mockEtcd{}
	state := &mockStateExporter{}
	mgr := New(etcd, state)

	_, err := mgr.Backup(context.Background(), testMachine(), "", BackupOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output directory is required")
}

func TestBackupWithCustomPrefix(t *testing.T) {
	etcd := &mockEtcd{}
	state := &mockStateExporter{}
	mgr := New(etcd, state)

	result, err := mgr.Backup(context.Background(), testMachine(), "/backups", BackupOpts{Prefix: "my-cluster"})
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(result.EtcdSnapshotPath, "/backups/my-cluster-"))
}
