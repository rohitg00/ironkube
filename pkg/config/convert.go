package config

import (
	"github.com/rohitg00/ironkube/pkg/config/v1alpha2"
)

func FromV1Alpha2(v2 *v1alpha2.ClusterConfig) *ClusterConfig {
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
