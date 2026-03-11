package security

import "fmt"

type Profile interface {
	Name() string
	APIServerFlags() []string
	EtcdFlags() []string
	KubeletFlags() []string
	PSALabels() map[string]string
	Description() string
}

func Get(name string) (Profile, error) {
	switch name {
	case "minimal":
		return &Minimal{}, nil
	case "cis-hardened":
		return &CISHardened{}, nil
	default:
		return nil, fmt.Errorf("unknown security profile: %s", name)
	}
}
