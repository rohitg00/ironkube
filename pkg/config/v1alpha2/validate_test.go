package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() *ClusterConfig {
	return &ClusterConfig{
		Metadata: Metadata{Name: "test-cluster"},
		Spec: ClusterSpec{
			Distro:  "k3s",
			Version: "v1.32.0",
			Infrastructure: Infrastructure{
				Provider: "ssh",
			},
			ControlPlane: ControlPlane{
				Replicas: 1,
				Nodes:    []Node{{Host: "10.0.0.1", User: "root", Port: 22}},
			},
			Networking: Networking{
				PodCIDR:     "10.42.0.0/16",
				ServiceCIDR: "10.43.0.0/16",
			},
			Security: Security{Profile: "minimal"},
		},
	}
}

func TestValidateValidConfig(t *testing.T) {
	err := Validate(validConfig())
	assert.NoError(t, err)
}

func TestValidateTableDriven(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*ClusterConfig)
		wantErr string
	}{
		{
			name:    "missing metadata name",
			modify:  func(c *ClusterConfig) { c.Metadata.Name = "" },
			wantErr: "metadata.name is required",
		},
		{
			name:    "invalid distro",
			modify:  func(c *ClusterConfig) { c.Spec.Distro = "minikube" },
			wantErr: "distro must be one of",
		},
		{
			name:    "empty distro",
			modify:  func(c *ClusterConfig) { c.Spec.Distro = "" },
			wantErr: "distro must be one of",
		},
		{
			name:    "missing version",
			modify:  func(c *ClusterConfig) { c.Spec.Version = "" },
			wantErr: "version is required",
		},
		{
			name:    "invalid provider",
			modify:  func(c *ClusterConfig) { c.Spec.Infrastructure.Provider = "aws" },
			wantErr: "infrastructure.provider must be one of",
		},
		{
			name:    "zero replicas",
			modify:  func(c *ClusterConfig) { c.Spec.ControlPlane.Replicas = 0 },
			wantErr: "controlPlane.replicas must be >= 1",
		},
		{
			name:    "negative replicas",
			modify:  func(c *ClusterConfig) { c.Spec.ControlPlane.Replicas = -1 },
			wantErr: "controlPlane.replicas must be >= 1",
		},
		{
			name:    "even HA replicas 2",
			modify:  func(c *ClusterConfig) { c.Spec.ControlPlane.Replicas = 2 },
			wantErr: "HA requires odd number",
		},
		{
			name:    "even HA replicas 4",
			modify:  func(c *ClusterConfig) { c.Spec.ControlPlane.Replicas = 4 },
			wantErr: "HA requires odd number",
		},
		{
			name: "ssh provider missing host in control plane",
			modify: func(c *ClusterConfig) {
				c.Spec.ControlPlane.Nodes = []Node{{User: "root", Port: 22}}
			},
			wantErr: "controlPlane.nodes[0].host is required",
		},
		{
			name: "ssh provider missing host in worker",
			modify: func(c *ClusterConfig) {
				c.Spec.Workers = []NodePool{
					{Name: "pool", Nodes: []Node{{User: "root"}}},
				}
			},
			wantErr: "workers[0].nodes[0].host is required",
		},
		{
			name:    "invalid security profile",
			modify:  func(c *ClusterConfig) { c.Spec.Security.Profile = "custom" },
			wantErr: "security.profile must be one of",
		},
		{
			name:    "invalid pod CIDR",
			modify:  func(c *ClusterConfig) { c.Spec.Networking.PodCIDR = "not-a-cidr" },
			wantErr: "networking.podCIDR is not a valid CIDR",
		},
		{
			name:    "invalid service CIDR",
			modify:  func(c *ClusterConfig) { c.Spec.Networking.ServiceCIDR = "10.0.0.1" },
			wantErr: "networking.serviceCIDR is not a valid CIDR",
		},
		{
			name: "empty addon name",
			modify: func(c *ClusterConfig) {
				c.Spec.Addons = []AddonConfig{{Name: "", Enabled: true}}
			},
			wantErr: "addons[0].name is required",
		},
		{
			name:   "valid distro kubeadm",
			modify: func(c *ClusterConfig) { c.Spec.Distro = "kubeadm" },
		},
		{
			name:   "valid distro k0s",
			modify: func(c *ClusterConfig) { c.Spec.Distro = "k0s" },
		},
		{
			name:   "valid distro talos",
			modify: func(c *ClusterConfig) { c.Spec.Distro = "talos" },
		},
		{
			name:   "valid odd HA replicas 3",
			modify: func(c *ClusterConfig) { c.Spec.ControlPlane.Replicas = 3 },
		},
		{
			name:   "valid odd HA replicas 5",
			modify: func(c *ClusterConfig) { c.Spec.ControlPlane.Replicas = 5 },
		},
		{
			name:   "valid provider hetzner no host check",
			modify: func(c *ClusterConfig) {
				c.Spec.Infrastructure.Provider = "hetzner"
				c.Spec.ControlPlane.Nodes = []Node{{}}
			},
		},
		{
			name:   "valid provider docker no host check",
			modify: func(c *ClusterConfig) {
				c.Spec.Infrastructure.Provider = "docker"
				c.Spec.ControlPlane.Nodes = nil
			},
		},
		{
			name:   "valid security profile cis-hardened",
			modify: func(c *ClusterConfig) { c.Spec.Security.Profile = "cis-hardened" },
		},
		{
			name:   "valid security profile soc2",
			modify: func(c *ClusterConfig) { c.Spec.Security.Profile = "soc2" },
		},
		{
			name:   "valid security profile pci-dss",
			modify: func(c *ClusterConfig) { c.Spec.Security.Profile = "pci-dss" },
		},
		{
			name: "valid addon with name",
			modify: func(c *ClusterConfig) {
				c.Spec.Addons = []AddonConfig{{Name: "cilium", Enabled: true}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modify(cfg)
			err := Validate(cfg)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNilConfig(t *testing.T) {
	err := Validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestValidateMultipleErrors(t *testing.T) {
	cfg := &ClusterConfig{
		Spec: ClusterSpec{
			ControlPlane: ControlPlane{Replicas: 0},
			Security:     Security{Profile: "invalid"},
		},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name is required")
	assert.Contains(t, err.Error(), "version is required")
	assert.Contains(t, err.Error(), "controlPlane.replicas must be >= 1")
}
