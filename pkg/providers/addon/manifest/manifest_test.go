package manifest

import (
	"context"
	"fmt"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ provider.AddonProvider = (*Provider)(nil)

type mockApplier struct {
	applyFunc  func(ctx context.Context, kubeconfig []byte, namespace string, manifests []byte) error
	deleteFunc func(ctx context.Context, kubeconfig []byte, namespace string, manifests []byte) error
}

func (m *mockApplier) Apply(ctx context.Context, kubeconfig []byte, namespace string, manifests []byte) error {
	if m.applyFunc != nil {
		return m.applyFunc(ctx, kubeconfig, namespace, manifests)
	}
	return nil
}

func (m *mockApplier) Delete(ctx context.Context, kubeconfig []byte, namespace string, manifests []byte) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, kubeconfig, namespace, manifests)
	}
	return nil
}

type mockFetcher struct {
	fetchFunc func(ctx context.Context, url string) ([]byte, error)
}

func (m *mockFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, url)
	}
	return []byte("apiVersion: v1\nkind: Namespace"), nil
}

func TestInstallFetchesThenApplies(t *testing.T) {
	calls := []string{}
	kubeconfig := []byte("kubeconfig-data")

	fetcher := &mockFetcher{
		fetchFunc: func(_ context.Context, url string) ([]byte, error) {
			calls = append(calls, fmt.Sprintf("fetch:%s", url))
			return []byte("manifest-content"), nil
		},
	}
	applier := &mockApplier{
		applyFunc: func(_ context.Context, kc []byte, ns string, data []byte) error {
			calls = append(calls, fmt.Sprintf("apply:%s", ns))
			assert.Equal(t, kubeconfig, kc)
			assert.Equal(t, []byte("manifest-content"), data)
			return nil
		},
	}
	p := New(applier, fetcher)

	addon := provider.AddonSpec{
		Name:        "flannel",
		ManifestURL: "https://raw.githubusercontent.com/flannel-io/flannel/master/kube-flannel.yml",
		Namespace:   "kube-flannel",
	}

	err := p.Install(context.Background(), kubeconfig, addon)
	require.NoError(t, err)
	require.Len(t, calls, 2)
	assert.Contains(t, calls[0], "fetch:")
	assert.Contains(t, calls[1], "apply:kube-flannel")
}

