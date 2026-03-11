package v1alpha2

type FleetConfig struct {
	APIVersion string    `yaml:"apiVersion"`
	Kind       string    `yaml:"kind"`
	Metadata   Metadata  `yaml:"metadata"`
	Spec       FleetSpec `yaml:"spec"`
}

type FleetSpec struct {
	Clusters []ClusterRef  `yaml:"clusters"`
	State    StateConfig   `yaml:"state,omitempty"`
	Policies FleetPolicies `yaml:"policies,omitempty"`
}

type ClusterRef struct {
	Path   string            `yaml:"path"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

type FleetPolicies struct {
	UpgradeWindow         string `yaml:"upgradeWindow,omitempty"`
	MaxConcurrentUpgrades int    `yaml:"maxConcurrentUpgrades,omitempty"`
}
