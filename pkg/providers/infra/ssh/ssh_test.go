package ssh

import (
	"context"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ provider.InfraProvider = (*Provider)(nil)

func TestName(t *testing.T) {
	p := New()
	assert.Equal(t, "ssh", p.Name())
}

func TestValidateValidSpecs(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "10.0.0.1", User: "root", Port: 22},
		{Role: "worker", Host: "10.0.0.2", User: "ubuntu", Port: 2222},
	}
	err := p.Validate(context.Background(), specs)
	assert.NoError(t, err)
}

func TestValidateEmptyHost(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "", User: "root", Port: 22},
	}
	err := p.Validate(context.Background(), specs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host")
}

func TestValidateEmptyUser(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "worker", Host: "10.0.0.1", User: "", Port: 22},
	}
	err := p.Validate(context.Background(), specs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user")
}

func TestValidateZeroPort(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "10.0.0.1", User: "root", Port: 0},
	}
	err := p.Validate(context.Background(), specs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port")
}

func TestValidateNegativePort(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "10.0.0.1", User: "root", Port: -1},
	}
	err := p.Validate(context.Background(), specs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port")
}

func TestValidateEmptySpecs(t *testing.T) {
	p := New()
	err := p.Validate(context.Background(), []provider.MachineSpec{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one")
}

func TestValidateNilSpecs(t *testing.T) {
	p := New()
	err := p.Validate(context.Background(), nil)
	assert.Error(t, err)
}

func TestValidateMultipleSpecsOneInvalid(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "10.0.0.1", User: "root", Port: 22},
		{Role: "worker", Host: "", User: "ubuntu", Port: 22},
	}
	err := p.Validate(context.Background(), specs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "machine[1]")
}

func TestProvisionCreatesMachines(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "10.0.0.1", User: "root", Port: 22},
		{Role: "worker", Host: "10.0.0.2", User: "ubuntu", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	require.Len(t, machines, 2)

	assert.Equal(t, "controlplane", machines[0].Spec.Role)
	assert.Equal(t, "10.0.0.1", machines[0].Spec.Host)
	assert.Equal(t, "root", machines[0].Spec.User)
	assert.Equal(t, 22, machines[0].Spec.Port)

	assert.Equal(t, "worker", machines[1].Spec.Role)
	assert.Equal(t, "10.0.0.2", machines[1].Spec.Host)
}

func TestProvisionSetsStatusReady(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "10.0.0.1", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	assert.Equal(t, provider.MachineStatusReady, machines[0].Status)
}

func TestProvisionSetsCreatedAt(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "10.0.0.1", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	assert.False(t, machines[0].CreatedAt.IsZero())
}

func TestProvisionGeneratesDeterministicIDs(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "10.0.0.1", User: "root", Port: 22},
		{Role: "worker", Host: "10.0.0.2", User: "ubuntu", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	assert.Equal(t, "ssh-10.0.0.1-0", machines[0].ID)
	assert.Equal(t, "ssh-10.0.0.2-1", machines[1].ID)
}

func TestProvisionGeneratesUniqueIDs(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "10.0.0.1", User: "root", Port: 22},
		{Role: "worker", Host: "10.0.0.2", User: "ubuntu", Port: 22},
		{Role: "worker", Host: "10.0.0.3", User: "ubuntu", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)

	ids := make(map[string]bool)
	for _, m := range machines {
		assert.False(t, ids[m.ID], "duplicate ID: %s", m.ID)
		ids[m.ID] = true
	}
}

func TestProvisionPreservesLabels(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{
			Role: "worker", Host: "10.0.0.1", User: "root", Port: 22,
			Labels: map[string]string{"env": "prod", "zone": "us-east-1"},
		},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	assert.Equal(t, "prod", machines[0].Spec.Labels["env"])
	assert.Equal(t, "us-east-1", machines[0].Spec.Labels["zone"])
}

func TestProvisionPreservesTaints(t *testing.T) {
	p := New()
	specs := []provider.MachineSpec{
		{
			Role: "controlplane", Host: "10.0.0.1", User: "root", Port: 22,
			Taints: []provider.Taint{{Key: "node-role.kubernetes.io/master", Effect: "NoSchedule"}},
		},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	require.Len(t, machines[0].Spec.Taints, 1)
	assert.Equal(t, "node-role.kubernetes.io/master", machines[0].Spec.Taints[0].Key)
}

func TestProvisionEmptySpecs(t *testing.T) {
	p := New()
	machines, err := p.Provision(context.Background(), []provider.MachineSpec{})
	require.NoError(t, err)
	assert.Empty(t, machines)
}

func TestStatusReturnsMachinesUnchanged(t *testing.T) {
	p := New()
	machines := []provider.Machine{
		{ID: "ssh-10.0.0.1-0", Status: provider.MachineStatusReady, Spec: provider.MachineSpec{Host: "10.0.0.1"}},
		{ID: "ssh-10.0.0.2-1", Status: provider.MachineStatusReady, Spec: provider.MachineSpec{Host: "10.0.0.2"}},
	}

	statuses, err := p.Status(context.Background(), machines)
	require.NoError(t, err)
	require.Len(t, statuses, 2)
	assert.Equal(t, provider.MachineStatusReady, statuses[0])
	assert.Equal(t, provider.MachineStatusReady, statuses[1])
}

func TestStatusEmptyMachines(t *testing.T) {
	p := New()
	statuses, err := p.Status(context.Background(), []provider.Machine{})
	require.NoError(t, err)
	assert.Empty(t, statuses)
}

func TestDestroyReturnsNil(t *testing.T) {
	p := New()
	machines := []provider.Machine{
		{ID: "ssh-10.0.0.1-0", Spec: provider.MachineSpec{Host: "10.0.0.1"}},
	}

	err := p.Destroy(context.Background(), machines)
	assert.NoError(t, err)
}

func TestDestroyEmptyMachines(t *testing.T) {
	p := New()
	err := p.Destroy(context.Background(), []provider.Machine{})
	assert.NoError(t, err)
}
