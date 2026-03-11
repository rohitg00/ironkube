package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBootstrapProvider struct {
	versions []string
}

func (m *mockBootstrapProvider) Name() string {
	return "kubeadm"
}

func (m *mockBootstrapProvider) ValidateVersion(version string) error {
	for _, v := range m.versions {
		if v == version {
			return nil
		}
	}
	return fmt.Errorf("unsupported version: %s", version)
}

func (m *mockBootstrapProvider) SupportedVersions() []string {
	return m.versions
}

func (m *mockBootstrapProvider) InitControlPlane(_ context.Context, cluster *ClusterSpec, machine Machine, token string) (*BootstrapData, error) {
	return &BootstrapData{
		Scripts: []Script{
			{Name: "kubeadm-init", Content: fmt.Sprintf("kubeadm init --token %s --kubernetes-version %s", token, cluster.Version), RunAs: "root"},
		},
		EnvVars: map[string]string{
			"KUBECONFIG": "/etc/kubernetes/admin.conf",
		},
	}, nil
}

func (m *mockBootstrapProvider) JoinControlPlane(_ context.Context, cluster *ClusterSpec, machine Machine, joinEndpoint, token string) (*BootstrapData, error) {
	return &BootstrapData{
		Scripts: []Script{
			{Name: "kubeadm-join-cp", Content: fmt.Sprintf("kubeadm join %s --token %s --control-plane", joinEndpoint, token), RunAs: "root"},
		},
	}, nil
}

func (m *mockBootstrapProvider) JoinWorker(_ context.Context, _ *ClusterSpec, _ Machine, joinEndpoint, token string) (*BootstrapData, error) {
	return &BootstrapData{
		Scripts: []Script{
			{Name: "kubeadm-join-worker", Content: fmt.Sprintf("kubeadm join %s --token %s", joinEndpoint, token), RunAs: "root"},
		},
	}, nil
}

func (m *mockBootstrapProvider) Uninstall(_ context.Context, _ Machine, role string) (*BootstrapData, error) {
	return &BootstrapData{
		Scripts: []Script{
			{Name: "kubeadm-reset", Content: fmt.Sprintf("kubeadm reset --force # role=%s", role), RunAs: "root"},
		},
	}, nil
}

func (m *mockBootstrapProvider) Upgrade(_ context.Context, _ Machine, toVersion string) (*BootstrapData, error) {
	return &BootstrapData{
		Scripts: []Script{
			{Name: "kubeadm-upgrade", Content: fmt.Sprintf("kubeadm upgrade apply v%s", toVersion), RunAs: "root"},
		},
	}, nil
}

func (m *mockBootstrapProvider) SecurityArgs(profile SecurityProfile) []string {
	var args []string
	args = append(args, profile.APIServerFlags()...)
	args = append(args, profile.KubeletFlags()...)
	return args
}

func (m *mockBootstrapProvider) KubeconfigPath() string {
	return "/etc/kubernetes/admin.conf"
}

func (m *mockBootstrapProvider) TokenCommand() string {
	return "kubeadm token create --print-join-command"
}

type mockSecurityProfile struct{}

func (s *mockSecurityProfile) Name() string {
	return "cis-hardened"
}

func (s *mockSecurityProfile) APIServerFlags() []string {
	return []string{"--anonymous-auth=false", "--audit-log-path=/var/log/audit.log"}
}

func (s *mockSecurityProfile) KubeletFlags() []string {
	return []string{"--protect-kernel-defaults=true"}
}

func (s *mockSecurityProfile) EtcdFlags() []string {
	return []string{"--cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"}
}

func TestBootstrapProviderName(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.30.0"}}
	assert.Equal(t, "kubeadm", p.Name())
}

func TestBootstrapProviderValidateVersionSuccess(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.29.0", "1.30.0", "1.31.0"}}
	assert.NoError(t, p.ValidateVersion("1.30.0"))
}

func TestBootstrapProviderValidateVersionError(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.30.0"}}
	err := p.ValidateVersion("1.99.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported version")
}

func TestBootstrapProviderSupportedVersions(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.29.0", "1.30.0", "1.31.0"}}
	versions := p.SupportedVersions()
	assert.Len(t, versions, 3)
	assert.Contains(t, versions, "1.30.0")
}

