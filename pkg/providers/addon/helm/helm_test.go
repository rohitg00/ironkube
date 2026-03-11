package helm

import (
	"context"
	"fmt"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ provider.AddonProvider = (*Provider)(nil)

type mockHelmClient struct {
	installFunc   func(ctx context.Context, opts InstallOpts) error
	upgradeFunc   func(ctx context.Context, opts UpgradeOpts) error
	uninstallFunc func(ctx context.Context, opts UninstallOpts) error
	statusFunc    func(ctx context.Context, name, namespace string) (*ReleaseStatus, error)
	addRepoFunc   func(ctx context.Context, name, url string) error
}

func (m *mockHelmClient) Install(ctx context.Context, opts InstallOpts) error {
	if m.installFunc != nil {
		return m.installFunc(ctx, opts)
	}
	return nil
}

func (m *mockHelmClient) Upgrade(ctx context.Context, opts UpgradeOpts) error {
	if m.upgradeFunc != nil {
		return m.upgradeFunc(ctx, opts)
	}
	return nil
}

func (m *mockHelmClient) Uninstall(ctx context.Context, opts UninstallOpts) error {
	if m.uninstallFunc != nil {
		return m.uninstallFunc(ctx, opts)
	}
	return nil
}

func (m *mockHelmClient) Status(ctx context.Context, name, namespace string) (*ReleaseStatus, error) {
	if m.statusFunc != nil {
		return m.statusFunc(ctx, name, namespace)
	}
	return &ReleaseStatus{Name: name, Namespace: namespace, Status: "deployed", Version: "1.0.0"}, nil
}

func (m *mockHelmClient) AddRepo(ctx context.Context, name, url string) error {
	if m.addRepoFunc != nil {
		return m.addRepoFunc(ctx, name, url)
	}
	return nil
}

func TestInstallSucceeds(t *testing.T) {
	var captured InstallOpts
	client := &mockHelmClient{
		installFunc: func(_ context.Context, opts InstallOpts) error {
			captured = opts
			return nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Chart:     "nginx/nginx-ingress",
		Version:   "1.0.0",
		Namespace: "ingress",
		Values:    map[string]any{"replicas": 2},
	}

	err := p.Install(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.Equal(t, "nginx", captured.Name)
	assert.Equal(t, "nginx/nginx-ingress", captured.Chart)
	assert.Equal(t, "1.0.0", captured.Version)
	assert.Equal(t, "ingress", captured.Namespace)
	assert.Equal(t, 2, captured.Values["replicas"])
	assert.True(t, captured.CreateNS)
}

func TestInstallWithRepositoryAddsRepoFirst(t *testing.T) {
	calls := []string{}
	client := &mockHelmClient{
		addRepoFunc: func(_ context.Context, name, url string) error {
			calls = append(calls, fmt.Sprintf("addRepo:%s:%s", name, url))
			return nil
		},
		installFunc: func(_ context.Context, opts InstallOpts) error {
			calls = append(calls, fmt.Sprintf("install:%s", opts.Name))
			return nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:       "cert-manager",
		Repository: "https://charts.jetstack.io",
		Chart:      "cert-manager/cert-manager",
		Version:    "1.14.0",
		Namespace:  "cert-manager",
	}

	err := p.Install(context.Background(), nil, addon)
	require.NoError(t, err)
	require.Len(t, calls, 2)
	assert.Equal(t, "addRepo:cert-manager:https://charts.jetstack.io", calls[0])
	assert.Equal(t, "install:cert-manager", calls[1])
}

func TestInstallAddRepoError(t *testing.T) {
	client := &mockHelmClient{
		addRepoFunc: func(_ context.Context, _, _ string) error {
			return fmt.Errorf("network unreachable")
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:       "nginx",
		Repository: "https://charts.example.com",
		Chart:      "nginx/nginx",
		Version:    "1.0.0",
		Namespace:  "default",
	}

	err := p.Install(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network unreachable")
	assert.Contains(t, err.Error(), "failed to add helm repo")
}

func TestInstallPropagatesError(t *testing.T) {
	client := &mockHelmClient{
		installFunc: func(_ context.Context, _ InstallOpts) error {
			return fmt.Errorf("chart not found")
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "missing",
		Chart:     "nonexistent/chart",
		Version:   "1.0.0",
		Namespace: "default",
	}

	err := p.Install(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chart not found")
	assert.Contains(t, err.Error(), "failed to install helm chart")
}

func TestInstallSetsWaitFromWaitReady(t *testing.T) {
	var captured InstallOpts
	client := &mockHelmClient{
		installFunc: func(_ context.Context, opts InstallOpts) error {
			captured = opts
			return nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Chart:     "nginx/nginx",
		Version:   "1.0.0",
		Namespace: "default",
		WaitReady: true,
	}

	err := p.Install(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.True(t, captured.Wait)
}

func TestInstallWithoutWaitReady(t *testing.T) {
	var captured InstallOpts
	client := &mockHelmClient{
		installFunc: func(_ context.Context, opts InstallOpts) error {
			captured = opts
			return nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Chart:     "nginx/nginx",
		Version:   "1.0.0",
		Namespace: "default",
		WaitReady: false,
	}

	err := p.Install(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.False(t, captured.Wait)
}

func TestUpgradeSucceeds(t *testing.T) {
	var captured UpgradeOpts
	client := &mockHelmClient{
		upgradeFunc: func(_ context.Context, opts UpgradeOpts) error {
			captured = opts
			return nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Chart:     "nginx/nginx-ingress",
		Version:   "2.0.0",
		Namespace: "ingress",
		Values:    map[string]any{"replicas": 3},
		WaitReady: true,
	}

	err := p.Upgrade(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.Equal(t, "nginx", captured.Name)
	assert.Equal(t, "nginx/nginx-ingress", captured.Chart)
	assert.Equal(t, "2.0.0", captured.Version)
	assert.Equal(t, "ingress", captured.Namespace)
	assert.Equal(t, 3, captured.Values["replicas"])
	assert.True(t, captured.Wait)
}

func TestUpgradePropagatesError(t *testing.T) {
	client := &mockHelmClient{
		upgradeFunc: func(_ context.Context, _ UpgradeOpts) error {
			return fmt.Errorf("release not found")
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Chart:     "nginx/nginx",
		Version:   "2.0.0",
		Namespace: "default",
	}

	err := p.Upgrade(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "release not found")
	assert.Contains(t, err.Error(), "failed to upgrade helm chart")
}

func TestUninstallSucceeds(t *testing.T) {
	var captured UninstallOpts
	client := &mockHelmClient{
		uninstallFunc: func(_ context.Context, opts UninstallOpts) error {
			captured = opts
			return nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Namespace: "ingress",
	}

	err := p.Uninstall(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.Equal(t, "nginx", captured.Name)
	assert.Equal(t, "ingress", captured.Namespace)
}

func TestUninstallPropagatesError(t *testing.T) {
	client := &mockHelmClient{
		uninstallFunc: func(_ context.Context, _ UninstallOpts) error {
			return fmt.Errorf("release not found")
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Namespace: "default",
	}

	err := p.Uninstall(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "release not found")
	assert.Contains(t, err.Error(), "failed to uninstall helm release")
}

func TestStatusDeployedReturnsInstalledAndReady(t *testing.T) {
	client := &mockHelmClient{
		statusFunc: func(_ context.Context, name, ns string) (*ReleaseStatus, error) {
			return &ReleaseStatus{
				Name:      name,
				Namespace: ns,
				Version:   "1.0.0",
				Status:    "deployed",
			}, nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Namespace: "ingress",
	}

	status, err := p.Status(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.Equal(t, "nginx", status.Name)
	assert.True(t, status.Installed)
	assert.True(t, status.Ready)
	assert.Equal(t, "1.0.0", status.Version)
}

func TestStatusFailedReturnsInstalledNotReady(t *testing.T) {
	client := &mockHelmClient{
		statusFunc: func(_ context.Context, name, ns string) (*ReleaseStatus, error) {
			return &ReleaseStatus{
				Name:      name,
				Namespace: ns,
				Version:   "1.0.0",
				Status:    "failed",
			}, nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Namespace: "default",
	}

	status, err := p.Status(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.Equal(t, "nginx", status.Name)
	assert.True(t, status.Installed)
	assert.False(t, status.Ready)
}

func TestStatusPendingInstallReturnsInstalledNotReady(t *testing.T) {
	client := &mockHelmClient{
		statusFunc: func(_ context.Context, name, ns string) (*ReleaseStatus, error) {
			return &ReleaseStatus{
				Name:      name,
				Namespace: ns,
				Version:   "1.0.0",
				Status:    "pending-install",
			}, nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Namespace: "default",
	}

	status, err := p.Status(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.True(t, status.Installed)
	assert.False(t, status.Ready)
}

func TestStatusNotFoundReturnsNotInstalled(t *testing.T) {
	client := &mockHelmClient{
		statusFunc: func(_ context.Context, _, _ string) (*ReleaseStatus, error) {
			return nil, nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Namespace: "default",
	}

	status, err := p.Status(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.Equal(t, "nginx", status.Name)
	assert.False(t, status.Installed)
	assert.False(t, status.Ready)
}

func TestStatusErrorPropagation(t *testing.T) {
	client := &mockHelmClient{
		statusFunc: func(_ context.Context, _, _ string) (*ReleaseStatus, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Namespace: "default",
	}

	status, err := p.Status(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Nil(t, status)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestInstallWithoutRepository(t *testing.T) {
	addRepoCalled := false
	client := &mockHelmClient{
		addRepoFunc: func(_ context.Context, _, _ string) error {
			addRepoCalled = true
			return nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "local-chart",
		Chart:     "./charts/local",
		Version:   "1.0.0",
		Namespace: "default",
	}

	err := p.Install(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.False(t, addRepoCalled)
}

func TestInstallWithNilValues(t *testing.T) {
	var captured InstallOpts
	client := &mockHelmClient{
		installFunc: func(_ context.Context, opts InstallOpts) error {
			captured = opts
			return nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Chart:     "nginx/nginx",
		Version:   "1.0.0",
		Namespace: "default",
	}

	err := p.Install(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.Nil(t, captured.Values)
}

func TestStatusReturnsVersion(t *testing.T) {
	client := &mockHelmClient{
		statusFunc: func(_ context.Context, name, ns string) (*ReleaseStatus, error) {
			return &ReleaseStatus{
				Name:       name,
				Namespace:  ns,
				Version:    "3.2.1",
				Status:     "deployed",
				AppVersion: "1.25.0",
			}, nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Namespace: "ingress",
	}

	status, err := p.Status(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.Equal(t, "3.2.1", status.Version)
}

func TestUpgradeWithoutWaitReady(t *testing.T) {
	var captured UpgradeOpts
	client := &mockHelmClient{
		upgradeFunc: func(_ context.Context, opts UpgradeOpts) error {
			captured = opts
			return nil
		},
	}
	p := New(client)

	addon := provider.AddonSpec{
		Name:      "nginx",
		Chart:     "nginx/nginx",
		Version:   "2.0.0",
		Namespace: "default",
		WaitReady: false,
	}

	err := p.Upgrade(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.False(t, captured.Wait)
}
