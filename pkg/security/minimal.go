package security

type Minimal struct{}

func (m *Minimal) Name() string {
	return "minimal"
}

func (m *Minimal) Description() string {
	return "Minimal security profile with basic RBAC and node restriction"
}

func (m *Minimal) APIServerFlags() []string {
	return []string{
		"--authorization-mode=RBAC,Node",
		"--enable-admission-plugins=NodeRestriction",
	}
}

func (m *Minimal) EtcdFlags() []string {
	return nil
}

func (m *Minimal) KubeletFlags() []string {
	return []string{
		"--read-only-port=0",
	}
}

func (m *Minimal) PSALabels() map[string]string {
	return map[string]string{
		"pod-security.kubernetes.io/enforce": "baseline",
		"pod-security.kubernetes.io/warn":    "restricted",
	}
}
