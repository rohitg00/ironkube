package kubeconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

var sampleKubeconfig = []byte(`apiVersion: v1
kind: Config
clusters:
- name: default
  cluster:
    server: https://127.0.0.1:6443
    certificate-authority-data: dGVzdA==
contexts:
- name: default
  context:
    cluster: default
    user: default-admin
users:
- name: default-admin
  user:
    client-certificate-data: dGVzdA==
    client-key-data: dGVzdA==
current-context: default
`)

func TestRewriteServer_ReplacesLocalhost(t *testing.T) {
	input := []byte(`apiVersion: v1
kind: Config
clusters:
- name: default
  cluster:
    server: https://127.0.0.1:6443
contexts:
- name: default
  context:
    cluster: default
    user: default-admin
users:
- name: default-admin
  user: {}
current-context: default
`)

	out, err := RewriteServer(input, "10.0.1.5", "prod")
	require.NoError(t, err)

	var cfg Config
	require.NoError(t, yaml.Unmarshal(out, &cfg))

	assert.Equal(t, "https://10.0.1.5:6443", cfg.Clusters[0].Cluster.Server)
}

func TestRewriteServer_RenamesCluster(t *testing.T) {
	out, err := RewriteServer(sampleKubeconfig, "10.0.1.5", "my-cluster")
	require.NoError(t, err)

	var cfg Config
	require.NoError(t, yaml.Unmarshal(out, &cfg))

	assert.Equal(t, "my-cluster", cfg.Clusters[0].Name)
	assert.Equal(t, "my-cluster", cfg.Contexts[0].Name)
	assert.Equal(t, "my-cluster", cfg.Contexts[0].Context.Cluster)
	assert.Equal(t, "my-cluster-admin", cfg.Users[0].Name)
	assert.Equal(t, "my-cluster", cfg.CurrentContext)
}

func TestRewriteServer_InvalidYAML(t *testing.T) {
	_, err := RewriteServer([]byte("not: [valid: yaml: {{"), "host", "name")
	assert.Error(t, err)
}

func TestSaveToFile_WritesWithCorrectPerms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kubeconfig")

	err := SaveToFile(sampleKubeconfig, path)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, sampleKubeconfig, data)
}

func TestSaveToFile_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "kubeconfig")

	err := SaveToFile(sampleKubeconfig, path)
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestMerge_CreatesFileIfNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kubeconfig")

	err := Merge(path, sampleKubeconfig)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var cfg Config
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	assert.Len(t, cfg.Clusters, 1)
}

func TestMerge_AddsNewCluster(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kubeconfig")

	err := SaveToFile(sampleKubeconfig, path)
	require.NoError(t, err)

	newKubeconfig := []byte(`apiVersion: v1
kind: Config
clusters:
- name: staging
  cluster:
    server: https://10.0.2.1:6443
contexts:
- name: staging
  context:
    cluster: staging
    user: staging-admin
users:
- name: staging-admin
  user:
    token: test-token
current-context: staging
`)

	err = Merge(path, newKubeconfig)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var cfg Config
	require.NoError(t, yaml.Unmarshal(data, &cfg))

	assert.Len(t, cfg.Clusters, 2)
	assert.Len(t, cfg.Contexts, 2)
	assert.Len(t, cfg.Users, 2)
	assert.Equal(t, "staging", cfg.CurrentContext)

	clusterNames := make(map[string]bool)
	for _, c := range cfg.Clusters {
		clusterNames[c.Name] = true
	}
	assert.True(t, clusterNames["default"])
	assert.True(t, clusterNames["staging"])
}

func TestMerge_DoesNotDuplicateExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kubeconfig")

	err := SaveToFile(sampleKubeconfig, path)
	require.NoError(t, err)

	err = Merge(path, sampleKubeconfig)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var cfg Config
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	assert.Len(t, cfg.Clusters, 1)
	assert.Len(t, cfg.Contexts, 1)
	assert.Len(t, cfg.Users, 1)
}
