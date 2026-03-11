package config

func ApplyDefaults(cfg *ClusterConfig) {
	applyNodeDefaults(&cfg.Spec.ControlPlane.Nodes)
	for i := range cfg.Spec.Workers {
		applyNodeDefaults(&cfg.Spec.Workers[i].Nodes)
	}

	if cfg.Spec.Security.Profile == "" {
		cfg.Spec.Security.Profile = "minimal"
	}

	if cfg.Spec.Networking.CNI == "" {
		cfg.Spec.Networking.CNI = "flannel"
	}

	if cfg.Spec.Networking.PodCIDR == "" {
		cfg.Spec.Networking.PodCIDR = "10.42.0.0/16"
	}

	if cfg.Spec.Networking.SvcCIDR == "" {
		cfg.Spec.Networking.SvcCIDR = "10.43.0.0/16"
	}
}

func applyNodeDefaults(nodes *[]Node) {
	for i := range *nodes {
		if (*nodes)[i].User == "" {
			(*nodes)[i].User = "root"
		}
		if (*nodes)[i].Port == 0 {
			(*nodes)[i].Port = 22
		}
	}
}
