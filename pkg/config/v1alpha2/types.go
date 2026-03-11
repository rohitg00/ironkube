package v1alpha2

type ClusterConfig struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   Metadata    `yaml:"metadata"`
	Spec       ClusterSpec `yaml:"spec"`
}

type Metadata struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

type ClusterSpec struct {
	Distro         string         `yaml:"distro"`
	Version        string         `yaml:"version"`
	Infrastructure Infrastructure `yaml:"infrastructure"`
	ControlPlane   ControlPlane   `yaml:"controlPlane"`
	Workers        []NodePool     `yaml:"workers,omitempty"`
	Networking     Networking     `yaml:"networking"`
	Security       Security       `yaml:"security"`
	Addons         []AddonConfig  `yaml:"addons,omitempty"`
	Lifecycle      Lifecycle      `yaml:"lifecycle,omitempty"`
	State          StateConfig    `yaml:"state,omitempty"`
}

type Infrastructure struct {
	Provider string         `yaml:"provider"`
	SSH      *SSHConfig     `yaml:"ssh,omitempty"`
	Hetzner  *HetznerConfig `yaml:"hetzner,omitempty"`
	Docker   *DockerConfig  `yaml:"docker,omitempty"`
}

type SSHConfig struct {
	KeyPath string `yaml:"keyPath,omitempty"`
}

type HetznerConfig struct {
	Token      string `yaml:"token,omitempty"`
	TokenEnv   string `yaml:"tokenEnv,omitempty"`
	Location   string `yaml:"location,omitempty"`
	ServerType string `yaml:"serverType,omitempty"`
	Image      string `yaml:"image,omitempty"`
	SSHKeyName string `yaml:"sshKeyName,omitempty"`
}

type DockerConfig struct {
	Image   string `yaml:"image,omitempty"`
	Network string `yaml:"network,omitempty"`
}

type ControlPlane struct {
	Replicas int    `yaml:"replicas"`
	Nodes    []Node `yaml:"nodes,omitempty"`
}

type Node struct {
	Host    string `yaml:"host,omitempty"`
	User    string `yaml:"user,omitempty"`
	Port    int    `yaml:"port,omitempty"`
	KeyPath string `yaml:"keyPath,omitempty"`
}

type NodePool struct {
	Name   string            `yaml:"name"`
	Nodes  []Node            `yaml:"nodes,omitempty"`
	Count  int               `yaml:"count,omitempty"`
	Labels map[string]string `yaml:"labels,omitempty"`
	Taints []TaintConfig     `yaml:"taints,omitempty"`
}

type TaintConfig struct {
	Key    string `yaml:"key"`
	Value  string `yaml:"value,omitempty"`
	Effect string `yaml:"effect"`
}

type Networking struct {
	PodCIDR     string `yaml:"podCIDR,omitempty"`
	ServiceCIDR string `yaml:"serviceCIDR,omitempty"`
}

type Security struct {
	Profile          string `yaml:"profile"`
	EncryptionAtRest bool   `yaml:"encryptionAtRest,omitempty"`
	AuditLogging     bool   `yaml:"auditLogging,omitempty"`
}

type AddonConfig struct {
	Name       string         `yaml:"name"`
	Enabled    bool           `yaml:"enabled"`
	Version    string         `yaml:"version,omitempty"`
	Namespace  string         `yaml:"namespace,omitempty"`
	Repository string         `yaml:"repository,omitempty"`
	Chart      string         `yaml:"chart,omitempty"`
	Values     map[string]any `yaml:"values,omitempty"`
}

type Lifecycle struct {
	Certs    CertsConfig   `yaml:"certs,omitempty"`
	Etcd     EtcdConfig    `yaml:"etcd,omitempty"`
	Upgrades UpgradeConfig `yaml:"upgrades,omitempty"`
}

type CertsConfig struct {
	AutoRotate         bool   `yaml:"autoRotate,omitempty"`
	RotateBeforeExpiry string `yaml:"rotateBeforeExpiry,omitempty"`
}

type EtcdConfig struct {
	BackupSchedule     string `yaml:"backupSchedule,omitempty"`
	BackupRetention    int    `yaml:"backupRetention,omitempty"`
	CompactionInterval string `yaml:"compactionInterval,omitempty"`
}

type UpgradeConfig struct {
	Strategy       string `yaml:"strategy,omitempty"`
	MaxUnavailable int    `yaml:"maxUnavailable,omitempty"`
}

type StateConfig struct {
	Backend string     `yaml:"backend,omitempty"`
	Git     *GitConfig `yaml:"git,omitempty"`
	S3      *S3Config  `yaml:"s3,omitempty"`
}

type GitConfig struct {
	Repo   string `yaml:"repo"`
	Branch string `yaml:"branch,omitempty"`
}

type S3Config struct {
	Bucket string `yaml:"bucket"`
	Region string `yaml:"region"`
	Prefix string `yaml:"prefix,omitempty"`
}
