package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockInfraProvider struct {
	name     string
	machines []Machine
}

func (m *mockInfraProvider) Name() string {
	return m.name
}

func (m *mockInfraProvider) Validate(_ context.Context, machines []MachineSpec) error {
	for _, spec := range machines {
		if spec.Host == "" {
			return fmt.Errorf("machine missing host")
		}
		if spec.User == "" {
			return fmt.Errorf("machine missing user")
		}
	}
	return nil
}

func (m *mockInfraProvider) Provision(_ context.Context, machines []MachineSpec) ([]Machine, error) {
	var result []Machine
	for i, spec := range machines {
		result = append(result, Machine{
			ID:        fmt.Sprintf("m-%03d", i+1),
			Spec:      spec,
			Status:    MachineStatusProvisioning,
			CreatedAt: time.Now(),
		})
	}
	m.machines = result
	return result, nil
}

func (m *mockInfraProvider) Status(_ context.Context, machines []Machine) ([]MachineStatus, error) {
	statuses := make([]MachineStatus, len(machines))
	for i := range machines {
		statuses[i] = MachineStatusReady
	}
	return statuses, nil
}

func (m *mockInfraProvider) Destroy(_ context.Context, machines []Machine) error {
	m.machines = nil
	return nil
}

func TestInfraProviderName(t *testing.T) {
	var p InfraProvider = &mockInfraProvider{name: "bare-metal"}
	assert.Equal(t, "bare-metal", p.Name())
}

func TestInfraProviderValidateSuccess(t *testing.T) {
	var p InfraProvider = &mockInfraProvider{name: "ssh"}
	specs := []MachineSpec{
		{Role: "control-plane", Host: "10.0.0.1", User: "ubuntu", Port: 22},
		{Role: "worker", Host: "10.0.0.2", User: "ubuntu", Port: 22},
	}
	err := p.Validate(context.Background(), specs)
	assert.NoError(t, err)
}

func TestInfraProviderValidateError(t *testing.T) {
	var p InfraProvider = &mockInfraProvider{name: "ssh"}
	specs := []MachineSpec{
		{Role: "worker", Host: "", User: "ubuntu"},
	}
	err := p.Validate(context.Background(), specs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing host")
}

func TestInfraProviderProvision(t *testing.T) {
	var p InfraProvider = &mockInfraProvider{name: "ssh"}
	specs := []MachineSpec{
		{Role: "control-plane", Host: "10.0.0.1", User: "ubuntu", Port: 22},
		{Role: "worker", Host: "10.0.0.2", User: "ubuntu", Port: 22},
		{Role: "worker", Host: "10.0.0.3", User: "ubuntu", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	require.Len(t, machines, 3)
	assert.Equal(t, "m-001", machines[0].ID)
	assert.Equal(t, "m-002", machines[1].ID)
	assert.Equal(t, "m-003", machines[2].ID)
	assert.Equal(t, MachineStatusProvisioning, machines[0].Status)
	assert.Equal(t, "control-plane", machines[0].Spec.Role)
	assert.Equal(t, "10.0.0.2", machines[1].Spec.Host)
}

func TestInfraProviderStatus(t *testing.T) {
	var p InfraProvider = &mockInfraProvider{name: "ssh"}
	machines := []Machine{
		{ID: "m-001", Status: MachineStatusProvisioning},
		{ID: "m-002", Status: MachineStatusProvisioning},
	}

	statuses, err := p.Status(context.Background(), machines)
	require.NoError(t, err)
	require.Len(t, statuses, 2)
	assert.Equal(t, MachineStatusReady, statuses[0])
	assert.Equal(t, MachineStatusReady, statuses[1])
}

func TestInfraProviderDestroy(t *testing.T) {
	mock := &mockInfraProvider{name: "ssh"}
	var p InfraProvider = mock

	specs := []MachineSpec{
		{Role: "worker", Host: "10.0.0.1", User: "ubuntu", Port: 22},
	}
	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	require.Len(t, mock.machines, 1)

	err = p.Destroy(context.Background(), machines)
	assert.NoError(t, err)
	assert.Empty(t, mock.machines)
}

func TestInfraProviderEmptyMachines(t *testing.T) {
	var p InfraProvider = &mockInfraProvider{name: "ssh"}

	err := p.Validate(context.Background(), nil)
	assert.NoError(t, err)

	machines, err := p.Provision(context.Background(), nil)
	assert.NoError(t, err)
	assert.Empty(t, machines)

	statuses, err := p.Status(context.Background(), nil)
	assert.NoError(t, err)
	assert.Empty(t, statuses)

	err = p.Destroy(context.Background(), nil)
	assert.NoError(t, err)
}
