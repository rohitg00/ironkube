package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_Minimal(t *testing.T) {
	p, err := Get("minimal")
	require.NoError(t, err)
	assert.Equal(t, "minimal", p.Name())
}

func TestGet_CISHardened(t *testing.T) {
	p, err := Get("cis-hardened")
	require.NoError(t, err)
	assert.Equal(t, "cis-hardened", p.Name())
}

func TestGet_Unknown(t *testing.T) {
	_, err := Get("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown security profile")
}

func TestMinimal_APIServerFlags(t *testing.T) {
	p := &Minimal{}
	flags := p.APIServerFlags()
	assert.NotEmpty(t, flags)
	assert.Contains(t, flags, "--authorization-mode=RBAC,Node")
	assert.Contains(t, flags, "--enable-admission-plugins=NodeRestriction")
}

func TestMinimal_KubeletFlags(t *testing.T) {
	p := &Minimal{}
	flags := p.KubeletFlags()
	assert.NotEmpty(t, flags)
	assert.Contains(t, flags, "--read-only-port=0")
}

func TestMinimal_PSALabels(t *testing.T) {
	p := &Minimal{}
	labels := p.PSALabels()
	assert.Equal(t, "baseline", labels["pod-security.kubernetes.io/enforce"])
	assert.Equal(t, "restricted", labels["pod-security.kubernetes.io/warn"])
}

func TestCISHardened_APIServerFlags(t *testing.T) {
	p := &CISHardened{}
	flags := p.APIServerFlags()
	assert.Contains(t, flags, "--anonymous-auth=false")
	assert.Contains(t, flags, "--authorization-mode=RBAC,Node")
	assert.Contains(t, flags, "--profiling=false")
	assert.Contains(t, flags, "--tls-min-version=VersionTLS12")
	assert.Contains(t, flags, "--audit-log-path=/var/log/kubernetes/audit.log")
	assert.Contains(t, flags, "--service-account-lookup=true")
}

func TestCISHardened_EtcdFlags(t *testing.T) {
	p := &CISHardened{}
	flags := p.EtcdFlags()
	assert.NotEmpty(t, flags)
	assert.Contains(t, flags, "--client-cert-auth=true")
	assert.Contains(t, flags, "--peer-client-cert-auth=true")
}

func TestCISHardened_KubeletFlags(t *testing.T) {
	p := &CISHardened{}
	flags := p.KubeletFlags()
	assert.Contains(t, flags, "--anonymous-auth=false")
	assert.Contains(t, flags, "--protect-kernel-defaults=true")
	assert.Contains(t, flags, "--rotate-certificates=true")
}

func TestCISHardened_PSALabels(t *testing.T) {
	p := &CISHardened{}
	labels := p.PSALabels()
	assert.Equal(t, "restricted", labels["pod-security.kubernetes.io/enforce"])
	assert.Equal(t, "latest", labels["pod-security.kubernetes.io/enforce-version"])
	assert.Equal(t, "restricted", labels["pod-security.kubernetes.io/warn"])
	assert.Equal(t, "restricted", labels["pod-security.kubernetes.io/audit"])
}

func TestCISHardened_MoreFlagsThanMinimal(t *testing.T) {
	minimal := &Minimal{}
	cis := &CISHardened{}
	assert.Greater(t, len(cis.APIServerFlags()), len(minimal.APIServerFlags()))
}

func TestAllProfiles_RequiredMethods(t *testing.T) {
	profiles := []Profile{&Minimal{}, &CISHardened{}}

	for _, p := range profiles {
		t.Run(p.Name(), func(t *testing.T) {
			assert.NotEmpty(t, p.Name())
			assert.NotNil(t, p.APIServerFlags())
			assert.NotNil(t, p.KubeletFlags())
			assert.NotNil(t, p.PSALabels())
			assert.NotEmpty(t, p.Description())
		})
	}
}
