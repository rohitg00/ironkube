package talos

import (
	"context"
	"strings"
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ provider.BootstrapProvider = (*Talos)(nil)

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
	tal := New()
	assert.Equal(t, "talos", tal.Name())
}

func TestValidateVersion(t *testing.T) {
	tal := New()
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid version", "v1.6.0", false},
		{"valid older version", "v1.5.5", false},
		{"valid patch version", "v1.7.1", false},
		{"missing v prefix", "1.6.0", true},
		{"empty version", "", true},
		{"only v prefix", "v", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tal.ValidateVersion(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSupportedVersions(t *testing.T) {
	tal := New()
	versions := tal.SupportedVersions()
	assert.NotEmpty(t, versions)
	for _, v := range versions {
		assert.True(t, strings.HasPrefix(v, "v"))
	}
}

func TestInitControlPlaneSingleNode(t *testing.T) {
	tal := New()
	cluster := &provider.ClusterSpec{
		Name:        "talos-cluster",
		Distro:      "talos",
		Version:     "v1.6.0",
		PodCIDR:     "10.244.0.0/16",
		ServiceCIDR: "10.96.0.0/12",
		HAEnabled:   false,
	}
	machine := provider.Machine{
		ID: "cp-1",
		Spec: provider.MachineSpec{
			Role: "controlplane",
			Host: "10.0.0.1",
		},
	}

	data, err := tal.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)
	require.NotNil(t, data)

	require.GreaterOrEqual(t, len(data.Scripts), 3)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "talosctl gen config")
	assert.Contains(t, combined, "talosctl apply-config")
	assert.Contains(t, combined, "--insecure")
	assert.Contains(t, combined, "talosctl bootstrap")

	require.NotNil(t, data.WaitCheck)
	assert.Contains(t, data.WaitCheck.Command, "talosctl health")
	assert.Equal(t, 300.0, data.WaitCheck.Timeout.Seconds())
}

func TestInitControlPlaneHA(t *testing.T) {
	tal := New()
	cluster := &provider.ClusterSpec{
		Name:        "ha-talos",
		Distro:      "talos",
		Version:     "v1.6.0",
		PodCIDR:     "10.244.0.0/16",
		ServiceCIDR: "10.96.0.0/12",
		HAEnabled:   true,
		FirstCPHost: "10.0.0.100",
	}
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := tal.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)
	require.NotNil(t, data)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "talosctl gen config")
}

func TestInitControlPlaneClusterName(t *testing.T) {
	tal := New()
	cluster := &provider.ClusterSpec{
		Name:    "my-cluster",
		Distro:  "talos",
		Version: "v1.6.0",
	}
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := tal.InitControlPlane(context.Background(), cluster, machine, "")
	require.NoError(t, err)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "my-cluster")
}

func TestJoinControlPlane(t *testing.T) {
	tal := New()
	cluster := &provider.ClusterSpec{
		Name:    "ha-talos",
		Distro:  "talos",
		Version: "v1.6.0",
	}
	machine := provider.Machine{
		ID:   "cp-2",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.2"},
	}

	data, err := tal.JoinControlPlane(context.Background(), cluster, machine, "10.0.0.1:50000", "")
	require.NoError(t, err)
	require.NotNil(t, data)
	require.NotEmpty(t, data.Scripts)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "talosctl apply-config")
	assert.Contains(t, combined, "10.0.0.2")
	assert.Contains(t, combined, "controlplane")
}

func TestJoinWorker(t *testing.T) {
	tal := New()
	cluster := &provider.ClusterSpec{
		Name:    "talos-cluster",
		Distro:  "talos",
		Version: "v1.6.0",
	}
	machine := provider.Machine{
		ID:   "worker-1",
		Spec: provider.MachineSpec{Role: "worker", Host: "10.0.0.3"},
	}

	data, err := tal.JoinWorker(context.Background(), cluster, machine, "10.0.0.1:50000", "")
	require.NoError(t, err)
	require.NotNil(t, data)
	require.NotEmpty(t, data.Scripts)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "talosctl apply-config")
	assert.Contains(t, combined, "10.0.0.3")
	assert.Contains(t, combined, "worker")
	assert.NotContains(t, combined, "controlplane")
}

func TestUninstall(t *testing.T) {
	tal := New()
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := tal.Uninstall(context.Background(), machine, "controlplane")
	require.NoError(t, err)
	require.NotEmpty(t, data.Scripts)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "talosctl reset")
}

func TestUpgrade(t *testing.T) {
	tal := New()
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.1"},
	}

	data, err := tal.Upgrade(context.Background(), machine, "v1.7.0")
	require.NoError(t, err)
	require.NotEmpty(t, data.Scripts)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "talosctl upgrade")
	assert.Contains(t, combined, "--image")
	assert.Contains(t, combined, "v1.7.0")
}

func TestSecurityArgsNilProfile(t *testing.T) {
	tal := New()
	args := tal.SecurityArgs(nil)
	assert.Empty(t, args)
}

func TestSecurityArgsDefaultProfile(t *testing.T) {
	tal := New()
	profile := &testSecurityProfile{name: "default"}
	args := tal.SecurityArgs(profile)
	assert.Empty(t, args)
}

func TestSecurityArgsCISProfile(t *testing.T) {
	tal := New()
	profile := &testSecurityProfile{
		name:           "cis",
		apiServerFlags: []string{"--audit-log-path=/var/log/audit.log"},
	}
	args := tal.SecurityArgs(profile)
	assert.Empty(t, args)
}

func TestKubeconfigPathEmpty(t *testing.T) {
	tal := New()
	assert.Equal(t, "", tal.KubeconfigPath())
}

func TestTokenCommandEmpty(t *testing.T) {
	tal := New()
	assert.Equal(t, "", tal.TokenCommand())
}

func TestUpgradeIncludesNodeEndpoint(t *testing.T) {
	tal := New()
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.5"},
	}

	data, err := tal.Upgrade(context.Background(), machine, "v1.7.0")
	require.NoError(t, err)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "10.0.0.5")
}

func TestUninstallIncludesNodeEndpoint(t *testing.T) {
	tal := New()
	machine := provider.Machine{
		ID:   "cp-1",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.5"},
	}

	data, err := tal.Uninstall(context.Background(), machine, "controlplane")
	require.NoError(t, err)

	combined := ""
	for _, s := range data.Scripts {
		combined += s.Content + "\n"
	}
	assert.Contains(t, combined, "10.0.0.5")
}

func TestJoinControlPlaneWaitCondition(t *testing.T) {
	tal := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "talos",
		Version: "v1.6.0",
	}
	machine := provider.Machine{
		ID:   "cp-2",
		Spec: provider.MachineSpec{Role: "controlplane", Host: "10.0.0.2"},
	}

	data, err := tal.JoinControlPlane(context.Background(), cluster, machine, "10.0.0.1:50000", "")
	require.NoError(t, err)
	require.NotNil(t, data.WaitCheck)
}

func TestJoinWorkerWaitCondition(t *testing.T) {
	tal := New()
	cluster := &provider.ClusterSpec{
		Name:    "test",
		Distro:  "talos",
		Version: "v1.6.0",
	}
	machine := provider.Machine{
		ID:   "worker-1",
		Spec: provider.MachineSpec{Role: "worker", Host: "10.0.0.3"},
	}

	data, err := tal.JoinWorker(context.Background(), cluster, machine, "10.0.0.1:50000", "")
	require.NoError(t, err)
	require.NotNil(t, data.WaitCheck)
}