func TestInstallPropagatesFetchError(t *testing.T) {
	fetcher := &mockFetcher{
		fetchFunc: func(_ context.Context, _ string) ([]byte, error) {
			return nil, fmt.Errorf("404 not found")
		},
	}
	p := New(&mockApplier{}, fetcher)

	addon := provider.AddonSpec{
		Name:        "flannel",
		ManifestURL: "https://example.com/missing.yaml",
		Namespace:   "default",
	}

	err := p.Install(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404 not found")
	assert.Contains(t, err.Error(), "failed to fetch manifest")
}

func TestInstallPropagatesApplyError(t *testing.T) {
	fetcher := &mockFetcher{}
	applier := &mockApplier{
		applyFunc: func(_ context.Context, _ []byte, _ string, _ []byte) error {
			return fmt.Errorf("forbidden")
		},
	}
	p := New(applier, fetcher)

	addon := provider.AddonSpec{
		Name:        "flannel",
		ManifestURL: "https://example.com/flannel.yaml",
		Namespace:   "default",
	}

	err := p.Install(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
	assert.Contains(t, err.Error(), "failed to apply manifest")
}

func TestInstallEmptyManifestURLReturnsError(t *testing.T) {
	p := New(&mockApplier{}, &mockFetcher{})

	addon := provider.AddonSpec{
		Name:      "flannel",
		Namespace: "default",
	}

	err := p.Install(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest URL is required")
}

func TestUpgradeFetchesThenApplies(t *testing.T) {
	calls := []string{}
	fetcher := &mockFetcher{
		fetchFunc: func(_ context.Context, url string) ([]byte, error) {
			calls = append(calls, "fetch")
			return []byte("manifest-v2"), nil
		},
	}
	applier := &mockApplier{
		applyFunc: func(_ context.Context, _ []byte, _ string, data []byte) error {
			calls = append(calls, "apply")
			assert.Equal(t, []byte("manifest-v2"), data)
			return nil
		},
	}
	p := New(applier, fetcher)

	addon := provider.AddonSpec{
		Name:        "flannel",
		ManifestURL: "https://example.com/flannel-v2.yaml",
		Namespace:   "kube-flannel",
	}

	err := p.Upgrade(context.Background(), nil, addon)
	require.NoError(t, err)
	require.Len(t, calls, 2)
	assert.Equal(t, "fetch", calls[0])
	assert.Equal(t, "apply", calls[1])
}

func TestUpgradeEmptyManifestURLReturnsError(t *testing.T) {
	p := New(&mockApplier{}, &mockFetcher{})

	addon := provider.AddonSpec{
		Name:      "flannel",
		Namespace: "default",
	}

	err := p.Upgrade(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest URL is required")
}

func TestUninstallFetchesThenDeletes(t *testing.T) {
	calls := []string{}
	kubeconfig := []byte("kubeconfig-data")

	fetcher := &mockFetcher{
		fetchFunc: func(_ context.Context, url string) ([]byte, error) {
			calls = append(calls, "fetch")
			return []byte("manifest-content"), nil
		},
	}
	applier := &mockApplier{
		deleteFunc: func(_ context.Context, kc []byte, ns string, data []byte) error {
			calls = append(calls, fmt.Sprintf("delete:%s", ns))
			assert.Equal(t, kubeconfig, kc)
			assert.Equal(t, []byte("manifest-content"), data)
			return nil
		},
	}
	p := New(applier, fetcher)

	addon := provider.AddonSpec{
		Name:        "flannel",
		ManifestURL: "https://example.com/flannel.yaml",
		Namespace:   "kube-flannel",
	}

	err := p.Uninstall(context.Background(), kubeconfig, addon)
	require.NoError(t, err)
	require.Len(t, calls, 2)
	assert.Equal(t, "fetch", calls[0])
	assert.Equal(t, "delete:kube-flannel", calls[1])
}

func TestUninstallPropagatesFetchError(t *testing.T) {
	fetcher := &mockFetcher{
		fetchFunc: func(_ context.Context, _ string) ([]byte, error) {
			return nil, fmt.Errorf("timeout")
		},
	}
	p := New(&mockApplier{}, fetcher)

	addon := provider.AddonSpec{
		Name:        "flannel",
		ManifestURL: "https://example.com/flannel.yaml",
		Namespace:   "default",
	}

	err := p.Uninstall(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Contains(t, err.Error(), "failed to fetch manifest")
}

func TestUninstallPropagatesDeleteError(t *testing.T) {
	fetcher := &mockFetcher{}
	applier := &mockApplier{
		deleteFunc: func(_ context.Context, _ []byte, _ string, _ []byte) error {
			return fmt.Errorf("resource busy")
		},
	}
	p := New(applier, fetcher)

	addon := provider.AddonSpec{
		Name:        "flannel",
		ManifestURL: "https://example.com/flannel.yaml",
		Namespace:   "default",
	}

	err := p.Uninstall(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource busy")
	assert.Contains(t, err.Error(), "failed to delete manifest")
}

func TestUninstallEmptyManifestURLReturnsError(t *testing.T) {
	p := New(&mockApplier{}, &mockFetcher{})

	addon := provider.AddonSpec{
		Name:      "flannel",
		Namespace: "default",
	}

	err := p.Uninstall(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest URL is required")
}

func TestStatusReturnsInstalled(t *testing.T) {
	p := New(&mockApplier{}, &mockFetcher{})

	addon := provider.AddonSpec{
		Name:    "flannel",
		Version: "0.25.0",
	}

	status, err := p.Status(context.Background(), nil, addon)
	require.NoError(t, err)
	assert.Equal(t, "flannel", status.Name)
	assert.True(t, status.Installed)
	assert.True(t, status.Ready)
	assert.Equal(t, "0.25.0", status.Version)
}

func TestInstallPassesKubeconfigToApplier(t *testing.T) {
	expectedKC := []byte("my-kubeconfig")
	var receivedKC []byte

	applier := &mockApplier{
		applyFunc: func(_ context.Context, kc []byte, _ string, _ []byte) error {
			receivedKC = kc
			return nil
		},
	}
	p := New(applier, &mockFetcher{})

	addon := provider.AddonSpec{
		Name:        "flannel",
		ManifestURL: "https://example.com/flannel.yaml",
		Namespace:   "default",
	}

	err := p.Install(context.Background(), expectedKC, addon)
	require.NoError(t, err)
	assert.Equal(t, expectedKC, receivedKC)
}

func TestUpgradePropagatesApplyError(t *testing.T) {
	applier := &mockApplier{
		applyFunc: func(_ context.Context, _ []byte, _ string, _ []byte) error {
			return fmt.Errorf("conflict")
		},
	}
	p := New(applier, &mockFetcher{})

	addon := provider.AddonSpec{
		Name:        "flannel",
		ManifestURL: "https://example.com/flannel.yaml",
		Namespace:   "default",
	}

	err := p.Upgrade(context.Background(), nil, addon)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conflict")
}
