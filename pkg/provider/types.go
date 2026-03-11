package provider

import "time"

type MachineSpec struct {
	Role    string            `yaml:"role" json:"role"`
	Host    string            `yaml:"host" json:"host"`
	User    string            `yaml:"user" json:"user"`
	Port    int               `yaml:"port" json:"port"`
	KeyPath string            `yaml:"keyPath" json:"keyPath"`
	Labels  map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Taints  []Taint           `yaml:"taints,omitempty" json:"taints,omitempty"`
}

type Taint struct {
	Key    string `yaml:"key" json:"key"`
	Value  string `yaml:"value,omitempty" json:"value,omitempty"`
	Effect string `yaml:"effect" json:"effect"`
}

type Machine struct {
	ID         string        `json:"id"`
	ProviderID string        `json:"providerId,omitempty"`
	Spec       MachineSpec   `json:"spec"`
	Status     MachineStatus `json:"status"`
	CreatedAt  time.Time     `json:"createdAt"`
}

type MachineStatus string

const (
	MachineStatusProvisioning MachineStatus = "provisioning"
	MachineStatusReady        MachineStatus = "ready"
	MachineStatusUpgrading    MachineStatus = "upgrading"
	MachineStatusDraining     MachineStatus = "draining"
	MachineStatusDeleted      MachineStatus = "deleted"
)

type Script struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	RunAs   string `json:"runAs,omitempty"`
}

type FileToWrite struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    uint32 `json:"mode"`
}

type WaitCondition struct {
	Command  string        `json:"command"`
	Expected string        `json:"expected"`
	Interval time.Duration `json:"interval"`
	Timeout  time.Duration `json:"timeout"`
}

type BootstrapData struct {
	Scripts   []Script          `json:"scripts"`
	Files     []FileToWrite     `json:"files,omitempty"`
	EnvVars   map[string]string `json:"envVars,omitempty"`
	WaitCheck *WaitCondition    `json:"waitCheck,omitempty"`
}

type NodeHealth struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Version string `json:"version"`
	Roles   string `json:"roles"`
}

type CertInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	ExpiresAt time.Time `json:"expiresAt"`
	Issuer    string    `json:"issuer"`
}

type EtcdMember struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	IsLeader bool   `json:"isLeader"`
	DbSize   int64  `json:"dbSize"`
}

type EtcdHealth struct {
	Healthy bool         `json:"healthy"`
	Members []EtcdMember `json:"members"`
	DbSize  int64        `json:"dbSize"`
	Version string       `json:"version"`
}

type AddonStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Ready     bool   `json:"ready"`
}

type HealthReport struct {
	ClusterName string        `json:"clusterName"`
	Nodes       []NodeHealth  `json:"nodes"`
	Certs       []CertInfo    `json:"certs,omitempty"`
	Etcd        *EtcdHealth   `json:"etcd,omitempty"`
	Addons      []AddonStatus `json:"addons,omitempty"`
}

func (r *HealthReport) AllHealthy() bool {
	for _, n := range r.Nodes {
		if !n.Ready {
			return false
		}
	}
	return true
}

func (r *HealthReport) ReadyCount() int {
	count := 0
	for _, n := range r.Nodes {
		if n.Ready {
			count++
		}
	}
	return count
}

func (r *HealthReport) TotalCount() int {
	return len(r.Nodes)
}

type DrainOptions struct {
	GracePeriod      int
	Timeout          time.Duration
	IgnoreDaemonSets bool
	DeleteLocalData  bool
	Force            bool
}

type AddonSpec struct {
	Name        string         `yaml:"name" json:"name"`
	Type        string         `yaml:"type" json:"type"`
	Repository  string         `yaml:"repository,omitempty" json:"repository,omitempty"`
	Chart       string         `yaml:"chart,omitempty" json:"chart,omitempty"`
	Version     string         `yaml:"version" json:"version"`
	Namespace   string         `yaml:"namespace" json:"namespace"`
	Values      map[string]any `yaml:"values,omitempty" json:"values,omitempty"`
	ManifestURL string         `yaml:"manifestURL,omitempty" json:"manifestURL,omitempty"`
	WaitReady   bool           `yaml:"waitReady" json:"waitReady"`
}

type ClusterSpec struct {
	Name        string
	Distro      string
	Version     string
	PodCIDR     string
	ServiceCIDR string
	HAEnabled   bool
	FirstCPHost string
}

type SecurityProfile interface {
	Name() string
	APIServerFlags() []string
	KubeletFlags() []string
	EtcdFlags() []string
}
