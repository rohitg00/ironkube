package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationTypeConstants(t *testing.T) {
	assert.Equal(t, OperationType("apply"), OpApply)
	assert.Equal(t, OperationType("destroy"), OpDestroy)
	assert.Equal(t, OperationType("upgrade"), OpUpgrade)
	assert.Equal(t, OperationType("backup"), OpBackup)
	assert.Equal(t, OperationType("restore"), OpRestore)
	assert.Equal(t, OperationType("cert-rotate"), OpCertRotate)
}

func TestOperationStatusConstants(t *testing.T) {
	assert.Equal(t, OperationStatus("running"), OpStatusRunning)
	assert.Equal(t, OperationStatus("success"), OpStatusSuccess)
	assert.Equal(t, OperationStatus("failed"), OpStatusFailed)
}

func TestRecordOperationAppendsHistory(t *testing.T) {
	s := &ClusterState{
		Metadata: StateMetadata{
			Name:      "test-cluster",
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Hour),
		},
	}

	op := OperationRecord{
		Type:      OpApply,
		Status:    OpStatusSuccess,
		StartedAt: time.Now(),
		Message:   "applied cluster",
	}

	s.RecordOperation(op)

	assert.Equal(t, op, s.LastOperation)
	require.Len(t, s.History, 1)
	assert.Equal(t, op, s.History[0])
}

func TestRecordOperationUpdatesTimestamp(t *testing.T) {
	before := time.Now().Add(-time.Second)
	s := &ClusterState{
		Metadata: StateMetadata{
			Name:      "test-cluster",
			UpdatedAt: before,
		},
	}

	s.RecordOperation(OperationRecord{
		Type:   OpDestroy,
		Status: OpStatusRunning,
	})

	assert.True(t, s.Metadata.UpdatedAt.After(before))
}

func TestRecordMultipleOperations(t *testing.T) {
	s := &ClusterState{
		Metadata: StateMetadata{Name: "test-cluster"},
	}

	op1 := OperationRecord{Type: OpApply, Status: OpStatusSuccess}
	op2 := OperationRecord{Type: OpUpgrade, Status: OpStatusFailed, Message: "timeout"}
	op3 := OperationRecord{Type: OpBackup, Status: OpStatusRunning}

	s.RecordOperation(op1)
	s.RecordOperation(op2)
	s.RecordOperation(op3)

	assert.Equal(t, op3, s.LastOperation)
	require.Len(t, s.History, 3)
	assert.Equal(t, op1, s.History[0])
	assert.Equal(t, op2, s.History[1])
	assert.Equal(t, op3, s.History[2])
}

func TestSetMachineCreatesMap(t *testing.T) {
	s := &ClusterState{}
	assert.Nil(t, s.Machines)

	m := MachineState{
		ID:   "node-1",
		Host: "10.0.0.1",
		Role: "control-plane",
	}

	s.SetMachine("node-1", m)

	require.NotNil(t, s.Machines)
	assert.Equal(t, m, s.Machines["node-1"])
}

func TestSetMachineOverwrites(t *testing.T) {
	s := &ClusterState{}

	s.SetMachine("node-1", MachineState{ID: "node-1", Host: "10.0.0.1", Role: "worker"})
	s.SetMachine("node-1", MachineState{ID: "node-1", Host: "10.0.0.2", Role: "control-plane"})

	require.Len(t, s.Machines, 1)
	assert.Equal(t, "10.0.0.2", s.Machines["node-1"].Host)
	assert.Equal(t, "control-plane", s.Machines["node-1"].Role)
}

func TestSetAddonCreatesMap(t *testing.T) {
	s := &ClusterState{}
	assert.Nil(t, s.Addons)

	a := AddonState{
		Name:      "cilium",
		Version:   "1.15.0",
		Installed: true,
		Ready:     true,
	}

	s.SetAddon("cilium", a)

	require.NotNil(t, s.Addons)
	assert.Equal(t, a, s.Addons["cilium"])
}

func TestSetAddonOverwrites(t *testing.T) {
	s := &ClusterState{}

	s.SetAddon("cilium", AddonState{Name: "cilium", Version: "1.14.0", Installed: true, Ready: false})
	s.SetAddon("cilium", AddonState{Name: "cilium", Version: "1.15.0", Installed: true, Ready: true})

	require.Len(t, s.Addons, 1)
	assert.Equal(t, "1.15.0", s.Addons["cilium"].Version)
	assert.True(t, s.Addons["cilium"].Ready)
}

func TestClusterStateStructFields(t *testing.T) {
	now := time.Now()
	s := ClusterState{
		Metadata: StateMetadata{
			Name:      "my-cluster",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   "v0.0.1",
		},
		Kubeconfig: []byte("kubeconfig-data"),
		Certs: []CertState{
			{Name: "ca", Path: "/etc/kubernetes/pki/ca.crt", ExpiresAt: now.Add(365 * 24 * time.Hour)},
		},
	}

	assert.Equal(t, "my-cluster", s.Metadata.Name)
	assert.Equal(t, "v0.0.1", s.Metadata.Version)
	assert.Equal(t, []byte("kubeconfig-data"), s.Kubeconfig)
	require.Len(t, s.Certs, 1)
	assert.Equal(t, "ca", s.Certs[0].Name)
}
