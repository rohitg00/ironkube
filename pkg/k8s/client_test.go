package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClientFromKubeconfig_InvalidBytes(t *testing.T) {
	_, err := NewClientFromKubeconfig([]byte("not-valid-yaml{{{"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing kubeconfig")
}

func TestNewClientFromKubeconfig_EmptyBytes(t *testing.T) {
	_, err := NewClientFromKubeconfig([]byte{})
	assert.Error(t, err)
}

func TestNewClientFromKubeconfig_NilBytes(t *testing.T) {
	_, err := NewClientFromKubeconfig(nil)
	assert.Error(t, err)
}

func TestNewClientFromKubeconfig_ValidStructureNoServer(t *testing.T) {
	kubeconfig := []byte(`
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: test
contexts:
- context:
    cluster: test
    user: test
  name: test
current-context: test
users:
- name: test
  user:
    token: fake-token
`)
	client, err := NewClientFromKubeconfig(kubeconfig)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}
