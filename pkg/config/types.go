package config

type ClusterConfig struct {
	APIVersion string      `yaml:"apiVersion" json:"apiVersion"`
	Kind       string      `yaml:"kind" json:"kind"`
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Spec       ClusterSpec `yaml:"spec" json:"spec"`
}

type Metadata struct {
	Name string `yaml:"name" json:"name"`
}

type ClusterSpec struct {
	Distro       string       `yaml:"distro" json:"distro"`
	Version      string       `yaml:"version" json:"version"`
	ControlPlane ControlPlane `yaml:"controlPlane" json:"controlPlane"`
	Workers      []NodePool   `yaml:"workers" json:"workers"`
	Security     Security     `yaml:"security" json:"security"`
	Networking   Networking   `yaml:"networking" json:"networking"`
}

type ControlPlane struct {
	Replicas int    `yaml:"replicas" json:"replicas"`
	Nodes    []Node `yaml:"nodes" json:"nodes"`
}

type NodePool struct {
	Name  string `yaml:"name" json:"name"`
	Nodes []Node `yaml:"nodes" json:"nodes"`
}

type Node struct {
	Host    string `yaml:"host" json:"host"`
	User    string `yaml:"user" json:"user"`
	Port    int    `yaml:"port" json:"port"`
	KeyPath string `yaml:"keyPath" json:"keyPath"`
}

type Security struct {
	Profile string `yaml:"profile" json:"profile"`
}

type Networking struct {
	CNI     string `yaml:"cni" json:"cni"`
	PodCIDR string `yaml:"podCIDR" json:"podCIDR"`
	SvcCIDR string `yaml:"serviceCIDR" json:"serviceCIDR"`
}

type SecurityFlags struct {
	APIServer []string
	Kubelet   []string
	Etcd      []string
}

func (sf SecurityFlags) IsEmpty() bool {
	return len(sf.APIServer) == 0 && len(sf.Kubelet) == 0 && len(sf.Etcd) == 0
}
