package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestClusterConfigConstruction(t *testing.T) {
	cfg := ClusterConfig{
		APIVersion: "ironkube.dev/v1alpha2",
		Kind:       "Cluster",
		Metadata:   Metadata{Name: "test"},
		Spec: ClusterSpec{
			Distro:  "k3s",
			Version: "v1.32.0",
		},
	}
	assert.Equal(t, "ironkube.dev/v1alpha2", cfg.APIVersion)
	assert.Equal(t, "Cluster", cfg.Kind)
	assert.Equal(t, "test", cfg.Metadata.Name)
	assert.Equal(t, "k3s", cfg.Spec.Distro)
}

func TestMetadataWithLabels(t *testing.T) {
	m := Metadata{
		Name: "prod-cluster",
		Labels: map[string]string{
			"env":    "production",
			"region": "eu-west-1",
		},
	}
	assert.Equal(t, "prod-cluster", m.Name)
	assert.Equal(t, "production", m.Labels["env"])
	assert.Equal(t, "eu-west-1", m.Labels["region"])
}

func TestInfrastructureSSH(t *testing.T) {
	infra := Infrastructure{
		Provider: "ssh",
		SSH:      &SSHConfig{KeyPath: "~/.ssh/id_rsa"},
	}
	assert.Equal(t, "ssh", infra.Provider)
	require.NotNil(t, infra.SSH)
	assert.Equal(t, "~/.ssh/id_rsa", infra.SSH.KeyPath)
	assert.Nil(t, infra.Hetzner)
	assert.Nil(t, infra.Docker)
}

func TestInfrastructureHetzner(t *testing.T) {
	infra := Infrastructure{
		Provider: "hetzner",
		Hetzner: &HetznerConfig{
			TokenEnv:   "HCLOUD_TOKEN",
			Location:   "fsn1",
			ServerType: "cx21",
			Image:      "ubuntu-22.04",
			SSHKeyName: "my-key",
		},
	}
	assert.Equal(t, "hetzner", infra.Provider)
	require.NotNil(t, infra.Hetzner)
	assert.Equal(t, "HCLOUD_TOKEN", infra.Hetzner.TokenEnv)
	assert.Equal(t, "fsn1", infra.Hetzner.Location)
}

func TestInfrastructureDocker(t *testing.T) {
	infra := Infrastructure{
		Provider: "docker",
		Docker: &DockerConfig{
			Image:   "kindest/node:v1.32.0",
			Network: "kind",
		},
	}
	assert.Equal(t, "docker", infra.Provider)
	require.NotNil(t, infra.Docker)
	assert.Equal(t, "kindest/node:v1.32.0", infra.Docker.Image)
}

func TestControlPlaneWithNodes(t *testing.T) {
	cp := ControlPlane{
		Replicas: 3,
		Nodes: []Node{
			{Host: "10.0.0.1", User: "root", Port: 22, KeyPath: "/keys/id"},
			{Host: "10.0.0.2", User: "root", Port: 22},
			{Host: "10.0.0.3", User: "root", Port: 22},
		},
	}
	assert.Equal(t, 3, cp.Replicas)
	assert.Len(t, cp.Nodes, 3)
	assert.Equal(t, "10.0.0.1", cp.Nodes[0].Host)
}

func TestNodePoolWithTaints(t *testing.T) {
	pool := NodePool{
		Name:  "gpu-pool",
		Count: 2,
		Labels: map[string]string{
			"accelerator": "nvidia-a100",
		},
		Taints: []TaintConfig{
			{Key: "nvidia.com/gpu", Effect: "NoSchedule"},
		},
	}
	assert.Equal(t, "gpu-pool", pool.Name)
	assert.Equal(t, 2, pool.Count)
	assert.Equal(t, "nvidia-a100", pool.Labels["accelerator"])
	require.Len(t, pool.Taints, 1)
	assert.Equal(t, "NoSchedule", pool.Taints[0].Effect)
}

