package config

import (
	"fmt"
	"os"

	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
)

func FromV1Alpha2(v2 *v1alpha2.ClusterConfig) *ClusterConfig {
	warnDroppedFields(v2)

	cfg := &ClusterConfig{
		APIVersion: "ironkube.dev/v1alpha1",
		Kind:       "Cluster",
		Metadata: Metadata{
			Name: v2.Metadata.Name,
		},
		Spec: ClusterSpec{
			Distro:  v2.Spec.Distro,
			Version: v2.Spec.Version,
			Security: Security{
				Profile: v2.Spec.Security.Profile,
			},
			Networking: Networking{
				CNI:     defaultCNI(v2.Spec.Distro),
				PodCIDR: v2.Spec.Networking.PodCIDR,
				SvcCIDR: v2.Spec.Networking.ServiceCIDR,
			},
		},
	}

	globalKeyPath := ""
	if v2.Spec.Infrastructure.SSH != nil {
		globalKeyPath = v2.Spec.Infrastructure.SSH.KeyPath
	}

	cfg.Spec.ControlPlane = ControlPlane{
		Replicas: v2.Spec.ControlPlane.Replicas,
		Nodes:    convertNodes(v2.Spec.ControlPlane.Nodes, globalKeyPath),
	}

	cfg.Spec.Workers = make([]NodePool, len(v2.Spec.Workers))
	for i, wp := range v2.Spec.Workers {
		cfg.Spec.Workers[i] = NodePool{
			Name:  wp.Name,
			Nodes: convertNodes(wp.Nodes, globalKeyPath),
		}
	}

	return cfg
}

func defaultCNI(distro string) string {
	if distro == "k3s" {
		return "flannel"
	}
	return "calico"
}

func warnDroppedFields(v2 *v1alpha2.ClusterConfig) {
	var dropped []string

	if len(v2.Spec.Addons) > 0 {
		dropped = append(dropped, "spec.addons")
	}
	if v2.Spec.Lifecycle.Certs.AutoRotate || v2.Spec.Lifecycle.Etcd.BackupSchedule != "" || v2.Spec.Lifecycle.Upgrades.Strategy != "" {
		dropped = append(dropped, "spec.lifecycle")
	}
	if v2.Spec.State.Backend != "" {
		dropped = append(dropped, "spec.state")
	}
	if v2.Spec.Infrastructure.Provider != "" && v2.Spec.Infrastructure.Provider != "ssh" {
		dropped = append(dropped, fmt.Sprintf("spec.infrastructure.provider=%s", v2.Spec.Infrastructure.Provider))
	}
	for _, wp := range v2.Spec.Workers {
		if len(wp.Labels) > 0 || len(wp.Taints) > 0 || wp.Count > 0 {
			dropped = append(dropped, fmt.Sprintf("spec.workers[%s].labels/taints/count", wp.Name))
		}
	}

	if len(dropped) > 0 {
		for _, field := range dropped {
			fmt.Fprintf(os.Stderr, "warning: %s dropped during bootstrap conversion\n", field)
		}
	}
}

func convertNodes(v2Nodes []v1alpha2.Node, globalKeyPath string) []Node {
	nodes := make([]Node, len(v2Nodes))
	for i, n := range v2Nodes {
		keyPath := n.KeyPath
		if keyPath == "" {
			keyPath = globalKeyPath
		}
		nodes[i] = Node{
			Host:    n.Host,
			User:    n.User,
			Port:    n.Port,
			KeyPath: keyPath,
		}
	}
	return nodes
}
