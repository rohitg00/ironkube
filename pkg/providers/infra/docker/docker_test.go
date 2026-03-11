package docker

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
	createFunc  func(ctx context.Context, name string, opts ContainerCreateOpts) (string, error)
	startFunc   func(ctx context.Context, id string) error
	inspectFunc func(ctx context.Context, id string) (*ContainerInfo, error)
	stopFunc    func(ctx context.Context, id string) error
	removeFunc  func(ctx context.Context, id string) error
}

func (m *mockClient) CreateContainer(ctx context.Context, name string, opts ContainerCreateOpts) (string, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, opts)
	}
	return "abc123", nil
}

func (m *mockClient) StartContainer(ctx context.Context, id string) error {
	if m.startFunc != nil {
		return m.startFunc(ctx, id)
	}
	return nil
}

func (m *mockClient) InspectContainer(ctx context.Context, id string) (*ContainerInfo, error) {
	if m.inspectFunc != nil {
		return m.inspectFunc(ctx, id)
	}
	return &ContainerInfo{ID: id, Name: "test", Status: "running", IP: "172.17.0.2"}, nil
}

func (m *mockClient) StopContainer(ctx context.Context, id string) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx, id)
	}
	return nil
}

func (m *mockClient) RemoveContainer(ctx context.Context, id string) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, id)
	}
	return nil
}

func TestName(t *testing.T) {
	p := New(Config{Image: "kindest/node:v1.28.0"}, &mockClient{})
	assert.Equal(t, "docker", p.Name())
}

func TestValidateValidSpecs(t *testing.T) {
	p := New(Config{Image: "kindest/node:v1.28.0"}, &mockClient{})
	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
	}
	err := p.Validate(context.Background(), specs)
	assert.NoError(t, err)
}

