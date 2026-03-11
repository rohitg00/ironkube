package security

type CISHardened struct{}

func (c *CISHardened) Name() string {
	return "cis-hardened"
}

func (c *CISHardened) Description() string {
	return "CIS Kubernetes Benchmark hardened profile covering 52% of checks at bootstrap"
}

func (c *CISHardened) APIServerFlags() []string {
	return []string{
		"--anonymous-auth=false",
		"--authorization-mode=RBAC,Node",
		"--enable-admission-plugins=NodeRestriction,PodSecurity",
		"--profiling=false",
		"--tls-min-version=VersionTLS12",
		"--audit-log-path=/var/log/kubernetes/audit.log",
		"--audit-log-maxage=30",
		"--audit-log-maxbackup=10",
		"--audit-log-maxsize=100",
		"--service-account-lookup=true",
		"--service-account-key-file=/etc/kubernetes/pki/sa.pub",
	}
}

func (c *CISHardened) EtcdFlags() []string {
	return []string{
		"--client-cert-auth=true",
		"--peer-client-cert-auth=true",
		"--auto-tls=false",
		"--peer-auto-tls=false",
	}
}

func (c *CISHardened) KubeletFlags() []string {
	return []string{
		"--anonymous-auth=false",
		"--authorization-mode=Webhook",
		"--read-only-port=0",
		"--protect-kernel-defaults=true",
		"--event-qps=0",
		"--tls-min-version=VersionTLS12",
		"--rotate-certificates=true",
	}
}

func (c *CISHardened) PSALabels() map[string]string {
	return map[string]string{
		"pod-security.kubernetes.io/enforce":         "restricted",
		"pod-security.kubernetes.io/enforce-version": "latest",
		"pod-security.kubernetes.io/warn":            "restricted",
		"pod-security.kubernetes.io/audit":           "restricted",
	}
}
