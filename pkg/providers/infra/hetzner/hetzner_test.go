package hetzner

import (
	"context"
	"fmt"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ provider.InfraProvider = (*Provider)(nil)

type mockClient struct {
	createFunc func(ctx context.Context, name string, opts ServerCreateOpts) (*ServerResult, error)
	getFunc    func(ctx context.Context, id string) (*ServerResult, error)
	deleteFunc func(ctx context.Context, id string) error
}

func (m *mockClient) CreateServer(ctx context.Context, name string, opts ServerCreateOpts) (*ServerResult, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, opts)
	}
	return &ServerResult{ID: "srv-123", Name: name, Status: "running", PublicIP: "1.2.3.4"}, nil
}

func (m *mockClient) GetServer(ctx context.Context, id string) (*ServerResult, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return &ServerResult{ID: id, Name: "test", Status: "running", PublicIP: "1.2.3.4"}, nil
}

func (m *mockClient) DeleteServer(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestName(t *testing.T) {
	p := New(Config{}, &mockClient{})
	assert.Equal(t, "hetzner", p.Name())
}

func TestValidateValidSpecs(t *testing.T) {
	p := New(Config{}, &mockClient{})
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "auto", User: "root", Port: 22},
	}
	err := p.Validate(context.Background(), specs)
	assert.NoError(t, err)
}