func TestValidateEmptySpecs(t *testing.T) {
	p := New(Config{Image: "kindest/node:v1.28.0"}, &mockClient{})
	err := p.Validate(context.Background(), []provider.MachineSpec{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one")
}

func TestValidateNilSpecs(t *testing.T) {
	p := New(Config{Image: "kindest/node:v1.28.0"}, &mockClient{})
	err := p.Validate(context.Background(), nil)
	assert.Error(t, err)
}

func TestValidateEmptyImage(t *testing.T) {
	p := New(Config{Image: ""}, &mockClient{})
	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
	}
	err := p.Validate(context.Background(), specs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image")
}

func TestValidateNilClient(t *testing.T) {
	p := New(Config{Image: "kindest/node:v1.28.0"}, nil)
	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
	}
	err := p.Validate(context.Background(), specs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client")
}

func TestProvisionSingleMachine(t *testing.T) {
	client := &mockClient{
		createFunc: func(_ context.Context, _ string, _ ContainerCreateOpts) (string, error) {
			return "container-abc", nil
		},
		inspectFunc: func(_ context.Context, id string) (*ContainerInfo, error) {
			return &ContainerInfo{ID: id, Name: "test", Status: "running", IP: "172.17.0.2"}, nil
		},
	}
	p := New(Config{Image: "kindest/node:v1.28.0", Network: "ironkube-net"}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	require.Len(t, machines, 1)

	assert.Equal(t, "docker-0", machines[0].ID)
	assert.Equal(t, "container-abc", machines[0].ProviderID)
	assert.Equal(t, "172.17.0.2", machines[0].Spec.Host)
	assert.Equal(t, provider.MachineStatusReady, machines[0].Status)
	assert.False(t, machines[0].CreatedAt.IsZero())
}

func TestProvisionMultipleMachines(t *testing.T) {
	callCount := 0
	client := &mockClient{
		createFunc: func(_ context.Context, _ string, _ ContainerCreateOpts) (string, error) {
			callCount++
			return fmt.Sprintf("container-%d", callCount), nil
		},
		inspectFunc: func(_ context.Context, id string) (*ContainerInfo, error) {
			return &ContainerInfo{ID: id, Name: "test", Status: "running", IP: fmt.Sprintf("172.17.0.%d", callCount+1)}, nil
		},
	}
	p := New(Config{Image: "kindest/node:v1.28.0"}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
		{Role: "worker", User: "root", Port: 22},
		{Role: "worker", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	require.Len(t, machines, 3)
	assert.Equal(t, "container-1", machines[0].ProviderID)
	assert.Equal(t, "container-2", machines[1].ProviderID)
	assert.Equal(t, "container-3", machines[2].ProviderID)
}

func TestProvisionCreateError(t *testing.T) {
	client := &mockClient{
		createFunc: func(_ context.Context, _ string, _ ContainerCreateOpts) (string, error) {
			return "", fmt.Errorf("image not found")
		},
	}
	p := New(Config{Image: "invalid:latest"}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	assert.Error(t, err)
	assert.Nil(t, machines)
	assert.Contains(t, err.Error(), "image not found")
}

func TestProvisionStartError(t *testing.T) {
	client := &mockClient{
		startFunc: func(_ context.Context, _ string) error {
			return fmt.Errorf("port already in use")
		},
	}
	p := New(Config{Image: "kindest/node:v1.28.0"}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	assert.Error(t, err)
	assert.Nil(t, machines)
	assert.Contains(t, err.Error(), "port already in use")
}

func TestProvisionInspectError(t *testing.T) {
	client := &mockClient{
		inspectFunc: func(_ context.Context, _ string) (*ContainerInfo, error) {
			return nil, fmt.Errorf("container disappeared")
		},
	}
	p := New(Config{Image: "kindest/node:v1.28.0"}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	assert.Error(t, err)
	assert.Nil(t, machines)
}

func TestProvisionPassesConfigToClient(t *testing.T) {
	var capturedOpts ContainerCreateOpts
	client := &mockClient{
		createFunc: func(_ context.Context, _ string, opts ContainerCreateOpts) (string, error) {
			capturedOpts = opts
			return "c-1", nil
		},
		inspectFunc: func(_ context.Context, id string) (*ContainerInfo, error) {
			return &ContainerInfo{ID: id, Name: "test", Status: "running", IP: "172.17.0.2"}, nil
		},
	}
	p := New(Config{Image: "kindest/node:v1.28.0", Network: "custom-net"}, client)

	specs := []provider.MachineSpec{
		{Role: "worker", User: "root", Port: 22, Labels: map[string]string{"tier": "backend"}},
	}

	_, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)

	assert.Equal(t, "kindest/node:v1.28.0", capturedOpts.Image)
	assert.Equal(t, "custom-net", capturedOpts.Network)
	assert.Equal(t, "backend", capturedOpts.Labels["tier"])
}

func TestProvisionEmptySpecs(t *testing.T) {
	p := New(Config{Image: "kindest/node:v1.28.0"}, &mockClient{})
	machines, err := p.Provision(context.Background(), []provider.MachineSpec{})
	require.NoError(t, err)
	assert.Empty(t, machines)
}

func TestStatusRunning(t *testing.T) {
	client := &mockClient{
		inspectFunc: func(_ context.Context, id string) (*ContainerInfo, error) {
			return &ContainerInfo{ID: id, Name: "test", Status: "running", IP: "172.17.0.2"}, nil
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "docker-0", ProviderID: "container-abc", Status: provider.MachineStatusProvisioning},
	}

	statuses, err := p.Status(context.Background(), machines)
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	assert.Equal(t, provider.MachineStatusReady, statuses[0])
}

func TestStatusExited(t *testing.T) {
	client := &mockClient{
		inspectFunc: func(_ context.Context, id string) (*ContainerInfo, error) {
			return &ContainerInfo{ID: id, Name: "test", Status: "exited", IP: ""}, nil
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "docker-0", ProviderID: "container-abc", Status: provider.MachineStatusReady},
	}

	statuses, err := p.Status(context.Background(), machines)
	require.NoError(t, err)
	assert.Equal(t, provider.MachineStatusDeleted, statuses[0])
}

func TestStatusCreated(t *testing.T) {
	client := &mockClient{
		inspectFunc: func(_ context.Context, id string) (*ContainerInfo, error) {
			return &ContainerInfo{ID: id, Name: "test", Status: "created", IP: ""}, nil
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "docker-0", ProviderID: "container-abc"},
	}

	statuses, err := p.Status(context.Background(), machines)
	require.NoError(t, err)
	assert.Equal(t, provider.MachineStatusProvisioning, statuses[0])
}

func TestStatusInspectError(t *testing.T) {
	client := &mockClient{
		inspectFunc: func(_ context.Context, _ string) (*ContainerInfo, error) {
			return nil, fmt.Errorf("no such container")
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "docker-0", ProviderID: "container-abc"},
	}

	statuses, err := p.Status(context.Background(), machines)
	assert.Error(t, err)
	assert.Nil(t, statuses)
}

func TestStatusMultipleMachines(t *testing.T) {
	client := &mockClient{
		inspectFunc: func(_ context.Context, id string) (*ContainerInfo, error) {
			status := "running"
			if id == "container-2" {
				status = "exited"
			}
			return &ContainerInfo{ID: id, Name: "test", Status: status, IP: "172.17.0.2"}, nil
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "docker-0", ProviderID: "container-1"},
		{ID: "docker-1", ProviderID: "container-2"},
	}

	statuses, err := p.Status(context.Background(), machines)
	require.NoError(t, err)
	assert.Equal(t, provider.MachineStatusReady, statuses[0])
	assert.Equal(t, provider.MachineStatusDeleted, statuses[1])
}

func TestStatusEmptyMachines(t *testing.T) {
	p := New(Config{}, &mockClient{})
	statuses, err := p.Status(context.Background(), []provider.Machine{})
	require.NoError(t, err)
	assert.Empty(t, statuses)
}

func TestDestroyStopsAndRemoves(t *testing.T) {
	stopped := make(map[string]bool)
	removed := make(map[string]bool)
	client := &mockClient{
		stopFunc: func(_ context.Context, id string) error {
			stopped[id] = true
			return nil
		},
		removeFunc: func(_ context.Context, id string) error {
			removed[id] = true
			return nil
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "docker-0", ProviderID: "container-1"},
		{ID: "docker-1", ProviderID: "container-2"},
	}

	err := p.Destroy(context.Background(), machines)
	require.NoError(t, err)
	assert.True(t, stopped["container-1"])
	assert.True(t, stopped["container-2"])
	assert.True(t, removed["container-1"])
	assert.True(t, removed["container-2"])
}

func TestDestroyStopError(t *testing.T) {
	client := &mockClient{
		stopFunc: func(_ context.Context, _ string) error {
			return fmt.Errorf("container not running")
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "docker-0", ProviderID: "container-1"},
	}

	err := p.Destroy(context.Background(), machines)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container not running")
}

func TestDestroyRemoveError(t *testing.T) {
	client := &mockClient{
		removeFunc: func(_ context.Context, _ string) error {
			return fmt.Errorf("device busy")
		},
	}
	p := New(Config{}, client)

	machines := []provider.Machine{
		{ID: "docker-0", ProviderID: "container-1"},
	}

	err := p.Destroy(context.Background(), machines)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "device busy")
}

func TestDestroyEmptyMachines(t *testing.T) {
	p := New(Config{}, &mockClient{})
	err := p.Destroy(context.Background(), []provider.Machine{})
	assert.NoError(t, err)
}

func TestProvisionPreservesRole(t *testing.T) {
	client := &mockClient{
		inspectFunc: func(_ context.Context, id string) (*ContainerInfo, error) {
			return &ContainerInfo{ID: id, Name: "test", Status: "running", IP: "172.17.0.2"}, nil
		},
	}
	p := New(Config{Image: "kindest/node:v1.28.0"}, client)

	specs := []provider.MachineSpec{
		{Role: "controlplane", User: "root", Port: 22},
		{Role: "worker", User: "root", Port: 22},
	}

	machines, err := p.Provision(context.Background(), specs)
	require.NoError(t, err)
	assert.Equal(t, "controlplane", machines[0].Spec.Role)
	assert.Equal(t, "worker", machines[1].Spec.Role)
}
