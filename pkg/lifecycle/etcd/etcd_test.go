package etcd

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSSH struct {
	responses map[string]string
	errors    map[string]error
	calls     []string
}

func newMockSSH() *mockSSH {
	return &mockSSH{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *mockSSH) Run(_ context.Context, host, user string, port int, command string) (string, error) {
	m.calls = append(m.calls, command)
	if err, ok := m.errors[command]; ok {
		return "", err
	}
	if resp, ok := m.responses[command]; ok {
		return resp, nil
	}
	return "", fmt.Errorf("unexpected command: %s", command)
}

func testMachine() provider.Machine {
	return provider.Machine{
		ID: "etcd-1",
		Spec: provider.MachineSpec{
			Role: "control-plane",
			Host: "10.0.0.1",
			User: "root",
			Port: 22,
		},
	}
}

func TestHealthParsesHealthyCluster(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses[etcdEnv+"etcdctl endpoint health --cluster -w json"] = `[{"endpoint":"https://10.0.0.1:2379","health":"true"},{"endpoint":"https://10.0.0.2:2379","health":"true"}]`
	ssh.responses[etcdEnv+"etcdctl member list -w json"] = `{"members":[{"name":"etcd-0","clientURLs":["https://10.0.0.1:2379"],"ID":1},{"name":"etcd-1","clientURLs":["https://10.0.0.2:2379"],"ID":2}]}`
	ssh.responses[etcdEnv+"etcdctl endpoint status -w json"] = `[{"Endpoint":"https://10.0.0.1:2379","Status":{"dbSize":1048576,"version":"3.5.9","leader":1}},{"Endpoint":"https://10.0.0.2:2379","Status":{"dbSize":1048576,"version":"3.5.9","leader":1}}]`

	mgr := New(ssh)
	health, err := mgr.Health(context.Background(), testMachine())
	require.NoError(t, err)

	assert.True(t, health.Healthy)
	assert.Len(t, health.Members, 2)
	assert.Equal(t, "3.5.9", health.Version)
	assert.Equal(t, int64(2097152), health.DbSize)

	assert.Equal(t, "etcd-0", health.Members[0].Name)
	assert.True(t, health.Members[0].IsLeader)
	assert.Equal(t, "https://10.0.0.1:2379", health.Members[0].Endpoint)
	assert.False(t, health.Members[1].IsLeader)
}

func TestHealthHandlesUnhealthyMember(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses[etcdEnv+"etcdctl endpoint health --cluster -w json"] = `[{"endpoint":"https://10.0.0.1:2379","health":"true"},{"endpoint":"https://10.0.0.2:2379","health":"false"}]`
	ssh.responses[etcdEnv+"etcdctl member list -w json"] = `{"members":[{"name":"etcd-0","clientURLs":["https://10.0.0.1:2379"],"ID":1}]}`
	ssh.responses[etcdEnv+"etcdctl endpoint status -w json"] = `[{"Endpoint":"https://10.0.0.1:2379","Status":{"dbSize":524288,"version":"3.5.9","leader":1}}]`

	mgr := New(ssh)
	health, err := mgr.Health(context.Background(), testMachine())
	require.NoError(t, err)

	assert.False(t, health.Healthy)
}

func TestHealthSSHError(t *testing.T) {
	ssh := newMockSSH()
	ssh.errors[etcdEnv+"etcdctl endpoint health --cluster -w json"] = fmt.Errorf("connection refused")

	mgr := New(ssh)
	_, err := mgr.Health(context.Background(), testMachine())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestHealthMemberListError(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses[etcdEnv+"etcdctl endpoint health --cluster -w json"] = `[{"endpoint":"https://10.0.0.1:2379","health":"true"}]`
	ssh.errors[etcdEnv+"etcdctl member list -w json"] = fmt.Errorf("permission denied")

	mgr := New(ssh)
	_, err := mgr.Health(context.Background(), testMachine())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "listing etcd members")
}

func TestBackupRunsCorrectCommand(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses[etcdEnv+"etcdctl snapshot save /backup/etcd.db"] = "Snapshot saved"

	mgr := New(ssh)
	err := mgr.Backup(context.Background(), testMachine(), "/backup/etcd.db")
	require.NoError(t, err)
	assert.Contains(t, ssh.calls, etcdEnv+"etcdctl snapshot save /backup/etcd.db")
}

func TestBackupSSHError(t *testing.T) {
	ssh := newMockSSH()
	ssh.errors[etcdEnv+"etcdctl snapshot save /backup/etcd.db"] = fmt.Errorf("disk full")

	mgr := New(ssh)
	err := mgr.Backup(context.Background(), testMachine(), "/backup/etcd.db")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}

func TestRestoreRunsCorrectCommand(t *testing.T) {
	ssh := newMockSSH()
	expectedCmd := etcdEnv + "etcdctl snapshot restore /backup/etcd.db --data-dir=/var/lib/etcd-restore"
	ssh.responses[expectedCmd] = "Restored"

	mgr := New(ssh)
	err := mgr.Restore(context.Background(), testMachine(), "/backup/etcd.db", "/var/lib/etcd-restore")
	require.NoError(t, err)
	assert.Contains(t, ssh.calls, expectedCmd)
}

func TestRestoreSSHError(t *testing.T) {
	ssh := newMockSSH()
	ssh.errors[etcdEnv+"etcdctl snapshot restore /backup/etcd.db --data-dir=/var/lib/etcd"] = fmt.Errorf("corrupted snapshot")

	mgr := New(ssh)
	err := mgr.Restore(context.Background(), testMachine(), "/backup/etcd.db", "/var/lib/etcd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "corrupted snapshot")
}

func TestDefragRunsCorrectCommand(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses[etcdEnv+"etcdctl defrag --cluster"] = "Finished"

	mgr := New(ssh)
	err := mgr.Defrag(context.Background(), testMachine())
	require.NoError(t, err)
	assert.Contains(t, ssh.calls, etcdEnv+"etcdctl defrag --cluster")
}

func TestDefragError(t *testing.T) {
	ssh := newMockSSH()
	ssh.errors[etcdEnv+"etcdctl defrag --cluster"] = fmt.Errorf("timeout")

	mgr := New(ssh)
	err := mgr.Defrag(context.Background(), testMachine())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestCompactRunsCorrectCommand(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses[etcdEnv+"etcdctl compact 12345"] = "compacted"

	mgr := New(ssh)
	err := mgr.Compact(context.Background(), testMachine(), "12345")
	require.NoError(t, err)
	assert.Contains(t, ssh.calls, etcdEnv+"etcdctl compact 12345")
}

func TestCompactError(t *testing.T) {
	ssh := newMockSSH()
	ssh.errors[etcdEnv+"etcdctl compact 99999"] = fmt.Errorf("revision already compacted")

	mgr := New(ssh)
	err := mgr.Compact(context.Background(), testMachine(), "99999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revision already compacted")
}

func TestCommandsHaveEtcdAPIPrefix(t *testing.T) {
	ssh := newMockSSH()
	ssh.responses[etcdEnv+"etcdctl endpoint health --cluster -w json"] = `[{"endpoint":"https://10.0.0.1:2379","health":"true"}]`
	ssh.responses[etcdEnv+"etcdctl member list -w json"] = `{"members":[]}`
	ssh.responses[etcdEnv+"etcdctl endpoint status -w json"] = `[]`
	ssh.responses[etcdEnv+"etcdctl snapshot save /tmp/backup.db"] = ""
	ssh.responses[etcdEnv+"etcdctl defrag --cluster"] = ""
	ssh.responses[etcdEnv+"etcdctl compact 100"] = ""

	mgr := New(ssh)
	machine := testMachine()

	_, _ = mgr.Health(context.Background(), machine)
	_ = mgr.Backup(context.Background(), machine, "/tmp/backup.db")
	_ = mgr.Defrag(context.Background(), machine)
	_ = mgr.Compact(context.Background(), machine, "100")

	for _, call := range ssh.calls {
		assert.True(t, strings.HasPrefix(call, "ETCDCTL_API=3 "), "command should have ETCDCTL_API=3 prefix: %s", call)
	}
}
