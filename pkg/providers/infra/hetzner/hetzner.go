package hetzner

import (
	"context"
	"fmt"
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type HetznerClient interface {
	CreateServer(ctx context.Context, name string, opts ServerCreateOpts) (*ServerResult, error)
	GetServer(ctx context.Context, id string) (*ServerResult, error)
	DeleteServer(ctx context.Context, id string) error
}

type ServerCreateOpts struct {
	ServerType string
	Image      string
	Location   string
	SSHKeyName string
	Labels     map[string]string
}

type ServerResult struct {
	ID       string
	Name     string
	Status   string
	PublicIP string
}

type Config struct {
	Location   string
	ServerType string
	Image      string
	SSHKeyName string
}

type Provider struct {
	config Config
	client HetznerClient
}

func New(config Config, client HetznerClient) *Provider {
	return &Provider{
		config: config,
		client: client,
	}
}

func (p *Provider) Name() string {
	return "hetzner"
}

func (p *Provider) Validate(_ context.Context, machines []provider.MachineSpec) error {
	if len(machines) == 0 {
		return fmt.Errorf("at least one machine spec is required")
	}
	if p.client == nil {
		return fmt.Errorf("hetzner client is not configured")
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
		opts := ServerCreateOpts{
			ServerType: p.config.ServerType,
			Image:      p.config.Image,
			Location:   p.config.Location,
			SSHKeyName: p.config.SSHKeyName,
			Labels:     spec.Labels,
		}

		name := fmt.Sprintf("ironkube-%s-%d", spec.Role, i)
		result, err := p.client.CreateServer(ctx, name, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create server for machine[%d]: %w", i, err)
		}

		spec.Host = result.PublicIP

		machines = append(machines, provider.Machine{
			ID:         fmt.Sprintf("hetzner-%d", i),
			ProviderID: result.ID,
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
		result, err := p.client.GetServer(ctx, m.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("failed to get server %s: %w", m.ProviderID, err)
		}
		statuses[i] = mapHetznerStatus(result.Status)
	}

	return statuses, nil
}

func (p *Provider) Destroy(ctx context.Context, machines []provider.Machine) error {
	for _, m := range machines {
		if err := p.client.DeleteServer(ctx, m.ProviderID); err != nil {
			return fmt.Errorf("failed to delete server %s: %w", m.ProviderID, err)
		}
	}
	return nil
}

func mapHetznerStatus(status string) provider.MachineStatus {
	switch status {
	case "running":
		return provider.MachineStatusReady
	case "initializing", "starting", "rebuilding":
		return provider.MachineStatusProvisioning
	case "off", "deleting":
		return provider.MachineStatusDeleted
	default:
		return provider.MachineStatusProvisioning
	}
}
