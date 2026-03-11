package k0s

import (
	"context"
	"strings"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ provider.BootstrapProvider = (*K0s)(nil)

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
	assert.Equal(t, "k0s", k.Name())
}

func TestValidateVersion(t *testing.T) {
	k := New()
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid version", "v1.28.2+k0s.0", false},
		{"valid newer version", "v1.30.1+k0s.0", false},
		{"missing v prefix", "1.28.2+k0s.0", true},
		{"missing k0s suffix", "v1.28.2", true},
		{"empty version", "", true},
		{"only v prefix", "v", true},
		{"k0s suffix variant", "v1.29.0+k0s.1", false},
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
		assert.Contains(t, v, "+k0s")
	}
}

func TestInitControlPlaneSingleNode(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:        "test",
		Distro:      "k0s",
		Version:     "v1.28.2+k0s.0",
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
		},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)
	require.NotNil(t, data)

	require.NotEmpty(t, data.Files)
	configFile := data.Files[0]
	assert.Equal(t, "/etc/k0s/k0s.yaml", configFile.Path)
	assert.Contains(t, configFile.Content, "10.244.0.0/16")
	assert.Contains(t, configFile.Content, "10.96.0.0/12")

	require.NotEmpty(t, data.Scripts)
	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "curl")
	assert.Contains(t, combined, "k0s install controller")
	assert.Contains(t, combined, "k0s start")

	require.NotNil(t, data.WaitCheck)
	assert.Contains(t, data.WaitCheck.Command, "k0s kubectl get nodes")
}

func TestInitControlPlaneHA(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:        "ha-cluster",
		Distro:      "k0s",
		Version:     "v1.28.2+k0s.0",
		PodCIDR:     "10.244.0.0/16",
		ServiceCIDR: "10.96.0.0/12",
		HAEnabled:   true,
	}
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "hatoken")
	require.NoError(t, err)
	require.NotNil(t, data)

	require.NotEmpty(t, data.Files)
	assert.Contains(t, data.Files[0].Content, "10.244.0.0/16")
}

func TestInitControlPlaneConfigFileMode(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "k0s",
		Version: "v1.28.2+k0s.0",
	}
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)
	require.NotEmpty(t, data.Files)
	assert.Equal(t, uint32(0644), data.Files[0].Mode)
}

func TestJoinControlPlane(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "ha-cluster",
		Distro:  "k0s",
		Version: "v1.28.2+k0s.0",
	}
	machine := provider.Machine{
		ID:   "cp-2",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.2"},
	}

	data, err := k.JoinControlPlane(context.Background(), cluster, machine, "10.0.0.1:6443", "controllertoken")
	require.NoError(t, err)
	require.NotNil(t, data)

	require.NotEmpty(t, data.Files)
	tokenFile := data.Files[0]
	assert.Contains(t, tokenFile.Path, "token")
	assert.Equal(t, "controllertoken", tokenFile.Content)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "k0s install controller")
	assert.Contains(t, combined, "--token-file")
}

func TestJoinWorker(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "k0s",
		Version: "v1.28.2+k0s.0",
	}
	machine := provider.Machine{
		ID:   "worker-1",
		Spec: provider.MachineSpec{Role: "worker", Host: "10.0.0.3"},
	}

	data, err := k.JoinWorker(context.Background(), cluster, machine, "10.0.0.1:6443", "workertoken")
	require.NoError(t, err)
	require.NotNil(t, data)

	require.NotEmpty(t, data.Files)
	assert.Equal(t, "workertoken", data.Files[0].Content)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "k0s install worker")
	assert.Contains(t, combined, "--token-file")
	assert.NotContains(t, combined, "controller")
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
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "k0s stop")
	assert.Contains(t, combined, "k0s reset")
}

func TestUpgrade(t *testing.T) {
	k := New()
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.Upgrade(context.Background(), machine, "v1.29.0+k0s.0")
	require.NoError(t, err)
	require.NotEmpty(t, data.Scripts)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "k0s stop")
	assert.Contains(t, combined, "v1.29.0+k0s.0")
	assert.Contains(t, combined, "k0s start")
}

func TestSecurityArgsCISProfile(t *testing.T) {
	k := New()
	profile := &testSecurityProfile{
		name:           "cis",
		apiServerFlags: []string{"--audit-log-path=/var/log/audit.log"},
		kubeletFlags:   []string{"--protect-kernel-defaults=true"},
	}

	args := k.SecurityArgs(profile)
	assert.NotEmpty(t, args)
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
	assert.Equal(t, "/var/lib/k0s/pki/admin.conf", k.KubeconfigPath())
}

func TestTokenCommand(t *testing.T) {
	k := New()
	cmd := k.TokenCommand()
	assert.Contains(t, cmd, "k0s token create")
	assert.Contains(t, cmd, "--role=controller")
}

func TestInitControlPlaneWaitCondition(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "k0s",
		Version: "v1.28.2+k0s.0",
	}
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := k.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)
	require.NotNil(t, data.WaitCheck)
	assert.Equal(t, "Ready", data.WaitCheck.Expected)
}

func TestJoinControlPlaneWaitCondition(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "k0s",
		Version: "v1.28.2+k0s.0",
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
		Distro:  "k0s",
		Version: "v1.28.2+k0s.0",
	}
	machine := provider.Machine{
		ID:   "worker-1",
		Spec: provider.MachineSpec{Role: "worker", Host: "10.0.0.3"},
	}

	data, err := k.JoinWorker(context.Background(), cluster, machine, "10.0.0.1:6443", "token")
	require.NoError(t, err)
	require.NotNil(t, data.WaitCheck)
}

func TestScriptsRunAsRoot(t *testing.T) {
	k := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "k0s",
		Version: "v1.28.2+k0s.0",
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
