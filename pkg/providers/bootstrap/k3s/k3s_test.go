package k3s

import (
	"context"
	"strings"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ provider.BootstrapProvider = (*K3s)(nil)

type testSecurityProfile struct {
	name           string
	apiServerFlags []string
	kubeletFlags   []string
	etcdFlags      []string
}

func (p *testSecurityProfile) Name() string           { return p.name }
func (p *testSecurityProfile) APIServerFlags() []string { return p.apiServerFlags }
func (p *testSecurityProfile) KubeletFlags() []string   { return p.kubeletFlags }
func (p *testSecurityProfile) EtcdFlags() []string      { return p.etcdFlags }

func TestName(t *testing.T) {
	k := New()
	assert.Equal(t, "k3s", k.Name())
}

func TestValidateVersion(t *testing.T) {
	k := New()
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid version", "v1.28.2+k3s1", false},
		{"valid older version", "v1.27.6+k3s1", false},
		{"missing v prefix", "1.28.2+k3s1", true},
		{"missing k3s suffix", "v1.28.2", true},
		{"empty version", "", true},
		{"only v", "v", true},
		{"k3s suffix no version", "v+k3s1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := k.ValidateVersion(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSupportedVersions(t *testing.T) {
	k := New()
	versions := k.SupportedVersions()
	assert.NotEmpty(t, versions)
	for _, v := range versions {
		assert.True(t, strings.HasPrefix(v, "v"))
		assert.Contains(t, v, "+k3s")
	}
}

func TestInitControlPlaneSingleNode(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:        "test",
		Distro:      "k3s",
		Version:     "v1.28.2+k3s1",
		PodCIDR:     "10.42.0.0/16",
		ServiceCIDR: "10.43.0.0/16",
		HAEnabled:   false,
	}
	machine := provider.Machine{
		ID: "node-1",
		Spec: provider.MachineSpec{
			Role: "controlplane",
			Host: "10.0.0.1",
			User: "root",
			Port: 22,
		},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "mytoken")
	require.NoError(t, err)
	require.NotNil(t, data)

	require.Len(t, data.Scripts, 1)
	assert.Contains(t, data.Scripts[0].Content, "curl -sfL https://get.k3s.io | sh -s - server")
	assert.Contains(t, data.Scripts[0].Content, "--cluster-cidr=10.42.0.0/16")
	assert.Contains(t, data.Scripts[0].Content, "--service-cidr=10.43.0.0/16")
	assert.NotContains(t, data.Scripts[0].Content, "--cluster-init")
	assert.Equal(t, "root", data.Scripts[0].RunAs)

	assert.Equal(t, "v1.28.2+k3s1", data.EnvVars["INSTALL_K3S_VERSION"])

	require.NotNil(t, data.WaitCheck)
	assert.Contains(t, data.WaitCheck.Command, "k3s kubectl get nodes")
}

func TestInitControlPlaneHA(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:        "ha-cluster",
		Distro:      "k3s",
		Version:     "v1.28.2+k3s1",
		PodCIDR:     "10.42.0.0/16",
		ServiceCIDR: "10.43.0.0/16",
		HAEnabled:   true,
	}
	machine := provider.Machine{
		ID: "node-1",
		Spec: provider.MachineSpec{
			Role: "controlplane",
			Host: "10.0.0.1",
		},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "hatoken")
	require.NoError(t, err)

	assert.Contains(t, data.Scripts[0].Content, "--cluster-init")
	assert.Equal(t, "hatoken", data.EnvVars["K3S_TOKEN"])
}

func TestInitControlPlaneNoToken(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "k3s",
		Version: "v1.28.2+k3s1",
	}
	machine := provider.Machine{
		ID:   "node-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)
	_, hasToken := data.EnvVars["K3S_TOKEN"]
	assert.False(t, hasToken)
}

func TestJoinControlPlane(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "ha-cluster",
		Distro:  "k3s",
		Version: "v1.28.2+k3s1",
	}
	machine := provider.Machine{
		ID:   "node-2",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.2"},
	}

	data, err := k.JoinControlPlane(context.Background(), cluster, machine, "https://10.0.0.1:6443", "jointoken")
	require.NoError(t, err)
	require.NotNil(t, data)

	require.Len(t, data.Scripts, 1)
	assert.Contains(t, data.Scripts[0].Content, "curl -sfL https://get.k3s.io | sh -s - server")
	assert.Equal(t, "https://10.0.0.1:6443", data.EnvVars["K3S_URL"])
	assert.Equal(t, "jointoken", data.EnvVars["K3S_TOKEN"])
	assert.Equal(t, "v1.28.2+k3s1", data.EnvVars["INSTALL_K3S_VERSION"])
}