func TestValidateEmptySpecs(t *testing.T) {
	p := New(Config{}, &mockClient{})
	err := p.Validate(context.Background(), []provider.MachineSpec{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one")
}

func TestValidateNilSpecs(t *testing.T) {
	p := New(Config{}, &mockClient{})
	err := p.Validate(context.Background(), nil)
	assert.Error(t, err)
}

func TestValidateNilClient(t *testing.T) {
	p := New(Config{}, nil)
	specs := []provider.MachineSpec{
		{Role: "controlplane", Host: "auto", User: "root", Port: 22},
	}
	err := p.Validate(context.Background(), specs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client")
}

func TestProvisionSingleMachine(t *testing.T) {
	client := &mockClient{
		createFunc: func(_ context.Context, name string, opts ServerCreateOpts) (*ServerResult, error) {
			return &ServerResult{
				ID:       "srv-100",
				Name:     name,
				Status:   "running",
				PublicIP: "203.0.113.10",
			}, nil
		},
	}
	p := New(Config{
		Location:   "fsn1",
		ServerType: "cx21",
		Image:      "ubuntu-22.04",
		SSHKeyName: "mykey",
	}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	require.Len(t, machines, 1)

	assert.Equal(t, "srv-100", machines[0].ProviderID)
	assert.Equal(t, "203.0.113.10", machines[0].Spec.Host)
	assert.Equal(t, provider.MachineStatusReady, machines[0].Status)
	assert.False(t, machines[0].CreatedAt.IsZero())
}

func TestProvisionMultipleMachines(t *testing.T) {
	callCount := 0
	client := &mockClient{
		createFunc: func(_ context.Context, name string, _ ServerCreateOpts) (*ServerResult, error) {
			callCount++
			return &ServerResult{
				ID:       fmt.Sprintf("srv-%d", callCount),
				Name:     name,
				Status:   "running",
				PublicIP: fmt.Sprintf("203.0.113.%d", callCount),
			}, nil
		},
	}
	p := New(Config{Location: "nbg1", ServerType: "cx21", Image: "ubuntu-22.04"}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
		{Role: "worker", User: "root", Port: 22},
		{Role: "worker", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	require.Len(t, machines, 3)

	assert.Equal(t, 3, callCount)
	assert.Equal(t, "srv-1", machines[0].ProviderID)
	assert.Equal(t, "srv-2", machines[1].ProviderID)
	assert.Equal(t, "srv-3", machines[2].ProviderID)
}

func TestProvisionCreateServerError(t *testing.T) {
	client := &mockClient{
		createFunc: func(_ context.Context, _ string, _ ServerCreateOpts) (*ServerResult, error) {
			return nil, fmt.Errorf("api error: rate limit exceeded")
		},
	}
	p := New(Config{Location: "fsn1", ServerType: "cx21", Image: "ubuntu-22.04"}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	assert.Error(t, err)
	assert.Nil(t, machines)
	assert.Contains(t, err.Error(), "rate limit")
}

func TestProvisionPartialFailure(t *testing.T) {
	callCount := 0
	client := &mockClient{
		createFunc: func(_ context.Context, _ string, _ ServerCreateOpts) (*ServerResult, error) {
			callCount++
			if callCount == 2 {
				return nil, fmt.Errorf("api error: out of capacity")
			}
			return &ServerResult{ID: fmt.Sprintf("srv-%d", callCount), Name: "test", Status: "running", PublicIP: "1.2.3.4"}, nil
		},
	}
	p := New(Config{Location: "fsn1", ServerType: "cx21", Image: "ubuntu-22.04"}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
		{Role: "worker", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	assert.Error(t, err)
	assert.Nil(t, machines)
	assert.Contains(t, err.Error(), "out of capacity")
}

func TestProvisionPassesConfigToClient(t *testing.T) {
	var capturedOpts ServerCreateOpts
	client := &mockClient{
		createFunc: func(_ context.Context, _ string, opts ServerCreateOpts) (*ServerResult, error) {
			capturedOpts = opts
			return &ServerResult{ID: "srv-1", Name: "test", Status: "running", PublicIP: "1.2.3.4"}, nil
		},
	}
	p := New(Config{
		Location:   "hel1",
		ServerType: "cpx31",
		Image:      "debian-12",
		SSHKeyName: "deploy-key",
	}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22, Labels: map[string]string{"env": "prod"}},
	}

	_, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)

	assert.Equal(t, "hel1", capturedOpts.Location)
	assert.Equal(t, "cpx31", capturedOpts.ServerType)
	assert.Equal(t, "debian-12", capturedOpts.Image)
	assert.Equal(t, "deploy-key", capturedOpts.SSHKeyName)
	assert.Equal(t, "prod", capturedOpts.Labels["env"])
}

func TestProvisionSetsUserAndPort(t *testing.T) {
	client := &mockClient{
		createFunc: func(_ context.Context, _ string, _ ServerCreateOpts) (*ServerResult, error) {
			return &ServerResult{ID: "srv-1", Name: "test", Status: "running", PublicIP: "5.6.7.8"}, nil
		},
	}
	p := New(Config{Location: "fsn1", ServerType: "cx21", Image: "ubuntu-22.04"}, client)

	specs := []provider.MachineSpec{
		{Role: "worker", User: "deploy", Port: 2222},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	assert.Equal(t, "deploy", machines[0].Spec.User)
	assert.Equal(t, 2222, machines[0].Spec.Port)
}

func TestStatusRunning(t *testing.T) {
	client := &mockClient{
		getFunc: func(_ context.Context, id string) (*ServerResult, error) {
			return &ServerResult{ID: id, Name: "test", Status: "running", PublicIP: "1.2.3.4"}, nil
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "hetzner-0", ProviderID: "srv-100", Status: provider.MachineStatusProvisioning},
	}

	statuses, err := p.Status(context.Background(), machines)
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	assert.Equal(t, provider.MachineStatusReady, statuses[0])
}

func TestStatusInitializing(t *testing.T) {
	client := &mockClient{
		getFunc: func(_ context.Context, id string) (*ServerResult, error) {
			return &ServerResult{ID: id, Name: "test", Status: "initializing", PublicIP: ""}, nil
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "hetzner-0", ProviderID: "srv-100", Status: provider.MachineStatusProvisioning},
	}

	statuses, err := p.Status(context.Background(), machines)
	require.NoError(t, err)
	assert.Equal(t, provider.MachineStatusProvisioning, statuses[0])
}

func TestStatusOff(t *testing.T) {
	client := &mockClient{
		getFunc: func(_ context.Context, id string) (*ServerResult, error) {
			return &ServerResult{ID: id, Name: "test", Status: "off", PublicIP: "1.2.3.4"}, nil
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "hetzner-0", ProviderID: "srv-100", Status: provider.MachineStatusReady},
	}

	statuses, err := p.Status(context.Background(), machines)
	require.NoError(t, err)
	assert.Equal(t, provider.MachineStatusDeleted, statuses[0])
}

func TestStatusGetServerError(t *testing.T) {
	client := &mockClient{
		getFunc: func(_ context.Context, _ string) (*ServerResult, error) {
			return nil, fmt.Errorf("server not found")
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "hetzner-0", ProviderID: "srv-100"},
	}

	statuses, err := p.Status(context.Background(), machines)
	assert.Error(t, err)
	assert.Nil(t, statuses)
}

func TestStatusMultipleMachines(t *testing.T) {
	client := &mockClient{
		getFunc: func(_ context.Context, id string) (*ServerResult, error) {
			status := "running"
			if id == "srv-200" {
				status = "initializing"
			}
			return &ServerResult{ID: id, Name: "test", Status: status, PublicIP: "1.2.3.4"}, nil
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "hetzner-0", ProviderID: "srv-100"},
		{ID: "hetzner-1", ProviderID: "srv-200"},
	}

	statuses, err := p.Status(context.Background(), machines)
	require.NoError(t, err)
	assert.Equal(t, provider.MachineStatusReady, statuses[0])
	assert.Equal(t, provider.MachineStatusProvisioning, statuses[1])
}

func TestDestroyCallsDeleteForEach(t *testing.T) {
	deleted := make(map[string]bool)
	client := &mockClient{
		deleteFunc: func(_ context.Context, id string) error {
			deleted[id] = true
			return nil
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "hetzner-0", ProviderID: "srv-100"},
		{ID: "hetzner-1", ProviderID: "srv-200"},
	}

	err := p.Destroy(context.Background(), machines)
	require.NoError(t, err)
	assert.True(t, deleted["srv-100"])
	assert.True(t, deleted["srv-200"])
	assert.Len(t, deleted, 2)
}

func TestDestroyDeleteError(t *testing.T) {
	client := &mockClient{
		deleteFunc: func(_ context.Context, _ string) error {
			return fmt.Errorf("permission denied")
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "hetzner-0", ProviderID: "srv-100"},
	}

	err := p.Destroy(context.Background(), machines)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestDestroyEmptyMachines(t *testing.T) {
	p := New(Config{}, &mockClient{})
	err := p.Destroy(context.Background(), []provider.Machine{})
	assert.NoError(t, err)
}

func TestProvisionEmptySpecs(t *testing.T) {
	p := New(Config{}, &mockClient{})
	machines, err := p.Provision(context.Background(), []provider.MachineSpec{})
	require.NoError(t, err)
	assert.Empty(t, machines)
}

func TestProvisionGeneratesDeterministicIDs(t *testing.T) {
	client := &mockClient{
		createFunc: func(_ context.Context, _ string, _ ServerCreateOpts) (*ServerResult, error) {
			return &ServerResult{ID: "srv-42", Name: "test", Status: "running", PublicIP: "1.2.3.4"}, nil
		},
	}
	p := New(Config{Location: "fsn1", ServerType: "cx21", Image: "ubuntu-22.04"}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	assert.Equal(t, "hetzner-0", machines[0].ID)
}

func TestStatusEmptyMachines(t *testing.T) {
	p := New(Config{}, &mockClient{})
	statuses, err := p.Status(context.Background(), []provider.Machine{})
	require.NoError(t, err)
	assert.Empty(t, statuses)
}
