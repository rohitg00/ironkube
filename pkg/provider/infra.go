package provider

import "context"

type InfraProvider interface {
	Name() string
	Validate(ctx context.Context, machines []MachineSpec) error
	Provision(ctx context.Context, machines []MachineSpec) ([]Machine, error)
	Status(ctx context.Context, machines []Machine) ([]MachineStatus, error)
	Destroy(ctx context.Context, machines []Machine) error
}