func TestAddonConfig(t *testing.T) {
	addon := AddonConfig{
		Name:       "cilium",
		Enabled:    true,
		Version:    "1.16.0",
		Namespace:  "kube-system",
		Repository: "https://helm.cilium.io",
		Chart:      "cilium",
		Values: map[string]any{
			"hubble": map[string]any{"enabled": true},
		},
	}
	assert.Equal(t, "cilium", addon.Name)
	assert.True(t, addon.Enabled)
	assert.Equal(t, "1.16.0", addon.Version)
	hubble := addon.Values["hubble"].(map[string]any)
	assert.Equal(t, true, hubble["enabled"])
}

func TestLifecycleConfig(t *testing.T) {
	lc := Lifecycle{
		Certs: CertsConfig{
			AutoRotate:         true,
			RotateBeforeExpiry: "30d",
		},
		Etcd: EtcdConfig{
			BackupSchedule:     "0 2 * * *",
			BackupRetention:    14,
			CompactionInterval: "5m",
		},
		Upgrades: UpgradeConfig{
			Strategy:       "rolling",
			MaxUnavailable: 1,
		},
	}
	assert.True(t, lc.Certs.AutoRotate)
	assert.Equal(t, "30d", lc.Certs.RotateBeforeExpiry)
	assert.Equal(t, 14, lc.Etcd.BackupRetention)
	assert.Equal(t, "rolling", lc.Upgrades.Strategy)
}

func TestStateConfigGit(t *testing.T) {
	sc := StateConfig{
		Backend: "git",
		Git: &GitConfig{
			Repo:   "git@github.com:org/state.git",
			Branch: "main",
		},
	}
	assert.Equal(t, "git", sc.Backend)
	require.NotNil(t, sc.Git)
	assert.Equal(t, "main", sc.Git.Branch)
	assert.Nil(t, sc.S3)
}

func TestStateConfigS3(t *testing.T) {
	sc := StateConfig{
		Backend: "s3",
		S3: &S3Config{
			Bucket: "ironkube-state",
			Region: "us-east-1",
			Prefix: "clusters/",
		},
	}
	assert.Equal(t, "s3", sc.Backend)
	require.NotNil(t, sc.S3)
	assert.Equal(t, "ironkube-state", sc.S3.Bucket)
	assert.Equal(t, "clusters/", sc.S3.Prefix)
}

func TestClusterConfigYAMLRoundTrip(t *testing.T) {
	original := ClusterConfig{
		APIVersion: "ironkube.dev/v1alpha2",
		Kind:       "Cluster",
		Metadata:   Metadata{Name: "roundtrip-test"},
		Spec: ClusterSpec{
			Distro:  "kubeadm",
			Version: "v1.31.0",
			Infrastructure: Infrastructure{
				Provider: "ssh",
			},
			ControlPlane: ControlPlane{Replicas: 1},
			Networking: Networking{
				PodCIDR:     "10.42.0.0/16",
				ServiceCIDR: "10.43.0.0/16",
			},
			Security: Security{Profile: "minimal"},
		},
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var parsed ClusterConfig
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, original.APIVersion, parsed.APIVersion)
	assert.Equal(t, original.Metadata.Name, parsed.Metadata.Name)
	assert.Equal(t, original.Spec.Distro, parsed.Spec.Distro)
	assert.Equal(t, original.Spec.Infrastructure.Provider, parsed.Spec.Infrastructure.Provider)
}

func TestSecurityConfig(t *testing.T) {
	s := Security{
		Profile:          "cis-hardened",
		EncryptionAtRest: true,
		AuditLogging:     true,
	}
	assert.Equal(t, "cis-hardened", s.Profile)
	assert.True(t, s.EncryptionAtRest)
	assert.True(t, s.AuditLogging)
}

func TestNetworkingConfig(t *testing.T) {
	n := Networking{
		PodCIDR:     "10.244.0.0/16",
		ServiceCIDR: "10.96.0.0/12",
	}
	assert.Equal(t, "10.244.0.0/16", n.PodCIDR)
	assert.Equal(t, "10.96.0.0/12", n.ServiceCIDR)
}
