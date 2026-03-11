package kubeadm

import (
	"context"
	"strings"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ provider.BootstrapProvider = (*Kubeadm)(nil)

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
	assert.Equal(t, "kubeadm", k.Name())
}

func TestValidateVersion(t *testing.T) {
	k := New()
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid version", "v1.28.2", false},
		{"valid minor only", "v1.29.0", false},
		{"missing v prefix", "1.28.2", true},
		{"empty version", "", true},
		{"no dots", "v128", true},
		{"one dot", "v1.28", true},
		{"valid with patch", "v1.30.1", false},
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
	}
}

func TestInitControlPlaneSingleNode(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:        "test-cluster",
		Distro:      "kubeadm",
		Version:     "v1.28.2",
		PodCIDR:     "10.244.0.0/16",
		ServiceCIDR: "10.96.0.0/12",
		HAEnabled:   false,
	}
	machine := provider.Machine{
		ID: "cp-1",
		Spec: provider.MachineSpec{
			Role: "controlplane",
			Host: "10.0.0.1",
			User: "root",
			Port: 22,
		},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)
	require.NotNil(t, data)

	require.GreaterOrEqual(t, len(data.Scripts), 2)

	assert.Contains(t, data.Scripts[0].Content, "apt-get")
	assert.Contains(t, data.Scripts[0].Content, "kubelet")
	assert.Contains(t, data.Scripts[0].Content, "kubeadm")
	assert.Contains(t, data.Scripts[0].Content, "kubectl")

	assert.Contains(t, data.Scripts[1].Content, "kubeadm init")
	assert.Contains(t, data.Scripts[1].Content, "--pod-network-cidr=10.244.0.0/16")
	assert.Contains(t, data.Scripts[1].Content, "--service-cidr=10.96.0.0/12")
	assert.NotContains(t, data.Scripts[1].Content, "--control-plane-endpoint")
	assert.NotContains(t, data.Scripts[1].Content, "--upload-certs")

	require.NotNil(t, data.WaitCheck)
	assert.Contains(t, data.WaitCheck.Command, "kubectl get nodes")
}

func TestInitControlPlaneHA(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:        "ha-cluster",
		Distro:      "kubeadm",
		Version:     "v1.28.2",
		PodCIDR:     "10.244.0.0/16",
		ServiceCIDR: "10.96.0.0/12",
		HAEnabled:   true,
		FirstCPHost: "10.0.0.100",
	}
	machine := provider.Machine{
		ID: "cp-1",
		Spec: provider.MachineSpec{
			Role: "controlplane",
			Host: "10.0.0.1",
		},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)

	initScript := data.Scripts[1].Content
	assert.Contains(t, initScript, "--control-plane-endpoint")
	assert.Contains(t, initScript, "--upload-certs")
}

func TestInitControlPlaneVersionPinning(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "kubeadm",
		Version: "v1.29.0",
	}
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)

	assert.Contains(t, data.Scripts[0].Content, "1.29.0")
}

func TestJoinControlPlane(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "ha-cluster",
		Distro:  "kubeadm",
		Version: "v1.28.2",
	}
	machine := provider.Machine{
		ID:   "cp-2",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.2"},
	}

	data, err := k.JoinControlPlane(context.Background(), cluster, machine, "10.0.0.1:6443", "jointoken")
	require.NoError(t, err)
	require.NotNil(t, data)

	found := false
	for _, s := range data.Scripts {
		if strings.Contains(s.Content, "kubeadm join") {
			found = true
			assert.Contains(t, s.Content, "10.0.0.1:6443")
			assert.Contains(t, s.Content, "--control-plane")
			assert.Contains(t, s.Content, "--token")
		}
	}
	assert.True(t, found, "expected kubeadm join script")
}

func TestJoinWorker(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "kubeadm",
		Version: "v1.28.2",
	}
	machine := provider.Machine{
		ID:   "worker-1",
		Spec: provider.MachineSpec{Role: "worker", Host: "10.0.0.3"},
	}

	data, err := k.JoinWorker(context.Background(), cluster, machine, "10.0.0.1:6443", "workertoken")
	require.NoError(t, err)
	require.NotNil(t, data)

	found := false
	for _, s := range data.Scripts {
		if strings.Contains(s.Content, "kubeadm join") {
			found = true
			assert.Contains(t, s.Content, "10.0.0.1:6443")
			assert.NotContains(t, s.Content, "--control-plane")
		}
	}
	assert.True(t, found, "expected kubeadm join script")
}

func TestUninstall(t *testing.T) {
	k := New()
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.Uninstall(context.Background(), machine, "controlplane")
	require.NoError(t, err)
	require.NotEmpty(t, data.Scripts)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + " "
	}
	assert.Contains(t, combined, "kubeadm reset")
	assert.Contains(t, combined, "apt-get")
}

func TestUpgrade(t *testing.T) {
	k := New()
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.Upgrade(context.Background(), machine, "v1.29.0")
	require.NoError(t, err)
	require.NotEmpty(t, data.Scripts)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + " "
	}
	assert.Contains(t, combined, "kubeadm")
	assert.Contains(t, combined, "1.29.0")
	assert.Contains(t, combined, "upgrade apply")
}

func TestSecurityArgsCISProfile(t *testing.T) {
	k := New()
	profile := &testSecurityProfile{
		name:           "cis",
		apiServerFlags: []string{"--audit-log-path=/var/log/apiserver/audit.log"},
		kubeletFlags:   []string{"--protect-kernel-defaults=true"},
		etcdFlags:      []string{"--peer-auto-tls=false"},
	}

	args := k.SecurityArgs(profile)
	assert.NotEmpty(t, args)
	assert.Contains(t, args, "--audit-log-path=/var/log/apiserver/audit.log")
	assert.Contains(t, args, "--protect-kernel-defaults=true")
	assert.Contains(t, args, "--peer-auto-tls=false")
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
	assert.Equal(t, "/etc/kubernetes/admin.conf", k.KubeconfigPath())
}

func TestTokenCommand(t *testing.T) {
	k := New()
	assert.Equal(t, "kubeadm token create --print-join-command", k.TokenCommand())
}

func TestInitControlPlaneWaitTimeout(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "kubeadm",
		Version: "v1.28.2",
	}
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)
	require.NotNil(t, data.WaitCheck)
	assert.Equal(t, 180.0, data.WaitCheck.Timeout.Seconds())
}

func TestJoinControlPlaneWaitCondition(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "kubeadm",
		Version: "v1.28.2",
	}
	machine := provider.Machine{
		ID:   "cp-2",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.2"},
	}

	data, err := k.JoinControlPlane(context.Background(), cluster, machine, "10.0.0.1:6443", "token")
	require.NoError(t, err)
	require.NotNil(t, data.WaitCheck)
}

func TestJoinWorkerWaitCondition(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "kubeadm",
		Version: "v1.28.2",
	}
	machine := provider.Machine{
		ID:   "worker-1",
		Spec: provider.MachineSpec{Role: "worker", Host: "10.0.0.3"},
	}

	data, err := k.JoinWorker(context.Background(), cluster, machine, "10.0.0.1:6443", "token")
	require.NoError(t, err)
	require.NotNil(t, data.WaitCheck)
}

func TestInitScriptRunsAsRoot(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "kubeadm",
		Version: "v1.28.2",
	}
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)
	for _, s := range data.Scripts {
		assert.Equal(t, "root", s.RunAs)
	}
}
