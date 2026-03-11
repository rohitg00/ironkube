package helm

import (
	"context"
	"fmt"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type HelmClient interface {
	Install(ctx context.Context, opts InstallOpts) error
	Upgrade(ctx context.Context, opts UpgradeOpts) error
	Uninstall(ctx context.Context, opts UninstallOpts) error
	Status(ctx context.Context, name, namespace string) (*ReleaseStatus, error)
	AddRepo(ctx context.Context, name, url string) error
}

type InstallOpts struct {
	Name      string
	Chart     string
	Version   string
	Namespace string
	Values    map[string]any
	Wait      bool
	CreateNS  bool
}

type UpgradeOpts struct {
	Name      string
	Chart     string
	Version   string
	Namespace string
	Values    map[string]any
	Wait      bool
}

type UninstallOpts struct {
	Name      string
	Namespace string
}

type ReleaseStatus struct {
	Name       string
	Namespace  string
	Version    string
	Status     string
	AppVersion string
}

type Provider struct {
	client HelmClient
}

var _ provider.AddonProvider = (*Provider)(nil)

func New(client HelmClient) *Provider {
	return &Provider{client: client}
}

func (p *Provider) Install(ctx context.Context, _ []byte, addon provider.AddonSpec) error {
	if addon.Repository != "" {
		if err := p.client.AddRepo(ctx, addon.Name, addon.Repository); err != nil {
			return fmt.Errorf("failed to add helm repo %s: %w", addon.Repository, err)
		}
	}

	opts := InstallOpts{
		Name:      addon.Name,
		Chart:     addon.Chart,
		Version:   addon.Version,
		Namespace: addon.Namespace,
		Values:    addon.Values,
		Wait:      addon.WaitReady,
		CreateNS:  true,
	}

	if err := p.client.Install(ctx, opts); err != nil {
		return fmt.Errorf("failed to install helm chart %s: %w", addon.Chart, err)
	}

	return nil
}

func (p *Provider) Upgrade(ctx context.Context, _ []byte, addon provider.AddonSpec) error {
	opts := UpgradeOpts{
		Name:      addon.Name,
		Chart:     addon.Chart,
		Version:   addon.Version,
		Namespace: addon.Namespace,
		Values:    addon.Values,
		Wait:      addon.WaitReady,
	}

	if err := p.client.Upgrade(ctx, opts); err != nil {
		return fmt.Errorf("failed to upgrade helm chart %s: %w", addon.Chart, err)
	}

	return nil
}

func (p *Provider) Uninstall(ctx context.Context, _ []byte, addon provider.AddonSpec) error {
	opts := UninstallOpts{
		Name:      addon.Name,
		Namespace: addon.Namespace,
	}

	if err := p.client.Uninstall(ctx, opts); err != nil {
		return fmt.Errorf("failed to uninstall helm release %s: %w", addon.Name, err)
	}

	return nil
}

func (p *Provider) Status(ctx context.Context, _ []byte, addon provider.AddonSpec) (*provider.AddonStatus, error) {
	release, err := p.client.Status(ctx, addon.Name, addon.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get helm release status for %s: %w", addon.Name, err)
	}

	if release == nil {
		return &provider.AddonStatus{
			Name:      addon.Name,
			Installed: false,
		}, nil
	}

	status := &provider.AddonStatus{
		Name:      addon.Name,
		Installed: true,
		Version:   release.Version,
	}

	switch release.Status {
	case "deployed":
		status.Ready = true
	default:
		status.Ready = false
	}

	return status, nil
}
