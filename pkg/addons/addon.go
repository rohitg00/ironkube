package addons

type Addon interface {
	Name() string
	InstallCmd() string
	WaitCmd() string
}

func Get(name string) (Addon, bool) {
	switch name {
	case "cilium":
		return &Cilium{}, true
	case "calico":
		return &Calico{}, true
	case "cert-manager":
		return &CertManager{}, true
	case "monitoring":
		return &Monitoring{}, true
	default:
		return nil, false
	}
}
