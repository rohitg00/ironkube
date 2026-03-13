package security

import (
	"strings"

	"github.com/rohitg00/ironkube/pkg/provider"
)

func FlagsForDistro(profileName, distroName string) []string {
	if profileName == "" || profileName == "minimal" {
		return nil
	}

	profile, err := GetProfile(profileName)
	if err != nil {
		return nil
	}

	switch distroName {
	case "k3s":
		return k3sFlags(profile)
	case "kubeadm":
		return kubeadmFlags(profile)
	default:
		return nil
	}
}

func k3sFlags(profile provider.SecurityProfile) []string {
	var flags []string

	for _, f := range profile.KubeletFlags() {
		flags = append(flags, toK3sFlag("kubelet-arg", f))
	}

	for _, f := range profile.APIServerFlags() {
		flags = append(flags, toK3sFlag("kube-apiserver-arg", f))
	}

	for _, f := range profile.EtcdFlags() {
		flags = append(flags, toK3sFlag("etcd-arg", f))
	}

	return flags
}

func toK3sFlag(wrapper, raw string) string {
	trimmed := strings.TrimPrefix(raw, "--")
	return "--" + wrapper + "=" + trimmed
}

func kubeadmFlags(profile provider.SecurityProfile) []string {
	var flags []string
	flags = append(flags, profile.APIServerFlags()...)
	flags = append(flags, profile.KubeletFlags()...)
	flags = append(flags, profile.EtcdFlags()...)
	return flags
}
