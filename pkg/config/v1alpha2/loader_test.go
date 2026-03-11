package v1alpha2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validYAML = `
apiVersion: ironkube.dev/v1alpha2
kind: Cluster
metadata:
  name: test-cluster
  labels:
    env: staging
spec:
  distro: k3s
  version: v1.32.0
  infrastructure:
    provider: ssh
    ssh:
      keyPath: ~/.ssh/id_rsa
  controlPlane:
    replicas: 3
    nodes:
      - host: 10.0.0.1
      - host: 10.0.0.2
      - host: 10.0.0.3
  workers:
    - name: general
      nodes:
        - host: 10.0.1.1
        - host: 10.0.1.2
      labels:
        tier: general
  networking:
    podCIDR: 10.42.0.0/16
    serviceCIDR: 10.43.0.0/16
  security:
    profile: cis-hardened
    encryptionAtRest: true
    auditLogging: true
  addons:
    - name: cilium
      enabled: true
      version: "1.16.0"
      namespace: kube-system
  lifecycle:
    certs:
      autoRotate: true
      rotateBeforeExpiry: "30d"
    etcd:
      backupSchedule: "0 2 * * *"
      backupRetention: 14
    upgrades:
      strategy: rolling
      maxUnavailable: 1
  state:
    backend: git
    git:
      repo: git@github.com:org/state.git
      branch: main
`

func TestLoadBytesValid(t *testing.T) {
	cfg, err := LoadBytes([]byte(validYAML))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "ironkube.dev/v1alpha2", cfg.APIVersion)
	assert.Equal(t, "Cluster", cfg.Kind)
	assert.Equal(t, "test-cluster", cfg.Metadata.Name)
	assert.Equal(t, "staging", cfg.Metadata.Labels["env"])
	assert.Equal(t, "k3s", cfg.Spec.Distro)
	assert.Equal(t, "v1.32.0", cfg.Spec.Version)
	assert.Equal(t, "ssh", cfg.Spec.Infrastructure.Provider)
	assert.Equal(t, 3, cfg.Spec.ControlPlane.Replicas)
	assert.Len(t, cfg.Spec.ControlPlane.Nodes, 3)
	assert.Equal(t, "root", cfg.Spec.ControlPlane.Nodes[0].User)
	assert.Equal(t, 22, cfg.Spec.ControlPlane.Nodes[0].Port)
	assert.Len(t, cfg.Spec.Workers, 1)
	assert.Equal(t, "general", cfg.Spec.Workers[0].Labels["tier"])
	assert.Equal(t, "cis-hardened", cfg.Spec.Security.Profile)
	assert.True(t, cfg.Spec.Security.EncryptionAtRest)
	assert.Len(t, cfg.Spec.Addons, 1)
	assert.Equal(t, "cilium", cfg.Spec.Addons[0].Name)
	assert.True(t, cfg.Spec.Lifecycle.Certs.AutoRotate)
	assert.Equal(t, 14, cfg.Spec.Lifecycle.Etcd.BackupRetention)
	assert.Equal(t, "git", cfg.Spec.State.Backend)
	require.NotNil(t, cfg.Spec.State.Git)
	assert.Equal(t, "main", cfg.Spec.State.Git.Branch)
}

func TestLoadBytesAppliesDefaults(t *testing.T) {
	minimalYAML := `
metadata:
  name: minimal
spec:
  distro: k3s
  version: v1.32.0
  controlPlane:
    replicas: 1
    nodes:
      - host: 10.0.0.1
`
	cfg, err := LoadBytes([]byte(minimalYAML))
	require.NoError(t, err)

	assert.Equal(t, "ssh", cfg.Spec.Infrastructure.Provider)
	assert.Equal(t, "minimal", cfg.Spec.Security.Profile)
	assert.Equal(t, "10.42.0.0/16", cfg.Spec.Networking.PodCIDR)
	assert.Equal(t, "10.43.0.0/16", cfg.Spec.Networking.ServiceCIDR)
	assert.Equal(t, "local", cfg.Spec.State.Backend)
	assert.Equal(t, "rolling", cfg.Spec.Lifecycle.Upgrades.Strategy)
	assert.Equal(t, 1, cfg.Spec.Lifecycle.Upgrades.MaxUnavailable)
	assert.Equal(t, 7, cfg.Spec.Lifecycle.Etcd.BackupRetention)
	assert.Equal(t, "root", cfg.Spec.ControlPlane.Nodes[0].User)
	assert.Equal(t, 22, cfg.Spec.ControlPlane.Nodes[0].Port)
}

func TestLoadBytesInvalidYAML(t *testing.T) {
	_, err := LoadBytes([]byte(`{{{invalid`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config YAML")
}

func TestLoadBytesValidationFailure(t *testing.T) {
	_, err := LoadBytes([]byte(`
metadata:
  name: ""
spec:
  distro: invalid
  version: v1.0.0
  controlPlane:
    replicas: 1
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cluster.yaml")
	require.NoError(t, os.WriteFile(path, []byte(validYAML), 0o644))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "test-cluster", cfg.Metadata.Name)
}

func TestLoadFromFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/cluster.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading config file")
}

func TestLoadBytesHetznerProvider(t *testing.T) {
	hetznerYAML := `
metadata:
  name: hetzner-cluster
spec:
  distro: k3s
  version: v1.32.0
  infrastructure:
    provider: hetzner
    hetzner:
      tokenEnv: HCLOUD_TOKEN
      location: fsn1
      serverType: cx21
  controlPlane:
    replicas: 1
  security:
    profile: minimal
`
	cfg, err := LoadBytes([]byte(hetznerYAML))
	require.NoError(t, err)
	assert.Equal(t, "hetzner", cfg.Spec.Infrastructure.Provider)
	require.NotNil(t, cfg.Spec.Hetzner())
	assert.Equal(t, "fsn1", cfg.Spec.Infrastructure.Hetzner.Location)
}

func (s *ClusterSpec) Hetzner() *HetznerConfig {
	return s.Infrastructure.Hetzner
}

func TestLoadBytesDockerProvider(t *testing.T) {
	dockerYAML := `
metadata:
  name: docker-cluster
spec:
  distro: k3s
  version: v1.32.0
  infrastructure:
    provider: docker
    docker:
      image: kindest/node:v1.32.0
  controlPlane:
    replicas: 1
  security:
    profile: minimal
`
	cfg, err := LoadBytes([]byte(dockerYAML))
	require.NoError(t, err)
	assert.Equal(t, "docker", cfg.Spec.Infrastructure.Provider)
	require.NotNil(t, cfg.Spec.Infrastructure.Docker)
	assert.Equal(t, "kindest/node:v1.32.0", cfg.Spec.Infrastructure.Docker.Image)
}
