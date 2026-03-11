package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyDefaultsProvider(t *testing.T) {
	cfg := &ClusterConfig{}
	ApplyDefaults(cfg)
	assert.Equal(t, "ssh", cfg.Spec.Infrastructure.Provider)
}

func TestApplyDefaultsSecurityProfile(t *testing.T) {
	cfg := &ClusterConfig{}
	ApplyDefaults(cfg)
	assert.Equal(t, "minimal", cfg.Spec.Security.Profile)
}

func TestApplyDefaultsNetworking(t *testing.T) {
	cfg := &ClusterConfig{}
	ApplyDefaults(cfg)
	assert.Equal(t, "10.42.0.0/16", cfg.Spec.Networking.PodCIDR)
	assert.Equal(t, "10.43.0.0/16", cfg.Spec.Networking.ServiceCIDR)
}

func TestApplyDefaultsState(t *testing.T) {
	cfg := &ClusterConfig{}
	ApplyDefaults(cfg)
	assert.Equal(t, "local", cfg.Spec.State.Backend)
}

func TestApplyDefaultsLifecycle(t *testing.T) {
	cfg := &ClusterConfig{}
	ApplyDefaults(cfg)
	assert.Equal(t, "rolling", cfg.Spec.Lifecycle.Upgrades.Strategy)
	assert.Equal(t, 1, cfg.Spec.Lifecycle.Upgrades.MaxUnavailable)
	assert.Equal(t, 7, cfg.Spec.Lifecycle.Etcd.BackupRetention)
}

func TestApplyDefaultsNodeFields(t *testing.T) {
	cfg := &ClusterConfig{
		Spec: ClusterSpec{
			ControlPlane: ControlPlane{
				Nodes: []Node{{Host: "10.0.0.1"}},
			},
			Workers: []NodePool{
				{Name: "pool-1", Nodes: []Node{{Host: "10.0.0.10"}}},
			},
		},
	}
	ApplyDefaults(cfg)
	assert.Equal(t, "root", cfg.Spec.ControlPlane.Nodes[0].User)
	assert.Equal(t, 22, cfg.Spec.ControlPlane.Nodes[0].Port)
	assert.Equal(t, "root", cfg.Spec.Workers[0].Nodes[0].User)
	assert.Equal(t, 22, cfg.Spec.Workers[0].Nodes[0].Port)
}

func TestApplyDefaultsPreservesExplicitValues(t *testing.T) {
	cfg := &ClusterConfig{
		Spec: ClusterSpec{
			Infrastructure: Infrastructure{Provider: "hetzner"},
			Security:       Security{Profile: "cis-hardened"},
			Networking: Networking{
				PodCIDR:     "10.244.0.0/16",
				ServiceCIDR: "10.96.0.0/12",
			},
			State: StateConfig{Backend: "s3"},
			Lifecycle: Lifecycle{
				Upgrades: UpgradeConfig{Strategy: "canary", MaxUnavailable: 2},
				Etcd:     EtcdConfig{BackupRetention: 30},
			},
			ControlPlane: ControlPlane{
				Nodes: []Node{{Host: "10.0.0.1", User: "ubuntu", Port: 2222}},
			},
		},
	}
	ApplyDefaults(cfg)
	assert.Equal(t, "hetzner", cfg.Spec.Infrastructure.Provider)
	assert.Equal(t, "cis-hardened", cfg.Spec.Security.Profile)
	assert.Equal(t, "10.244.0.0/16", cfg.Spec.Networking.PodCIDR)
	assert.Equal(t, "10.96.0.0/12", cfg.Spec.Networking.ServiceCIDR)
	assert.Equal(t, "s3", cfg.Spec.State.Backend)
	assert.Equal(t, "canary", cfg.Spec.Lifecycle.Upgrades.Strategy)
	assert.Equal(t, 2, cfg.Spec.Lifecycle.Upgrades.MaxUnavailable)
	assert.Equal(t, 30, cfg.Spec.Lifecycle.Etcd.BackupRetention)
	assert.Equal(t, "ubuntu", cfg.Spec.ControlPlane.Nodes[0].User)
	assert.Equal(t, 2222, cfg.Spec.ControlPlane.Nodes[0].Port)
}
