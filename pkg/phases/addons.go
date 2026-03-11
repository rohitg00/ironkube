package phases

import (
	"context"
	"fmt"

	"github.com/rohitg00/ironkube/pkg/addons"
	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/distro"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/ssh"
)

type AddonsPhase struct {
	Config *config.ClusterConfig
}

func (p *AddonsPhase) Name() string { return "addons" }

func (p *AddonsPhase) Run(ctx context.Context, state *engine.State) error {
	v, ok := state.Get("executor")
	if !ok {
		return fmt.Errorf("executor not found in state")
	}
	exec := v.(*ssh.Executor)

	v, ok = state.Get("distro_plugin")
	if !ok {
		return fmt.Errorf("distro_plugin not found in state")
	}
	plugin := v.(distro.Plugin)

	firstNode := p.Config.Spec.ControlPlane.Nodes[0]

	var toInstall []addons.Addon

	if cni := p.Config.Spec.Addons.CNI; cni != "" && cni != "flannel" {
		if addon, ok := addons.Get(cni); ok {
			toInstall = append(toInstall, addon)
		}
	}

	if p.Config.Spec.Addons.CertManager {
		if addon, ok := addons.Get("cert-manager"); ok {
			toInstall = append(toInstall, addon)
		}
	}

	if p.Config.Spec.Addons.Monitoring {
		if addon, ok := addons.Get("monitoring"); ok {
			toInstall = append(toInstall, addon)
		}
	}

	for _, addon := range toInstall {
		installCmd := withKubeconfig(addon.InstallCmd(), plugin)
		out, err := exec.RunOnHost(firstNode.Host, installCmd)
		if err != nil {
			return fmt.Errorf("failed to install addon %s: %w\nOutput: %s", addon.Name(), err, out)
		}

		if waitCmd := addon.WaitCmd(); waitCmd != "" {
			waitCmd = withKubeconfig(waitCmd, plugin)
			exec.RunOnHost(firstNode.Host, waitCmd)
		}
	}

	return nil
}

func withKubeconfig(cmd string, plugin distro.Plugin) string {
	switch plugin.Name() {
	case "k3s":
		return "KUBECONFIG=/etc/rancher/k3s/k3s.yaml " + cmd
	default:
		return "KUBECONFIG=/etc/kubernetes/admin.conf " + cmd
	}
}
