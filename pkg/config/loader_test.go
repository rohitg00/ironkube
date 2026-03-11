package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validYAML() string {
	return `apiVersion: ironkube.dev/v1alpha1
kind: Cluster
metadata:
  name: test-cluster
spec:
  distro: k3s
  version: "1.30.0"
  controlPlane:
    replicas: 1
    nodes:
      - host: 10.0.0.1
        keyPath: /home/user/.ssh/id_rsa
  workers:
    - name: pool-1
      nodes:
        - host: 10.0.0.2
          keyPath: /home/user/.ssh/id_rsa
  security:
    profile: minimal
  networking:
    cni: flannel
    podCIDR: "10.42.0.0/16"
    serviceCIDR: "10.43.0.0/16"
`
}

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "cluster.yaml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

func TestLoadValidConfig(t *testing.T) {
	path := writeTestFile(t, validYAML())
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "ironkube.dev/v1alpha1", cfg.APIVersion)
	assert.Equal(t, "Cluster", cfg.Kind)
	assert.Equal(t, "test-cluster", cfg.Metadata.Name)
	assert.Equal(t, "k3s", cfg.Spec.Distro)
	assert.Equal(t, "1.30.0", cfg.Spec.Version)
	assert.Equal(t, 1, cfg.Spec.ControlPlane.Replicas)
	assert.Len(t, cfg.Spec.ControlPlane.Nodes, 1)
	assert.Equal(t, "10.0.0.1", cfg.Spec.ControlPlane.Nodes[0].Host)
	assert.Len(t, cfg.Spec.Workers, 1)
	assert.Equal(t, "pool-1", cfg.Spec.Workers[0].Name)
	assert.Equal(t, "10.0.0.2", cfg.Spec.Workers[0].Nodes[0].Host)
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/cluster.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading config file")
}

func TestLoadInvalidYAML(t *testing.T) {
	path := writeTestFile(t, `{{{invalid yaml content`)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config file")
}

func TestLoadAppliesDefaults(t *testing.T) {
	yamlContent := `apiVersion: ironkube.dev/v1alpha1
kind: Cluster
metadata:
  name: defaults-test
spec:
  distro: k3s
  version: "1.30.0"
  controlPlane:
    replicas: 1
    nodes:
      - host: 10.0.0.1
`
	path := writeTestFile(t, yamlContent)
	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "root", cfg.Spec.ControlPlane.Nodes[0].User)
	assert.Equal(t, 22, cfg.Spec.ControlPlane.Nodes[0].Port)
	assert.Equal(t, "minimal", cfg.Spec.Security.Profile)
	assert.Equal(t, "flannel", cfg.Spec.Networking.CNI)
	assert.Equal(t, "10.42.0.0/16", cfg.Spec.Networking.PodCIDR)
	assert.Equal(t, "10.43.0.0/16", cfg.Spec.Networking.SvcCIDR)
}

func TestLoadPreservesExplicitValues(t *testing.T) {
	yamlContent := `apiVersion: ironkube.dev/v1alpha1
kind: Cluster
metadata:
  name: explicit-test
spec:
  distro: kubeadm
  version: "1.30.0"
  controlPlane:
    replicas: 1
    nodes:
      - host: 10.0.0.1
        user: admin
        port: 2222
  security:
    profile: cis-hardened
  networking:
    cni: cilium
    podCIDR: "192.168.0.0/16"
    serviceCIDR: "172.16.0.0/16"
`
	path := writeTestFile(t, yamlContent)
	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "admin", cfg.Spec.ControlPlane.Nodes[0].User)
	assert.Equal(t, 2222, cfg.Spec.ControlPlane.Nodes[0].Port)
	assert.Equal(t, "cis-hardened", cfg.Spec.Security.Profile)
	assert.Equal(t, "cilium", cfg.Spec.Networking.CNI)
	assert.Equal(t, "192.168.0.0/16", cfg.Spec.Networking.PodCIDR)
	assert.Equal(t, "172.16.0.0/16", cfg.Spec.Networking.SvcCIDR)
}

func TestLoadWorkerNodeDefaults(t *testing.T) {
	yamlContent := `apiVersion: ironkube.dev/v1alpha1
kind: Cluster
metadata:
  name: worker-defaults
spec:
  distro: k3s
  version: "1.30.0"
  controlPlane:
    replicas: 1
    nodes:
      - host: 10.0.0.1
  workers:
    - name: pool-1
      nodes:
        - host: 10.0.0.2
        - host: 10.0.0.3
          user: deploy
          port: 3333
`
	path := writeTestFile(t, yamlContent)
	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "root", cfg.Spec.Workers[0].Nodes[0].User)
	assert.Equal(t, 22, cfg.Spec.Workers[0].Nodes[0].Port)
	assert.Equal(t, "deploy", cfg.Spec.Workers[0].Nodes[1].User)
	assert.Equal(t, 3333, cfg.Spec.Workers[0].Nodes[1].Port)
}
