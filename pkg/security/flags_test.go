package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagsForProfile_EmptyProfile(t *testing.T) {
	flags, err := FlagsForProfile("")
	require.NoError(t, err)
	assert.True(t, flags.IsEmpty())
}

func TestFlagsForProfile_MinimalCases(t *testing.T) {
	cases := []struct {
		profile string
	}{
		{"minimal"},
	}
	for _, tc := range cases {
		t.Run(tc.profile, func(t *testing.T) {
			flags, err := FlagsForProfile(tc.profile)
			require.NoError(t, err)
			assert.True(t, flags.IsEmpty())
		})
	}
}

func TestFlagsForProfile_UnknownProfile(t *testing.T) {
	_, err := FlagsForProfile("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown security profile")
}

func TestFlagsForProfile_CIS(t *testing.T) {
	flags, err := FlagsForProfile("cis")
	require.NoError(t, err)
	assert.False(t, flags.IsEmpty())

	assert.Contains(t, flags.APIServer, "--anonymous-auth=false")
	assert.Contains(t, flags.APIServer, "--profiling=false")
	assert.Contains(t, flags.Kubelet, "--protect-kernel-defaults=true")
	assert.Contains(t, flags.Kubelet, "--read-only-port=0")
	assert.Contains(t, flags.Etcd, "--client-cert-auth=true")
}

func TestFlagsForProfile_Hardened(t *testing.T) {
	flags, err := FlagsForProfile("hardened")
	require.NoError(t, err)
	assert.False(t, flags.IsEmpty())

	assert.Contains(t, flags.APIServer, "--encryption-provider-config=/etc/kubernetes/encryption.yaml")
	assert.Contains(t, flags.APIServer, "--tls-min-version=VersionTLS12")
	assert.Contains(t, flags.Kubelet, "--anonymous-auth=false")
	assert.Contains(t, flags.Etcd, "--cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")
}

func TestFlagsForProfile_ComponentSeparation(t *testing.T) {
	flags, err := FlagsForProfile("cis")
	require.NoError(t, err)

	for _, f := range flags.APIServer {
		assert.NotContains(t, f, "protect-kernel-defaults", "apiserver should not have kubelet flags")
	}
	for _, f := range flags.Kubelet {
		assert.NotContains(t, f, "anonymous-auth", "kubelet should not have apiserver flags")
	}
	for _, f := range flags.Etcd {
		assert.NotContains(t, f, "anonymous-auth", "etcd should not have apiserver flags")
	}
}
