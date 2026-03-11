package distro

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetK3s(t *testing.T) {
	p, err := Get("k3s")
	require.NoError(t, err)
	assert.Equal(t, "k3s", p.Name())
}

func TestGetKubeadm(t *testing.T) {
	p, err := Get("kubeadm")
	require.NoError(t, err)
	assert.Equal(t, "kubeadm", p.Name())
}

func TestGetUnknown(t *testing.T) {
	_, err := Get("unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported distro")
}
