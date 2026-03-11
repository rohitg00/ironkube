package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type DockerClient interface {
	CreateContainer(ctx context.Context, name string, opts ContainerCreateOpts) (string, error)
	StartContainer(ctx context.Context, id string) error
	InspectContainer(ctx context.Context, id string) (*ContainerInfo, error)
	StopContainer(ctx context.Context, id string) error
	RemoveContainer(ctx context.Context, id string) error
}

type ContainerCreateOpts struct {
	Image   string
	Network string
	Labels  map[string]string
}

type ContainerInfo struct {
	ID     string
	Name   string
	Status string
	IP     string
}

type Config struct {
	Image   string
	Network string
}

type Provider struct {
	config Config
	client DockerClient
}

func New(config Config, client DockerClient) *Provider {
	return &Provider{
		config: config,
		client: client,
	}
}

func (p *Provider) Name() string {
	return "docker"
}

func (p *Provider) Validate(_ context.Context, machines []provider.MachineSpec) error {
	if len(machines) == 0 {
		return fmt.Errorf("at least one machine spec is required")
	}
	if p.config.Image == "" {
		return fmt.Errorf("docker image must be set")
	}
	if p.client == nil {
		return fmt.Errorf("docker client is not configured")
	}
	return nil
}

func (p *Provider) Provision(ctx context.Context, specs []provider.MachineSpec) ([]provider.Machine, error) {
	if len(specs) == 0 {
		return []provider.Machine{}, nil
	}

	machines := make([]provider.Machine, 0, len(specs))
	now := time.Now()

	for i, spec := range specs {
		opts := ContainerCreateOpts{
			Image:   p.config.Image,
			Network: p.config.Network,
			Labels:  spec.Labels,
		}

		name := fmt.Sprintf("ironkube-%s-%d", spec.Role, i)
		containerID, err := p.client.CreateContainer(ctx, name, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create container for machine[%d]: %w", i, err)
		}

		if err := p.client.StartContainer(ctx, containerID); err != nil {
			return nil, fmt.Errorf("failed to start container for machine[%d]: %w", i, err)
		}

		info, err := p.client.InspectContainer(ctx, containerID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect container for machine[%d]: %w", i, err)
		}

		spec.Host = info.IP

		machines = append(machines, provider.Machine{
			ID:         fmt.Sprintf("docker-%d", i),
			ProviderID: containerID,
			Spec:       spec,
			Status:     provider.MachineStatusReady,
			CreatedAt:  now,
		})
	}

	return machines, nil
}

func (p *Provider) Status(ctx context.Context, machines []provider.Machine) ([]provider.MachineStatus, error) {
	if len(machines) == 0 {
		return []provider.MachineStatus{}, nil
	}

	statuses := make([]provider.MachineStatus, len(machines))

	for i, m := range machines {
		info, err := p.client.InspectContainer(ctx, m.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect container %s: %w", m.ProviderID, err)
		}
		statuses[i] = mapDockerStatus(info.Status)
	}

	return statuses, nil
}

func (p *Provider) Destroy(ctx context.Context, machines []provider.Machine) error {
	for _, m := range machines {
		if err := p.client.StopContainer(ctx, m.ProviderID); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", m.ProviderID, err)
		}
		if err := p.client.RemoveContainer(ctx, m.ProviderID); err != nil {
			return fmt.Errorf("failed to remove container %s: %w", m.ProviderID, err)
		}
	}
	return nil
}

func mapDockerStatus(status string) provider.MachineStatus {
	switch status {
	case "running":
		return provider.MachineStatusReady
	case "exited", "dead", "removing":
		return provider.MachineStatusDeleted
	case "created", "restarting", "paused":
		return provider.MachineStatusProvisioning
	default:
		return provider.MachineStatusProvisioning
	}
}
