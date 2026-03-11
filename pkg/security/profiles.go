package security

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/provider"
)

var (
	_ provider.SecurityProfile = (*MinimalProfile)(nil)
	_ provider.SecurityProfile = (*CISProfile)(nil)
	_ provider.SecurityProfile = (*HardenedProfile)(nil)
)

type MinimalProfile struct{}

func (p *MinimalProfile) Name() string {
	return "minimal"
}

func (p *MinimalProfile) APIServerFlags() []string {
	return []string{}
}

func (p *MinimalProfile) KubeletFlags() []string {
	return []string{}
}

func (p *MinimalProfile) EtcdFlags() []string {
	return []string{}
}

type CISProfile struct{}

func (p *CISProfile) Name() string {
	return "cis"
}

func (p *CISProfile) APIServerFlags() []string {
	return []string{
		"--anonymous-auth=false",
		"--audit-log-path=/var/log/kubernetes/audit.log",
		"--audit-log-maxage=30",
		"--audit-log-maxbackup=10",
		"--audit-log-maxsize=100",
		"--authorization-mode=Node,RBAC",
		"--profiling=false",
		"--service-account-lookup=true",
	}
}

func (p *CISProfile) KubeletFlags() []string {
	return []string{
		"--protect-kernel-defaults=true",
		"--read-only-port=0",
		"--streaming-connection-idle-timeout=5m",
		"--make-iptables-util-chains=true",
	}
}

func (p *CISProfile) EtcdFlags() []string {
	return []string{
		"--client-cert-auth=true",
		"--peer-client-cert-auth=true",
		"--auto-tls=false",
	}
}

type HardenedProfile struct{}

func (p *HardenedProfile) Name() string {
	return "hardened"
}

func (p *HardenedProfile) APIServerFlags() []string {
	cis := &CISProfile{}
	flags := make([]string, len(cis.APIServerFlags()))
	copy(flags, cis.APIServerFlags())
	flags = append(flags,
		"--enable-admission-plugins=NodeRestriction,PodSecurity",
		"--encryption-provider-config=/etc/kubernetes/encryption.yaml",
		"--tls-min-version=VersionTLS12",
	)
	return flags
}

func (p *HardenedProfile) KubeletFlags() []string {
	cis := &CISProfile{}
	flags := make([]string, len(cis.KubeletFlags()))
	copy(flags, cis.KubeletFlags())
	flags = append(flags,
		"--anonymous-auth=false",
		"--event-qps=0",
	)
	return flags
}

func (p *HardenedProfile) EtcdFlags() []string {
	cis := &CISProfile{}
	flags := make([]string, len(cis.EtcdFlags()))
	copy(flags, cis.EtcdFlags())
	flags = append(flags,
		"--cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	)
	return flags
}

func GetProfile(name string) (provider.SecurityProfile, error) {
	switch name {
	case "minimal":
		return &MinimalProfile{}, nil
	case "cis":
		return &CISProfile{}, nil
	case "hardened":
		return &HardenedProfile{}, nil
	default:
		return nil, fmt.Errorf("unknown profile: %q, valid profiles are [minimal, cis, hardened]", name)
	}
}
