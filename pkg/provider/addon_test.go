package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAddonProvider struct {
	installed map[string]AddonStatus
}

func newMockAddonProvider() *mockAddonProvider {
	return &mockAddonProvider{
		installed: make(map[string]AddonStatus),
	}
}

func (m *mockAddonProvider) Install(_ context.Context, _ []byte, addon AddonSpec) error {
	if addon.Name == "" {
		return fmt.Errorf("addon name required")
	}
	m.installed[addon.Name] = AddonStatus{
		Name:      addon.Name,
		Installed: true,
		Version:   addon.Version,
		Ready:     true,
	}
	return nil
}

func (m *mockAddonProvider) Upgrade(_ context.Context, _ []byte, addon AddonSpec) error {
	status, exists := m.installed[addon.Name]
	if !exists {
		return fmt.Errorf("addon %s not installed", addon.Name)
	}
	status.Version = addon.Version
	m.installed[addon.Name] = status
	return nil
}

func (m *mockAddonProvider) Uninstall(_ context.Context, _ []byte, addon AddonSpec) error {
	if _, exists := m.installed[addon.Name]; !exists {
		return fmt.Errorf("addon %s not installed", addon.Name)
	}
	delete(m.installed, addon.Name)
	return nil
}

func (m *mockAddonProvider) Status(_ context.Context, _ []byte, addon AddonSpec) (*AddonStatus, error) {
	status, exists := m.installed[addon.Name]
	if !exists {
		return &AddonStatus{Name: addon.Name, Installed: false}, nil
	}
	return &status, nil
}

func TestAddonProviderInstall(t *testing.T) {
	var p AddonProvider = newMockAddonProvider()

	addon := AddonSpec{
		Name:       "cilium",
		Type:       "helm",
		Repository: "https://helm.cilium.io",
		Chart:      "cilium",
		Version:    "1.15.0",
		Namespace:  "kube-system",
		WaitReady:  true,
	}

	err := p.Install(context.Background(), []byte("kc"), addon)
	assert.NoError(t, err)

	status, err := p.Status(context.Background(), []byte("kc"), addon)
	require.NoError(t, err)
	assert.True(t, status.Installed)
	assert.True(t, status.Ready)
	assert.Equal(t, "1.15.0", status.Version)
}

func TestAddonProviderInstallEmptyName(t *testing.T) {
	var p AddonProvider = newMockAddonProvider()

	addon := AddonSpec{Type: "helm", Version: "1.0.0"}
	err := p.Install(context.Background(), []byte("kc"), addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name required")
}

func TestAddonProviderUpgrade(t *testing.T) {
	mock := newMockAddonProvider()
	var p AddonProvider = mock

	addon := AddonSpec{Name: "cilium", Type: "helm", Version: "1.15.0", Namespace: "kube-system"}
	err := p.Install(context.Background(), []byte("kc"), addon)
	require.NoError(t, err)

	upgraded := AddonSpec{Name: "cilium", Type: "helm", Version: "1.16.0", Namespace: "kube-system"}
	err = p.Upgrade(context.Background(), []byte("kc"), upgraded)
	assert.NoError(t, err)

	status, err := p.Status(context.Background(), []byte("kc"), upgraded)
	require.NoError(t, err)
	assert.Equal(t, "1.16.0", status.Version)
}

func TestAddonProviderUpgradeNotInstalled(t *testing.T) {
	var p AddonProvider = newMockAddonProvider()

	addon := AddonSpec{Name: "nonexistent", Version: "1.0.0"}
	err := p.Upgrade(context.Background(), []byte("kc"), addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestAddonProviderUninstall(t *testing.T) {
	mock := newMockAddonProvider()
	var p AddonProvider = mock

	addon := AddonSpec{Name: "metrics-server", Type: "manifest", Version: "0.7.0", Namespace: "kube-system"}
	err := p.Install(context.Background(), []byte("kc"), addon)
	require.NoError(t, err)

	err = p.Uninstall(context.Background(), []byte("kc"), addon)
	assert.NoError(t, err)

	status, err := p.Status(context.Background(), []byte("kc"), addon)
	require.NoError(t, err)
	assert.False(t, status.Installed)
}

func TestAddonProviderUninstallNotInstalled(t *testing.T) {
	var p AddonProvider = newMockAddonProvider()

	addon := AddonSpec{Name: "ghost", Version: "1.0.0"}
	err := p.Uninstall(context.Background(), []byte("kc"), addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestAddonProviderStatusNotInstalled(t *testing.T) {
	var p AddonProvider = newMockAddonProvider()

	addon := AddonSpec{Name: "unknown", Version: "1.0.0"}
	status, err := p.Status(context.Background(), []byte("kc"), addon)
	require.NoError(t, err)
	assert.Equal(t, "unknown", status.Name)
	assert.False(t, status.Installed)
}

func TestAddonProviderMultipleAddons(t *testing.T) {
	mock := newMockAddonProvider()
	var p AddonProvider = mock

	addons := []AddonSpec{
		{Name: "cilium", Type: "helm", Version: "1.15.0", Namespace: "kube-system"},
		{Name: "metrics-server", Type: "manifest", Version: "0.7.0", Namespace: "kube-system"},
		{Name: "cert-manager", Type: "helm", Version: "1.14.0", Namespace: "cert-manager"},
	}

	for _, a := range addons {
		err := p.Install(context.Background(), []byte("kc"), a)
		require.NoError(t, err)
	}

	assert.Len(t, mock.installed, 3)

	for _, a := range addons {
		status, err := p.Status(context.Background(), []byte("kc"), a)
		require.NoError(t, err)
		assert.True(t, status.Installed)
		assert.True(t, status.Ready)
	}
}
