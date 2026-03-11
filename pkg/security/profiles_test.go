package security

import (
	"testing"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinimalProfile_Name(t *testing.T) {
	p := &MinimalProfile{}
	assert.Equal(t, "minimal", p.Name())
}

func TestMinimalProfile_EmptyAPIServerFlags(t *testing.T) {
	p := &MinimalProfile{}
	assert.Empty(t, p.APIServerFlags())
}

func TestMinimalProfile_EmptyKubeletFlags(t *testing.T) {
	p := &MinimalProfile{}
	assert.Empty(t, p.KubeletFlags())
}

func TestMinimalProfile_EmptyEtcdFlags(t *testing.T) {
	p := &MinimalProfile{}
	assert.Empty(t, p.EtcdFlags())
}

func TestCISProfile_Name(t *testing.T) {
	p := &CISProfile{}
	assert.Equal(t, "cis", p.Name())
}

func TestCISProfile_APIServerFlags(t *testing.T) {
	p := &CISProfile{}
	flags := p.APIServerFlags()
	assert.Contains(t, flags, "--anonymous-auth=false")
	assert.Contains(t, flags, "--audit-log-path=/var/log/kubernetes/audit.log")
	assert.Contains(t, flags, "--audit-log-maxage=30")
	assert.Contains(t, flags, "--audit-log-maxbackup=10")
	assert.Contains(t, flags, "--audit-log-maxsize=100")
	assert.Contains(t, flags, "--authorization-mode=Node,RBAC")
	assert.Contains(t, flags, "--profiling=false")
	assert.Contains(t, flags, "--service-account-lookup=true")
	assert.Len(t, flags, 8)
}

func TestCISProfile_KubeletFlags(t *testing.T) {
	p := &CISProfile{}
	flags := p.KubeletFlags()
	assert.Contains(t, flags, "--protect-kernel-defaults=true")
	assert.Contains(t, flags, "--read-only-port=0")
	assert.Contains(t, flags, "--streaming-connection-idle-timeout=5m")
	assert.Contains(t, flags, "--make-iptables-util-chains=true")
	assert.Len(t, flags, 4)
}

func TestCISProfile_EtcdFlags(t *testing.T) {
	p := &CISProfile{}
	flags := p.EtcdFlags()
	assert.Contains(t, flags, "--client-cert-auth=true")
	assert.Contains(t, flags, "--peer-client-cert-auth=true")
	assert.Contains(t, flags, "--auto-tls=false")
	assert.Len(t, flags, 3)
}

func TestHardenedProfile_Name(t *testing.T) {
	p := &HardenedProfile{}
	assert.Equal(t, "hardened", p.Name())
}

func TestHardenedProfile_MoreAPIServerFlagsThanCIS(t *testing.T) {
	cis := &CISProfile{}
	hardened := &HardenedProfile{}
	assert.Greater(t, len(hardened.APIServerFlags()), len(cis.APIServerFlags()))
}

func TestHardenedProfile_IncludesAllCISAPIServerFlags(t *testing.T) {
	cis := &CISProfile{}
	hardened := &HardenedProfile{}
	for _, flag := range cis.APIServerFlags() {
		assert.Contains(t, hardened.APIServerFlags(), flag)
	}
}

func TestHardenedProfile_IncludesAllCISKubeletFlags(t *testing.T) {
	cis := &CISProfile{}
	hardened := &HardenedProfile{}
	for _, flag := range cis.KubeletFlags() {
		assert.Contains(t, hardened.KubeletFlags(), flag)
	}
}

func TestHardenedProfile_IncludesAllCISEtcdFlags(t *testing.T) {
	cis := &CISProfile{}
	hardened := &HardenedProfile{}
	for _, flag := range cis.EtcdFlags() {
		assert.Contains(t, hardened.EtcdFlags(), flag)
	}
}

func TestHardenedProfile_HasEncryptionProviderFlag(t *testing.T) {
	p := &HardenedProfile{}
	assert.Contains(t, p.APIServerFlags(), "--encryption-provider-config=/etc/kubernetes/encryption.yaml")
}

func TestHardenedProfile_HasTLSMinVersion(t *testing.T) {
	p := &HardenedProfile{}
	assert.Contains(t, p.APIServerFlags(), "--tls-min-version=VersionTLS12")
}

func TestHardenedProfile_HasCipherSuites(t *testing.T) {
	p := &HardenedProfile{}
	assert.Contains(t, p.EtcdFlags(), "--cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")
}

func TestGetProfile_Minimal(t *testing.T) {
	p, err := GetProfile("minimal")
	require.NoError(t, err)
	assert.Equal(t, "minimal", p.Name())
	_, ok := p.(*MinimalProfile)
	assert.True(t, ok)
}

func TestGetProfile_CIS(t *testing.T) {
	p, err := GetProfile("cis")
	require.NoError(t, err)
	assert.Equal(t, "cis", p.Name())
	_, ok := p.(*CISProfile)
	assert.True(t, ok)
}

func TestGetProfile_Hardened(t *testing.T) {
	p, err := GetProfile("hardened")
	require.NoError(t, err)
	assert.Equal(t, "hardened", p.Name())
	_, ok := p.(*HardenedProfile)
	assert.True(t, ok)
}

func TestGetProfile_Unknown(t *testing.T) {
	_, err := GetProfile("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown profile")
}

func TestAllProviderProfiles_ImplementInterface(t *testing.T) {
	profiles := []provider.SecurityProfile{
		&MinimalProfile{},
		&CISProfile{},
		&HardenedProfile{},
	}

	for _, p := range profiles {
		t.Run(p.Name(), func(t *testing.T) {
			assert.NotEmpty(t, p.Name())
			assert.NotNil(t, p.APIServerFlags())
			assert.NotNil(t, p.KubeletFlags())
			assert.NotNil(t, p.EtcdFlags())
		})
	}
}