func TestBootstrapProviderInitControlPlane(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.30.0"}}
	cluster := &ClusterSpec{Name: "test", Version: "1.30.0", PodCIDR: "10.244.0.0/16"}
	machine := Machine{ID: "cp-1", Spec: MachineSpec{Role: "control-plane", Host: "10.0.0.1"}}

	bd, err := p.InitControlPlane(context.Background(), cluster, machine, "abc123")
	require.NoError(t, err)
	require.NotNil(t, bd)
	require.Len(t, bd.Scripts, 1)
	assert.Equal(t, "kubeadm-init", bd.Scripts[0].Name)
	assert.Contains(t, bd.Scripts[0].Content, "abc123")
	assert.Contains(t, bd.Scripts[0].Content, "1.30.0")
	assert.Equal(t, "root", bd.Scripts[0].RunAs)
	assert.Equal(t, "/etc/kubernetes/admin.conf", bd.EnvVars["KUBECONFIG"])
}

func TestBootstrapProviderJoinControlPlane(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.30.0"}}
	cluster := &ClusterSpec{Name: "test", Version: "1.30.0"}
	machine := Machine{ID: "cp-2", Spec: MachineSpec{Role: "control-plane", Host: "10.0.0.2"}}

	bd, err := p.JoinControlPlane(context.Background(), cluster, machine, "10.0.0.1:6443", "token123")
	require.NoError(t, err)
	require.NotNil(t, bd)
	require.Len(t, bd.Scripts, 1)
	assert.Contains(t, bd.Scripts[0].Content, "10.0.0.1:6443")
	assert.Contains(t, bd.Scripts[0].Content, "token123")
	assert.Contains(t, bd.Scripts[0].Content, "--control-plane")
}

func TestBootstrapProviderJoinWorker(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.30.0"}}
	cluster := &ClusterSpec{Name: "test", Version: "1.30.0"}
	machine := Machine{ID: "w-1", Spec: MachineSpec{Role: "worker", Host: "10.0.0.5"}}

	bd, err := p.JoinWorker(context.Background(), cluster, machine, "10.0.0.1:6443", "token456")
	require.NoError(t, err)
	require.NotNil(t, bd)
	require.Len(t, bd.Scripts, 1)
	assert.Contains(t, bd.Scripts[0].Content, "10.0.0.1:6443")
	assert.Contains(t, bd.Scripts[0].Content, "token456")
	assert.NotContains(t, bd.Scripts[0].Content, "--control-plane")
}

func TestBootstrapProviderUninstall(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.30.0"}}
	machine := Machine{ID: "w-1", Spec: MachineSpec{Role: "worker", Host: "10.0.0.5"}}

	bd, err := p.Uninstall(context.Background(), machine, "worker")
	require.NoError(t, err)
	require.NotNil(t, bd)
	require.Len(t, bd.Scripts, 1)
	assert.Contains(t, bd.Scripts[0].Content, "kubeadm reset")
	assert.Contains(t, bd.Scripts[0].Content, "worker")
}

func TestBootstrapProviderUpgrade(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.30.0", "1.31.0"}}
	machine := Machine{ID: "cp-1", Spec: MachineSpec{Role: "control-plane", Host: "10.0.0.1"}}

	bd, err := p.Upgrade(context.Background(), machine, "1.31.0")
	require.NoError(t, err)
	require.NotNil(t, bd)
	require.Len(t, bd.Scripts, 1)
	assert.Contains(t, bd.Scripts[0].Content, "1.31.0")
}

func TestBootstrapProviderSecurityArgs(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.30.0"}}
	profile := &mockSecurityProfile{}

	args := p.SecurityArgs(profile)
	assert.Len(t, args, 3)
	assert.Contains(t, args, "--anonymous-auth=false")
	assert.Contains(t, args, "--audit-log-path=/var/log/audit.log")
	assert.Contains(t, args, "--protect-kernel-defaults=true")
}

func TestBootstrapProviderKubeconfigPath(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.30.0"}}
	path := p.KubeconfigPath()
	assert.NotEmpty(t, path)
	assert.Equal(t, "/etc/kubernetes/admin.conf", path)
}

func TestBootstrapProviderTokenCommand(t *testing.T) {
	var p BootstrapProvider = &mockBootstrapProvider{versions: []string{"1.30.0"}}
	cmd := p.TokenCommand()
	assert.NotEmpty(t, cmd)
	assert.Contains(t, cmd, "token")
}
