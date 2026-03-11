package provider

import "context"

type AddonProvider interface {
	Install(ctx context.Context, kubeconfig []byte, addon AddonSpec) error
	Upgrade(ctx context.Context, kubeconfig []byte, addon AddonSpec) error
	Uninstall(ctx context.Context, kubeconfig []byte, addon AddonSpec) error
	Status(ctx context.Context, kubeconfig []byte, addon AddonSpec) (*AddonStatus, error)
}
