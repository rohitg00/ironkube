package talos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type Talos struct{}

func New() *Talos {
	return &Talos{}
}

func (t *Talos) Name() string {
	return "talos"
}

func (t *Talos) ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	if !strings.HasPrefix(version, "v") {
		return fmt.Errorf("version must start with 'v': %s", version)
	}
	if len(version) < 2 {
		return fmt.Errorf("version too short: %s", version)
	}
	return nil
}

func (t *Talos) SupportedVersions() []string {
	return []string{
		"v1.7.1",
		"v1.7.0",
		"v1.6.7",
		"v1.6.6",
		"v1.6.5",
		"v1.5.5",
		"v1.5.4",
	}
}

func (t *Talos) InitControlPlane(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, token string) (*provider.BootstrapData, error) {
	endpoint := fmt.Sprintf("https://%s:6443", machine.Spec.Host)

	genCmd := fmt.Sprintf("talosctl gen config %s %s --output-dir /tmp/talos-config", cluster.Name, endpoint)

	applyCmd := fmt.Sprintf("talosctl apply-config --insecure --nodes %s --file /tmp/talos-config/controlplane.yaml", machine.Spec.Host)

	bootstrapCmd := fmt.Sprintf("talosctl bootstrap --nodes %s --endpoints %s", machine.Spec.Host, machine.Spec.Host)

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "talos-gen-config",
				Content: genCmd,
				RunAs:   "root",
			},
			{
				Name:    "talos-apply-config",
				Content: applyCmd,
				RunAs:   "root",
			},
			{
				Name:    "talos-bootstrap",
				Content: bootstrapCmd,
				RunAs:   "root",
			},
		},
		WaitCheck: &provider.WaitCondition{
			Command:  fmt.Sprintf("talosctl health --nodes %s", machine.Spec.Host),
			Expected: "OK",
			Interval: 15 * time.Second,
			Timeout:  5 * time.Minute,
		},
	}, nil
}

func (t *Talos) JoinControlPlane(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, joinEndpoint, token string) (*provider.BootstrapData, error) {
	applyCmd := fmt.Sprintf("talosctl apply-config --insecure --nodes %s --file /tmp/talos-config/controlplane.yaml", machine.Spec.Host)

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "talos-join-controlplane",
				Content: applyCmd,
				RunAs:   "root",
			},
		},
		WaitCheck: &provider.WaitCondition{
			Command:  fmt.Sprintf("talosctl health --nodes %s", machine.Spec.Host),
			Expected: "OK",
			Interval: 15 * time.Second,
			Timeout:  5 * time.Minute,
		},
	}, nil
}

func (t *Talos) JoinWorker(ctx context.Context, cluster *provider.ClusterSpec, machine provider.Machine, joinEndpoint, token string) (*provider.BootstrapData, error) {
	applyCmd := fmt.Sprintf("talosctl apply-config --insecure --nodes %s --file /tmp/talos-config/worker.yaml", machine.Spec.Host)

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "talos-join-worker",
				Content: applyCmd,
				RunAs:   "root",
			},
		},
		WaitCheck: &provider.WaitCondition{
			Command:  fmt.Sprintf("talosctl health --nodes %s", machine.Spec.Host),
			Expected: "OK",
			Interval: 15 * time.Second,
			Timeout:  5 * time.Minute,
		},
	}, nil
}

func (t *Talos) Uninstall(ctx context.Context, machine provider.Machine, role string) (*provider.BootstrapData, error) {
	resetCmd := fmt.Sprintf("talosctl reset --nodes %s --graceful=false", machine.Spec.Host)

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "talos-reset",
				Content: resetCmd,
				RunAs:   "root",
			},
		},
	}, nil
}

func (t *Talos) Upgrade(ctx context.Context, machine provider.Machine, toVersion string) (*provider.BootstrapData, error) {
	upgradeCmd := fmt.Sprintf("talosctl upgrade --nodes %s --image ghcr.io/siderolabs/installer:%s", machine.Spec.Host, toVersion)

	return &provider.BootstrapData{
		Scripts: []provider.Script{
			{
				Name:    "talos-upgrade",
				Content: upgradeCmd,
				RunAs:   "root",
			},
		},
	}, nil
}

func (t *Talos) SecurityArgs(profile provider.SecurityProfile) []string {
	return nil
}

func (t *Talos) KubeconfigPath() string {
	return ""
}

func (t *Talos) TokenCommand() string {
	return ""
}
