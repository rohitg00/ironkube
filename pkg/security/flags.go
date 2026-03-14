package security

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/config"
)

func FlagsForProfile(profileName string) (config.SecurityFlags, error) {
	if profileName == "" || profileName == "minimal" {
		return config.SecurityFlags{}, nil
	}

	profile, err := GetProfile(profileName)
	if err != nil {
		return config.SecurityFlags{}, fmt.Errorf("unknown security profile %q: %w", profileName, err)
	}

	return config.SecurityFlags{
		APIServer: profile.APIServerFlags(),
		Kubelet:   profile.KubeletFlags(),
		Etcd:      profile.EtcdFlags(),
	}, nil
}