func TestJoinWorker(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "k3s",
		Version: "v1.28.2+k3s1",
	}
	machine := provider.Machine{
		ID:   "worker-1",
		Spec: provider.MachineSpec{Role: "worker", Host: "10.0.0.3"},
	}

	data, err := k.JoinWorker(context.Background(), cluster, machine, "https://10.0.0.1:6443", "workertoken")
	require.NoError(t, err)
	require.NotNil(t, data)

	require.Len(t, data.Scripts, 1)
	assert.Contains(t, data.Scripts[0].Content, "sh -s - agent")
	assert.NotContains(t, data.Scripts[0].Content, "server")
	assert.Equal(t, "https://10.0.0.1:6443", data.EnvVars["K3S_URL"])
	assert.Equal(t, "workertoken", data.EnvVars["K3S_TOKEN"])
}

func TestUninstallServer(t *testing.T) {
	k := New()
	machine := provider.Machine{
		ID:   "node-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.Uninstall(context.Background(), machine, "controlplane")
	require.NoError(t, err)
	require.Len(t, data.Scripts, 1)
	assert.Contains(t, data.Scripts[0].Content, "k3s-uninstall.sh")
}

func TestUninstallAgent(t *testing.T) {
	k := New()
	machine := provider.Machine{
		ID:   "worker-1",
		Spec: provider.MachineSpec{Role: "worker", Host: "10.0.0.3"},
	}

	data, err := k.Uninstall(context.Background(), machine, "worker")
	require.NoError(t, err)
	require.Len(t, data.Scripts, 1)
	assert.Contains(t, data.Scripts[0].Content, "k3s-agent-uninstall.sh")
}

func TestUpgrade(t *testing.T) {
	k := New()
	machine := provider.Machine{
		ID:   "node-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.Upgrade(context.Background(), machine, "v1.29.0+k3s1")
	require.NoError(t, err)
	require.Len(t, data.Scripts, 1)
	assert.Contains(t, data.Scripts[0].Content, "curl -sfL https://get.k3s.io")
	assert.Contains(t, data.Scripts[0].Content, "v1.29.0+k3s1")
}

func TestSecurityArgsCISProfile(t *testing.T) {
	k := New()
	profile := &testSecurityProfile{
		name:         "cis",
		kubeletFlags: []string{"--protect-kernel-defaults=true"},
	}

	args := k.SecurityArgs(profile)
	assert.Contains(t, args, "--protect-kernel-defaults")
	assert.Contains(t, args, "--secrets-encryption")
}

func TestSecurityArgsDefaultProfile(t *testing.T) {
	k := New()
	profile := &testSecurityProfile{
		name: "default",
	}

	args := k.SecurityArgs(profile)
	assert.Empty(t, args)
}

func TestSecurityArgsNilProfile(t *testing.T) {
	k := New()
	args := k.SecurityArgs(nil)
	assert.Empty(t, args)
}

func TestKubeconfigPath(t *testing.T) {
	k := New()
	assert.Equal(t, "/etc/rancher/k3s/k3s.yaml", k.KubeconfigPath())
}

func TestTokenCommand(t *testing.T) {
	k := New()
	assert.Equal(t, "cat /var/lib/rancher/k3s/server/node-token", k.TokenCommand())
}

func TestInitControlPlaneWaitCondition(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "k3s",
		Version: "v1.28.2+k3s1",
	}
	machine := provider.Machine{
		ID:   "node-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "token")
	require.NoError(t, err)
	require.NotNil(t, data.WaitCheck)
	assert.Equal(t, "Ready", data.WaitCheck.Expected)
	assert.Equal(t, 120.0, data.WaitCheck.Timeout.Seconds())
}

func TestJoinControlPlaneWaitCondition(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "k3s",
		Version: "v1.28.2+k3s1",
	}
	machine := provider.Machine{
		ID:   "node-2",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.2"},
	}

	data, err := k.JoinControlPlane(context.Background(), cluster, machine, "https://10.0.0.1:6443", "token")
	require.NoError(t, err)
	require.NotNil(t, data.WaitCheck)
}

func TestJoinWorkerWaitCondition(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "k3s",
		Version: "v1.28.2+k3s1",
	}
	machine := provider.Machine{
		ID:   "worker-1",
		Spec: provider.MachineSpec{Role: "worker", Host: "10.0.0.3"},
	}

	data, err := k.JoinWorker(context.Background(), cluster, machine, "https://10.0.0.1:6443", "token")
	require.NoError(t, err)
	require.NotNil(t, data.WaitCheck)
}
