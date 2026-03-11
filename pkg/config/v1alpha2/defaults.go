package v1alpha2

func ApplyDefaults(cfg *ClusterConfig) {
	if cfg.Spec.Infrastructure.Provider == "" {
		cfg.Spec.Infrastructure.Provider = "ssh"
	}
	if cfg.Spec.Security.Profile == "" {
		cfg.Spec.Security.Profile = "minimal"
	}
	if cfg.Spec.Networking.PodCIDR == "" {
		cfg.Spec.Networking.PodCIDR = "10.42.0.0/16"
	}
	if cfg.Spec.Networking.ServiceCIDR == "" {
		cfg.Spec.Networking.ServiceCIDR = "10.43.0.0/16"
	}
	if cfg.Spec.State.Backend == "" {
		cfg.Spec.State.Backend = "local"
	}
	if cfg.Spec.Lifecycle.Upgrades.Strategy == "" {
		cfg.Spec.Lifecycle.Upgrades.Strategy = "rolling"
	}
	if cfg.Spec.Lifecycle.Upgrades.MaxUnavailable == 0 {
		cfg.Spec.Lifecycle.Upgrades.MaxUnavailable = 1
	}
	if cfg.Spec.Lifecycle.Etcd.BackupRetention == 0 {
		cfg.Spec.Lifecycle.Etcd.BackupRetention = 7
	}

	applyNodeDefaults(&cfg.Spec.ControlPlane)
	for i := range cfg.Spec.Workers {
		for j := range cfg.Spec.Workers[i].Nodes {
			applyNodeFieldDefaults(&cfg.Spec.Workers[i].Nodes[j])
		}
	}
}

func applyNodeDefaults(cp *ControlPlane) {
	for i := range cp.Nodes {
		applyNodeFieldDefaults(&cp.Nodes[i])
	}
}

func applyNodeFieldDefaults(n *Node) {
	if n.User == "" {
		n.User = "root"
	}
	if n.Port == 0 {
		n.Port = 22
	}
}
