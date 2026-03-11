package manifest

import (
	"context"
	"fmt"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type ManifestApplier interface {
	Apply(ctx context.Context, kubeconfig []byte, namespace string, manifests []byte) error
	Delete(ctx context.Context, kubeconfig []byte, namespace string, manifests []byte) error
}

type ManifestFetcher interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}

type Provider struct {
	applier ManifestApplier
	fetcher ManifestFetcher
}

var _ provider.AddonProvider = (*Provider)(nil)

func New(applier ManifestApplier, fetcher ManifestFetcher) *Provider {
	return &Provider{
		applier: applier,
		fetcher: fetcher,
	}
}

func (p *Provider) Install(ctx context.Context, kubeconfig []byte, addon provider.AddonSpec) error {
	if addon.ManifestURL == "" {
		return fmt.Errorf("manifest URL is required for addon %s", addon.Name)
	}

	manifests, err := p.fetcher.Fetch(ctx, addon.ManifestURL)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest from %s: %w", addon.ManifestURL, err)
	}

	if err := p.applier.Apply(ctx, kubeconfig, addon.Namespace, manifests); err != nil {
		return fmt.Errorf("failed to apply manifest for addon %s: %w", addon.Name, err)
	}

	return nil
}

func (p *Provider) Upgrade(ctx context.Context, kubeconfig []byte, addon provider.AddonSpec) error {
	return p.Install(ctx, kubeconfig, addon)
}

func (p *Provider) Uninstall(ctx context.Context, kubeconfig []byte, addon provider.AddonSpec) error {
	if addon.ManifestURL == "" {
		return fmt.Errorf("manifest URL is required for addon %s", addon.Name)
	}

	manifests, err := p.fetcher.Fetch(ctx, addon.ManifestURL)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest from %s: %w", addon.ManifestURL, err)
	}

	if err := p.applier.Delete(ctx, kubeconfig, addon.Namespace, manifests); err != nil {
		return fmt.Errorf("failed to delete manifest for addon %s: %w", addon.Name, err)
	}

	return nil
}

func (p *Provider) Status(_ context.Context, _ []byte, addon provider.AddonSpec) (*provider.AddonStatus, error) {
	return &provider.AddonStatus{
		Name:      addon.Name,
		Installed: true,
		Version:   addon.Version,
		Ready:     true,
	}, nil
}
