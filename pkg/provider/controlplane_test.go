package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockControlPlaneProvider struct {
	nodes    []NodeHealth
	certs    []CertInfo
	etcd     *EtcdHealth
	drained  map[string]bool
}

func newMockControlPlaneProvider() *mockControlPlaneProvider {
	return &mockControlPlaneProvider{
		nodes: []NodeHealth{
			{Name: "cp-1", Ready: true, Version: "v1.30.0", Roles: "control-plane"},
			{Name: "w-1", Ready: true, Version: "v1.30.0", Roles: "worker"},
			{Name: "w-2", Ready: true, Version: "v1.30.0", Roles: "worker"},
		},
		certs: []CertInfo{
			{Name: "apiserver", Path: "/etc/kubernetes/pki/apiserver.crt", ExpiresAt: time.Date(2027, 3, 11, 0, 0, 0, 0, time.UTC), Issuer: "kubernetes-ca"},
			{Name: "front-proxy", Path: "/etc/kubernetes/pki/front-proxy-client.crt", ExpiresAt: time.Date(2027, 3, 11, 0, 0, 0, 0, time.UTC), Issuer: "kubernetes-ca"},
		},
		etcd: &EtcdHealth{
			Healthy: true,
			Members: []EtcdMember{
				{Name: "etcd-0", Endpoint: "https://10.0.0.1:2379", IsLeader: true, DbSize: 50 * 1024 * 1024},
			},
			DbSize:  50 * 1024 * 1024,
			Version: "3.5.12",
		},
		drained: make(map[string]bool),
	}
}

func (m *mockControlPlaneProvider) WaitForAPI(_ context.Context, kubeconfig []byte, _ time.Duration) error {
	if len(kubeconfig) == 0 {
		return fmt.Errorf("empty kubeconfig")
	}
	return nil
}

func (m *mockControlPlaneProvider) NodeReady(_ context.Context, _ []byte, nodeName string) (bool, error) {
	for _, n := range m.nodes {
		if n.Name == nodeName {
			return n.Ready, nil
		}
	}
	return false, fmt.Errorf("node %s not found", nodeName)
}

func (m *mockControlPlaneProvider) WaitForNodes(_ context.Context, _ []byte, expected int, _ time.Duration) error {
	ready := 0
	for _, n := range m.nodes {
		if n.Ready {
			ready++
		}
	}
	if ready < expected {
		return fmt.Errorf("expected %d ready nodes, got %d", expected, ready)
	}
	return nil
}

func (m *mockControlPlaneProvider) DrainNode(_ context.Context, _ []byte, nodeName string, _ DrainOptions) error {
	for _, n := range m.nodes {
		if n.Name == nodeName {
			m.drained[nodeName] = true
			return nil
		}
	}
	return fmt.Errorf("node %s not found", nodeName)
}

func (m *mockControlPlaneProvider) UncordonNode(_ context.Context, _ []byte, nodeName string) error {
	if !m.drained[nodeName] {
		return fmt.Errorf("node %s is not cordoned", nodeName)
	}
	delete(m.drained, nodeName)
	return nil
}

func (m *mockControlPlaneProvider) Health(_ context.Context, _ []byte) (*HealthReport, error) {
	return &HealthReport{
		ClusterName: "test-cluster",
		Nodes:       m.nodes,
		Certs:       m.certs,
		Etcd:        m.etcd,
	}, nil
}

func (m *mockControlPlaneProvider) ListNodes(_ context.Context, _ []byte) ([]NodeHealth, error) {
	return m.nodes, nil
}

func (m *mockControlPlaneProvider) CertExpiry(_ context.Context, _ []byte) ([]CertInfo, error) {
	return m.certs, nil
}

func (m *mockControlPlaneProvider) EtcdHealth(_ context.Context, _ []byte) (*EtcdHealth, error) {
	return m.etcd, nil
}

func TestControlPlaneProviderWaitForAPI(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	err := p.WaitForAPI(context.Background(), []byte("kubeconfig-data"), 30*time.Second)
	assert.NoError(t, err)
}

func TestControlPlaneProviderWaitForAPIEmptyKubeconfig(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	err := p.WaitForAPI(context.Background(), nil, 30*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty kubeconfig")
}

func TestControlPlaneProviderNodeReady(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	ready, err := p.NodeReady(context.Background(), []byte("kc"), "cp-1")
	require.NoError(t, err)
	assert.True(t, ready)
}

func TestControlPlaneProviderNodeReadyNotFound(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	_, err := p.NodeReady(context.Background(), []byte("kc"), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestControlPlaneProviderWaitForNodes(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	err := p.WaitForNodes(context.Background(), []byte("kc"), 3, time.Minute)
	assert.NoError(t, err)
}

func TestControlPlaneProviderWaitForNodesInsufficient(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	err := p.WaitForNodes(context.Background(), []byte("kc"), 10, time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected 10")
}

func TestControlPlaneProviderDrainAndUncordon(t *testing.T) {
	mock := newMockControlPlaneProvider()
	var p ControlPlaneProvider = mock

	opts := DrainOptions{GracePeriod: 30, Timeout: 5 * time.Minute, IgnoreDaemonSets: true}
	err := p.DrainNode(context.Background(), []byte("kc"), "w-1", opts)
	require.NoError(t, err)
	assert.True(t, mock.drained["w-1"])

	err = p.UncordonNode(context.Background(), []byte("kc"), "w-1")
	assert.NoError(t, err)
	assert.False(t, mock.drained["w-1"])
}

func TestControlPlaneProviderDrainNotFound(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	err := p.DrainNode(context.Background(), []byte("kc"), "ghost", DrainOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestControlPlaneProviderUncordonNotDrained(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	err := p.UncordonNode(context.Background(), []byte("kc"), "w-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not cordoned")
}

func TestControlPlaneProviderHealth(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	report, err := p.Health(context.Background(), []byte("kc"))
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, "test-cluster", report.ClusterName)
	assert.Len(t, report.Nodes, 3)
	assert.True(t, report.AllHealthy())
	assert.Equal(t, 3, report.ReadyCount())
	require.NotNil(t, report.Etcd)
	assert.True(t, report.Etcd.Healthy)
}

func TestControlPlaneProviderListNodes(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	nodes, err := p.ListNodes(context.Background(), []byte("kc"))
	require.NoError(t, err)
	assert.Len(t, nodes, 3)
	assert.Equal(t, "cp-1", nodes[0].Name)
	assert.Equal(t, "control-plane", nodes[0].Roles)
}

func TestControlPlaneProviderCertExpiry(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	certs, err := p.CertExpiry(context.Background(), []byte("kc"))
	require.NoError(t, err)
	assert.Len(t, certs, 2)
	assert.Equal(t, "apiserver", certs[0].Name)
	assert.Equal(t, "kubernetes-ca", certs[0].Issuer)
	assert.Equal(t, 2027, certs[0].ExpiresAt.Year())
}

func TestControlPlaneProviderEtcdHealth(t *testing.T) {
	var p ControlPlaneProvider = newMockControlPlaneProvider()

	etcd, err := p.EtcdHealth(context.Background(), []byte("kc"))
	require.NoError(t, err)
	require.NotNil(t, etcd)
	assert.True(t, etcd.Healthy)
	assert.Len(t, etcd.Members, 1)
	assert.True(t, etcd.Members[0].IsLeader)
	assert.Equal(t, "3.5.12", etcd.Version)
}
