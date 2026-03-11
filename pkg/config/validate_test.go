package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() *ClusterConfig {
	return &ClusterConfig{
		APIVersion: "ironkube.dev/v1alpha1",
		Kind:       "Cluster",
		Metadata:   Metadata{Name: "test-cluster"},
		Spec: ClusterSpec{
			Distro:  "k3s",
			Version: "1.30.0",
			ControlPlane: ControlPlane{
				Replicas: 1,
				Nodes: []Node{
					{Host: "10.0.0.1", User: "root", Port: 22},
				},
			},
			Workers: []NodePool{
				{
					Name: "pool-1",
					Nodes: []Node{
						{Host: "10.0.0.2", User: "root", Port: 22},
					},
				},
			},
			Security: Security{Profile: "minimal"},
			Networking: Networking{
				CNI:     "flannel",
				PodCIDR: "10.42.0.0/16",
				SvcCIDR: "10.43.0.0/16",
			},
		},
	}
}

func TestValidateValidConfig(t *testing.T) {
	cfg := validConfig()
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidateK3sDistro(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Distro = "k3s"
	assert.NoError(t, Validate(cfg))
}

func TestValidateKubeadmDistro(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Distro = "kubeadm"
	assert.NoError(t, Validate(cfg))
}

func TestValidateMissingName(t *testing.T) {
	cfg := validConfig()
	cfg.Metadata.Name = ""
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name is required")
}

func TestValidateInvalidDistro(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Distro = "rke2"
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "distro must be k3s or kubeadm")
}

func TestValidateEmptyDistro(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Distro = ""
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "distro must be k3s or kubeadm")
}

func TestValidateMissingVersion(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Version = ""
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.version is required")
}

func TestValidateNoControlPlaneNodes(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.ControlPlane.Nodes = nil
	cfg.Spec.ControlPlane.Replicas = 0
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 1 control plane node is required")
}

func TestValidateHAEvenReplicas(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.ControlPlane.Replicas = 2
	cfg.Spec.ControlPlane.Nodes = []Node{
		{Host: "10.0.0.1", User: "root", Port: 22},
		{Host: "10.0.0.2", User: "root", Port: 22},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HA requires odd number of replicas")
}

func TestValidateHAOddReplicas(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.ControlPlane.Replicas = 3
	cfg.Spec.ControlPlane.Nodes = []Node{
		{Host: "10.0.0.1", User: "root", Port: 22},
		{Host: "10.0.0.2", User: "root", Port: 22},
		{Host: "10.0.0.3", User: "root", Port: 22},
	}
	assert.NoError(t, Validate(cfg))
}

func TestValidateHA5Replicas(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.ControlPlane.Replicas = 5
	cfg.Spec.ControlPlane.Nodes = []Node{
		{Host: "10.0.0.1", User: "root", Port: 22},
		{Host: "10.0.0.2", User: "root", Port: 22},
		{Host: "10.0.0.3", User: "root", Port: 22},
		{Host: "10.0.0.4", User: "root", Port: 22},
		{Host: "10.0.0.5", User: "root", Port: 22},
	}
	assert.NoError(t, Validate(cfg))
}

func TestValidateReplicaMismatch(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.ControlPlane.Replicas = 3
	cfg.Spec.ControlPlane.Nodes = []Node{
		{Host: "10.0.0.1", User: "root", Port: 22},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "replicas (3) must match control plane node count (1)")
}

func TestValidateInvalidProfile(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Security.Profile = "hardened"
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "security profile must be minimal or cis-hardened")
}

func TestValidateCISHardenedProfile(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Security.Profile = "cis-hardened"
	assert.NoError(t, Validate(cfg))
}

func TestValidateInvalidCNI(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Networking.CNI = "weave"
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CNI must be flannel, cilium, or calico")
}

func TestValidateCiliumCNI(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Networking.CNI = "cilium"
	assert.NoError(t, Validate(cfg))
}

func TestValidateCalicoCNI(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Networking.CNI = "calico"
	assert.NoError(t, Validate(cfg))
}

func TestValidateMissingControlPlaneHost(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.ControlPlane.Nodes[0].Host = ""
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "controlPlane.nodes[0].host is required")
}

func TestValidateMissingWorkerHost(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Workers[0].Nodes[0].Host = ""
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workers[0].nodes[0].host is required")
}

func TestValidateValidCIDRs(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Networking.PodCIDR = "192.168.0.0/16"
	cfg.Spec.Networking.SvcCIDR = "172.16.0.0/12"
	assert.NoError(t, Validate(cfg))
}

func TestValidateInvalidPodCIDR(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Networking.PodCIDR = "not-a-cidr"
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid podCIDR")
}

func TestValidateInvalidSvcCIDR(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Networking.SvcCIDR = "999.999.999.999/99"
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid serviceCIDR")
}

func TestValidateMultipleErrors(t *testing.T) {
	cfg := &ClusterConfig{}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name is required")
	assert.Contains(t, err.Error(), "spec.version is required")
	assert.Contains(t, err.Error(), "at least 1 control plane node is required")
}

func TestValidateHA4Replicas(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.ControlPlane.Replicas = 4
	cfg.Spec.ControlPlane.Nodes = []Node{
		{Host: "10.0.0.1", User: "root", Port: 22},
		{Host: "10.0.0.2", User: "root", Port: 22},
		{Host: "10.0.0.3", User: "root", Port: 22},
		{Host: "10.0.0.4", User: "root", Port: 22},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HA requires odd number of replicas, got 4")
}

func TestValidateSingleReplicaNotHA(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.ControlPlane.Replicas = 1
	cfg.Spec.ControlPlane.Nodes = []Node{
		{Host: "10.0.0.1", User: "root", Port: 22},
	}
	assert.NoError(t, Validate(cfg))
}

func TestValidateMultipleWorkerPoolsMissingHost(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Workers = []NodePool{
		{
			Name: "pool-1",
			Nodes: []Node{
				{Host: "10.0.0.2", User: "root", Port: 22},
			},
		},
		{
			Name: "pool-2",
			Nodes: []Node{
				{Host: "", User: "root", Port: 22},
			},
		},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workers[1].nodes[0].host is required")
}
