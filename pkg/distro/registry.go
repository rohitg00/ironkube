package distro

import (
	"fmt"

	"github.com/rohitg00/ironkube/pkg/distro/k3s"
	"github.com/rohitg00/ironkube/pkg/distro/kubeadm"
)

func Get(name string) (Plugin, error) {
	switch name {
	case "k3s":
		return k3s.New(), nil
	case "kubeadm":
		return kubeadm.New(), nil
	default:
		return nil, fmt.Errorf("unsupported distro: %s", name)
	}
}
