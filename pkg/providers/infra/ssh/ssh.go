package ssh

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "ssh"
}

func (p *Provider) Validate(_ context.Context, machines []provider.MachineSpec) error {
	if len(machines) == 0 {
		return fmt.Errorf("at least one machine spec is required")
	}

	for i, m := range machines {
		if m.Host == "" {
			return fmt.Errorf("machine[%d]: host cannot be empty", i)
		}
		if m.User == "" {
			return fmt.Errorf("machine[%d]: user cannot be empty", i)
		}
		if m.Port <= 0 {
			return fmt.Errorf("machine[%d]: port must be greater than 0", i)
		}
	}

	return nil
}

func (p *Provider) Provision(_ context.Context, specs []provider.MachineSpec) ([]provider.Machine, error) {
	machines := make([]provider.Machine, len(specs))
	now := time.Now()

	for i, spec := range specs {
		machines[i] = provider.Machine{
			ID:        fmt.Sprintf("ssh-%s-%d", spec.Host, i),
			Spec:      spec,
			Status:    provider.MachineStatusReady,
			CreatedAt: now,
		}
	}

	return machines, nil
}

func (p *Provider) Status(_ context.Context, machines []provider.Machine) ([]provider.MachineStatus, error) {
	statuses := make([]provider.MachineStatus, len(machines))
	for i, m := range machines {
		statuses[i] = m.Status
	}
	return statuses, nil
}

func (p *Provider) Destroy(_ context.Context, machines []provider.Machine) error {
	if len(machines) > 0 {
		log.Printf("ssh provider: destroy is a no-op for bare-metal machines (count=%d)", len(machines))
	}
	return nil
}
