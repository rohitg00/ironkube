package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagsForDistro_EmptyProfile(t *testing.T) {
	flags, err := FlagsForDistro("", "k3s")
	require.NoError(t, err)
	assert.Nil(t, flags)
}

func TestFlagsForDistro_MinimalProfile(t *testing.T) {
	flags, err := FlagsForDistro("minimal", "k3s")
	require.NoError(t, err)
	assert.Nil(t, flags)
}

func TestFlagsForDistro_UnknownProfile(t *testing.T) {
	_, err := FlagsForDistro("nonexistent", "k3s")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown security profile")
}

func TestFlagsForDistro_UnknownDistro(t *testing.T) {
	_, err := FlagsForDistro("cis", "rke2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported distro")
}

func TestFlagsForDistro_CISK3s(t *testing.T) {
	flags, err := FlagsForDistro("cis", "k3s")
	require.NoError(t, err)
	assert.NotEmpty(t, flags)

	assert.Contains(t, flags, "--kubelet-arg=protect-kernel-defaults=true")
	assert.Contains(t, flags, "--kubelet-arg=read-only-port=0")
	assert.Contains(t, flags, "--kube-apiserver-arg=anonymous-auth=false")
	assert.Contains(t, flags, "--kube-apiserver-arg=profiling=false")
	assert.Contains(t, flags, "--etcd-arg=client-cert-auth=true")
}

func TestFlagsForDistro_CISKubeadm(t *testing.T) {
	flags, err := FlagsForDistro("cis", "kubeadm")
	require.NoError(t, err)
	assert.NotEmpty(t, flags)

	assert.Contains(t, flags, "--anonymous-auth=false")
	assert.Contains(t, flags, "--protect-kernel-defaults=true")
	assert.Contains(t, flags, "--client-cert-auth=true")
}

func TestFlagsForDistro_HardenedK3s(t *testing.T) {
	flags, err := FlagsForDistro("hardened", "k3s")
	require.NoError(t, err)
	assert.NotEmpty(t, flags)

	assert.Contains(t, flags, "--kube-apiserver-arg=encryption-provider-config=/etc/kubernetes/encryption.yaml")
	assert.Contains(t, flags, "--kube-apiserver-arg=tls-min-version=VersionTLS12")
	assert.Contains(t, flags, "--kubelet-arg=anonymous-auth=false")
	assert.Contains(t, flags, "--etcd-arg=cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")
}

func TestFlagsForDistro_HardenedKubeadm(t *testing.T) {
	flags, err := FlagsForDistro("hardened", "kubeadm")
	require.NoError(t, err)
	assert.NotEmpty(t, flags)

	assert.Contains(t, flags, "--encryption-provider-config=/etc/kubernetes/encryption.yaml")
	assert.Contains(t, flags, "--tls-min-version=VersionTLS12")
	assert.Contains(t, flags, "--anonymous-auth=false")
	assert.Contains(t, flags, "--cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")
}

func TestFlagsForDistro_K3sFlagWrapping(t *testing.T) {
	flags, err := FlagsForDistro("cis", "k3s")
	require.NoError(t, err)

	for _, f := range flags {
		hasPrefix := false
		for _, prefix := range []string{"--kubelet-arg=", "--kube-apiserver-arg=", "--etcd-arg="} {
			if len(f) > len(prefix) && f[:len(prefix)] == prefix {
				hasPrefix = true
				break
			}
		}
		assert.True(t, hasPrefix, "k3s flag %q must be wrapped with a k3s-specific prefix", f)
	}
}

func TestFlagsForDistro_KubeadmFlagsPassedRaw(t *testing.T) {
	flags, err := FlagsForDistro("cis", "kubeadm")
	require.NoError(t, err)

	for _, f := range flags {
		assert.NotContains(t, f, "kubelet-arg=")
		assert.NotContains(t, f, "kube-apiserver-arg=")
		assert.NotContains(t, f, "etcd-arg=")
	}
}

func TestFlagsForDistro_MinimalK3sReturnsNil(t *testing.T) {
	flags, err := FlagsForDistro("minimal", "k3s")
	require.NoError(t, err)
	assert.Nil(t, flags)
}

func TestFlagsForDistro_MinimalKubeadmReturnsNil(t *testing.T) {
	flags, err := FlagsForDistro("minimal", "kubeadm")
	require.NoError(t, err)
	assert.Nil(t, flags)
}
